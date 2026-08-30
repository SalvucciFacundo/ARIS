package services_test

import (
	"bytes"
	"context"
	img "image"
	imgColor "image/color"
	imgPNG "image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"aris/internal/adapters/db"
	"aris/internal/adapters/image"
	"aris/internal/adapters/llm"
	"aris/internal/core/domain"
	"aris/internal/core/services"
	"aris/pkg/imgutil"
)

func TestAgentService_GenerateWorkflow(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "aris-agent-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	sqlDB, err := db.NewSQLiteDB(dbPath)
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}
	defer sqlDB.Close()

	kg := db.NewKnowledgeGraph(sqlDB.DB())
	history := db.NewHistoryStore(sqlDB.DB())
	llmProvider := llm.NewPassthroughProvider()

	reg := image.NewRegistry()
	mockBackend := image.NewPollinationsBackend(
		image.WithOutputDir(tmpDir),
	)
	_ = reg.Register(mockBackend)

	agent := services.NewAgentService(llmProvider, reg, kg, history, nil)
	ctx := context.Background()

	// 1. Add style memory fact
	_, err = agent.LearnFact(ctx, "style:anime", "lineart", "clean thick anime lineart, studio ghibli aesthetic", domain.ScopeStyle, []string{"anime", "ghibli"})
	if err != nil {
		t.Fatalf("LearnFact failed: %v", err)
	}

	// 2. Search Memory
	facts, err := agent.SearchMemory(ctx, "anime", domain.ScopeStyle, 5)
	if err != nil {
		t.Fatalf("SearchMemory failed: %v", err)
	}
	if len(facts) == 0 {
		t.Fatalf("expected to find at least 1 fact")
	}

	// 3. Test Reasoning
	spec, err := llmProvider.ReasonPrompt(ctx, "a peaceful floating island in the sky", facts)
	if err != nil {
		t.Fatalf("ReasonPrompt failed: %v", err)
	}
	if spec.RawPrompt != "a peaceful floating island in the sky" {
		t.Errorf("unexpected raw prompt: %s", spec.RawPrompt)
	}

	// Save generation to history
	res := &domain.ImageResult{
		ID:          "gen-mock-1",
		SpecID:      spec.ID,
		LocalPath:   filepath.Join(tmpDir, "output.png"),
		Format:      "png",
		SizeInBytes: 512,
		Duration:    100 * time.Millisecond,
	}
	_ = history.SaveGeneration(ctx, spec, res)

	historyList, err := agent.GetHistory(ctx, 10, 0)
	if err != nil {
		t.Fatalf("GetHistory failed: %v", err)
	}
	if len(historyList) != 1 {
		t.Fatalf("expected 1 history entry, got %d", len(historyList))
	}

	// 4. Test Subagent Manager Integration on AgentService
	subStore := db.NewSubagentStore(sqlDB.DB())
	subMgr := services.NewSubagentManager(subStore, llmProvider, reg, nil, kg)
	agent.SetSubagents(subMgr)

	if agent.Subagents() == nil {
		t.Fatal("expected subagents manager to be set")
	}

	resp, err := agent.ExecuteSubagent(ctx, "director", "cinematic cyberpunk cityscape")
	if err != nil {
		t.Fatalf("ExecuteSubagent failed: %v", err)
	}
	if resp == "" {
		t.Error("expected non-empty subagent response")
	}

	pipeRes, err := agent.PipelineGenerate(ctx, "cyberpunk alley", services.PipelineOptions{
		AspectRatio: domain.RatioLandscape,
	})
	if err != nil {
		t.Fatalf("PipelineGenerate failed: %v", err)
	}
	if pipeRes.DirectorConcept == "" {
		t.Error("expected director concept in pipeline result")
	}
}

func TestAgentService_GenerateImg2ImgAndInpaintOptions(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	sqlDB, _ := db.NewSQLiteDB(dbPath)
	defer sqlDB.Close()

	llmProvider := llm.NewPassthroughProvider()
	reg := image.NewRegistry()
	mockBackend := &MockBackend{
		name: "mock-backend",
		generateFunc: func(ctx context.Context, spec *domain.ImageSpec) (*domain.ImageResult, error) {
			return &domain.ImageResult{
				ID:        "res-1",
				SpecID:    spec.ID,
				LocalPath: "/tmp/out.png",
			}, nil
		},
	}
	_ = reg.Register(mockBackend)
	_ = reg.SetDefault("mock-backend")

	agent := services.NewAgentService(llmProvider, reg, nil, nil, nil)
	ctx := context.Background()

	// Img2Img generation with reference image and strength
	optsI2I := services.GenerateOptions{
		InputImage:      "/tmp/base.png",
		DenoiseStrength: 0.65,
		Mode:            domain.ModeImg2Img,
	}
	specI2I, _, err := agent.Generate(ctx, "turn to cyberpunk", optsI2I)
	if err != nil {
		t.Fatalf("Generate img2img failed: %v", err)
	}
	if specI2I.InputImagePath != "/tmp/base.png" {
		t.Errorf("expected input image path to propagate, got %s", specI2I.InputImagePath)
	}
	if specI2I.DenoiseStrength != 0.65 {
		t.Errorf("expected denoise strength 0.65, got %f", specI2I.DenoiseStrength)
	}
	if specI2I.Mode != domain.ModeImg2Img {
		t.Errorf("expected ModeImg2Img, got %s", specI2I.Mode)
	}

	// Inpaint generation with mask
	optsInpaint := services.GenerateOptions{
		InputImage:      "/tmp/base.png",
		MaskImage:       "/tmp/mask.png",
		DenoiseStrength: 0.85,
		Mode:            domain.ModeInpaint,
	}
	specInp, _, err := agent.Generate(ctx, "remove glasses", optsInpaint)
	if err != nil {
		t.Fatalf("Generate inpaint failed: %v", err)
	}
	if specInp.MaskImagePath != "/tmp/mask.png" {
		t.Errorf("expected mask image path to propagate, got %s", specInp.MaskImagePath)
	}
	if specInp.DenoiseStrength != 0.85 {
		t.Errorf("expected denoise 0.85, got %f", specInp.DenoiseStrength)
	}
	if specInp.Mode != domain.ModeInpaint {
		t.Errorf("expected ModeInpaint, got %s", specInp.Mode)
	}
}

func TestSubagentManager_VisualSubagents(t *testing.T) {
	ctx := context.Background()
	store := NewMockSubagentStore()
	llmProvider := &MockLLMProvider{}
	mgr := services.NewSubagentManager(store, llmProvider, nil, nil, nil)

	// Verify @upscaler preset exists
	upscaler, err := mgr.GetSubagent(ctx, "upscaler")
	if err != nil {
		t.Fatalf("expected @upscaler subagent to exist: %v", err)
	}
	if upscaler.Name != "upscaler" {
		t.Errorf("expected name upscaler, got %s", upscaler.Name)
	}
	if upscaler.Role != "Image Enhancement & Artifact Restoration" && upscaler.Role != "Super-Resolution & Face Restoration Specialist" {
		t.Errorf("unexpected role for @upscaler: %s", upscaler.Role)
	}

	// Verify @inpainter preset exists
	inpainter, err := mgr.GetSubagent(ctx, "inpainter")
	if err != nil {
		t.Fatalf("expected @inpainter subagent to exist: %v", err)
	}
	if inpainter.Name != "inpainter" {
		t.Errorf("expected name inpainter, got %s", inpainter.Name)
	}

	// Verify @restyler preset exists
	restyler, err := mgr.GetSubagent(ctx, "restyler")
	if err != nil {
		t.Fatalf("expected @restyler subagent to exist: %v", err)
	}
	if restyler.Name != "restyler" {
		t.Errorf("expected name restyler, got %s", restyler.Name)
	}
}

func TestAgentService_GenerateUpscaleOptions(t *testing.T) {
	llmProvider := llm.NewPassthroughProvider()
	reg := image.NewRegistry()
	mockBackend := &MockBackend{
		name: "mock-backend",
		generateFunc: func(ctx context.Context, spec *domain.ImageSpec) (*domain.ImageResult, error) {
			return &domain.ImageResult{
				ID:        "res-upscale-1",
				SpecID:    spec.ID,
				LocalPath: "/tmp/upscaled.png",
			}, nil
		},
	}
	_ = reg.Register(mockBackend)
	_ = reg.SetDefault("mock-backend")

	agent := services.NewAgentService(llmProvider, reg, nil, nil, nil)
	ctx := context.Background()

	optsUpscale := services.GenerateOptions{
		InputImage:    "/tmp/portrait.png",
		Mode:          domain.ModeUpscale,
		ScaleFactor:   4,
		RestoreFaces:  true,
		FaceFidelity:  0.80,
		UpscalerModel: "RealESRGAN_x4plus",
	}

	spec, res, err := agent.Generate(ctx, "upscale portrait to 4k with face restoration", optsUpscale)
	if err != nil {
		t.Fatalf("Generate upscale failed: %v", err)
	}
	if res == nil {
		t.Fatal("expected non-nil result")
	}
	if !spec.IsUpscale() {
		t.Errorf("expected IsUpscale to be true")
	}
	if spec.ScaleFactor != 4 {
		t.Errorf("expected ScaleFactor 4, got %d", spec.ScaleFactor)
	}
	if !spec.RestoreFaces {
		t.Errorf("expected RestoreFaces true")
	}
	if spec.FaceFidelity != 0.80 {
		t.Errorf("expected FaceFidelity 0.80, got %f", spec.FaceFidelity)
	}
	if spec.UpscalerModel != "RealESRGAN_x4plus" {
		t.Errorf("expected UpscalerModel RealESRGAN_x4plus, got %s", spec.UpscalerModel)
	}
}

type mockPNGBackend struct {
	outputDir string
}

func (m *mockPNGBackend) Name() string { return "mock_png" }
func (m *mockPNGBackend) SupportsModels() []string { return []string{"mock-model"} }
func (m *mockPNGBackend) Generate(ctx context.Context, spec *domain.ImageSpec) (*domain.ImageResult, error) {
	rawImg := img.NewRGBA(img.Rect(0, 0, 4, 4))
	rawImg.Set(0, 0, imgColor.RGBA{R: 200, G: 100, B: 50, A: 255})
	var buf bytes.Buffer
	_ = imgPNG.Encode(&buf, rawImg)

	outPath := filepath.Join(m.outputDir, "mock_output.png")
	if err := os.WriteFile(outPath, buf.Bytes(), 0644); err != nil {
		return nil, err
	}

	return &domain.ImageResult{
		ID:        "mock-res-1",
		SpecID:    spec.ID,
		LocalPath: outPath,
		Format:    "png",
		Metadata:  map[string]any{},
	}, nil
}

func TestAgentService_EmbedsGenerationParametersMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	sqlDB, err := db.NewSQLiteDB(dbPath)
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}
	defer sqlDB.Close()

	kg := db.NewKnowledgeGraph(sqlDB.DB())
	history := db.NewHistoryStore(sqlDB.DB())
	llmProvider := llm.NewPassthroughProvider()

	reg := image.NewRegistry()
	mockBackend := &mockPNGBackend{outputDir: tmpDir}
	_ = reg.Register(mockBackend)
	_ = reg.SetDefault("mock_png")

	agent := services.NewAgentService(llmProvider, reg, kg, history, nil)
	ctx := context.Background()

	spec, res, err := agent.Generate(ctx, "a futuristic neo-tokyo street at night", services.GenerateOptions{})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if spec == nil {
		t.Fatalf("expected non-nil spec")
	}

	if res == nil || res.LocalPath == "" {
		t.Fatalf("expected valid image result")
	}

	meta, err := imgutil.ExtractPNGMetadataFile(res.LocalPath)
	if err != nil {
		t.Fatalf("ExtractPNGMetadataFile failed: %v", err)
	}

	paramsStr, ok := meta["parameters"]
	if !ok || paramsStr == "" {
		t.Fatalf("expected 'parameters' metadata embedded in generated PNG, got: %+v", meta)
	}

	if !strings.Contains(paramsStr, "neo-tokyo") {
		t.Errorf("expected parameters to contain prompt text, got: %s", paramsStr)
	}
	if !strings.Contains(paramsStr, "Steps:") {
		t.Errorf("expected parameters to contain Steps, got: %s", paramsStr)
	}
	if !strings.Contains(paramsStr, "CFG scale:") {
		t.Errorf("expected parameters to contain CFG scale, got: %s", paramsStr)
	}
}

