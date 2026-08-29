package web

import (
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"aris/internal/adapters/ui/web/views"
	"aris/internal/core/domain"
	"aris/internal/core/services"
)

// Handlers handles all HTTP REST endpoints.
type Handlers struct {
	agent  *services.AgentService
	broker *SSEBroker
	cfg    Config
}

// NewHandlers creates a new Handlers instance.
func NewHandlers(agent *services.AgentService, broker *SSEBroker, cfg Config) *Handlers {
	return &Handlers{
		agent:  agent,
		broker: broker,
		cfg:    cfg,
	}
}

// GenerateRequest represents the JSON payload for /api/generate.
type GenerateRequest struct {
	Prompt           string  `json:"prompt"`
	NegativePrompt   string  `json:"negative_prompt,omitempty"`
	Subagent         string  `json:"subagent,omitempty"`
	Backend          string  `json:"backend,omitempty"`
	Model            string  `json:"model,omitempty"`
	AspectRatio      string  `json:"aspect_ratio,omitempty"`
	Width            int     `json:"width,omitempty"`
	Height           int     `json:"height,omitempty"`
	Steps            int     `json:"steps,omitempty"`
	CFGScale         float64 `json:"cfg_scale,omitempty"`
	Seed             int64   `json:"seed,omitempty"`
	EnableCritic     bool    `json:"enable_critic,omitempty"`
	CriticMaxRetries int     `json:"critic_max_retries,omitempty"`
}

// HandleIndex renders the main application shell.
func (h *Handlers) HandleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = views.Layout("ARIS Web Interface").Render(r.Context(), w)
}

// HandleGenerate processes generation requests and streams events via SSE.
func (h *Handlers) HandleGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req GenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request payload: " + err.Error()})
		return
	}

	if strings.TrimSpace(req.Prompt) == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Prompt cannot be empty"})
		return
	}

	jobID := fmt.Sprintf("job-%d", os.Getpid())

	// Emit initial SSE progress event
	if h.broker != nil {
		h.broker.Broadcast(SSEEvent{
			Event: "progress",
			Data: map[string]any{
				"job_id":  jobID,
				"stage":   "reasoning",
				"percent": 10,
				"message": "Reasoning prompt architecture...",
			},
		})
	}

	if h.agent == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"job_id":  jobID,
			"status":  "mock_accepted",
			"message": "Agent service not connected (mock mode)",
		})
		return
	}

	// Run generation asynchronously or inline
	opts := services.GenerateOptions{
		AspectRatio:    domain.ParseAspectRatio(req.AspectRatio),
		Model:          req.Model,
		Backend:        req.Backend,
		Seed:           req.Seed,
		NegativePrompt: req.NegativePrompt,
		EnableCritic:   req.EnableCritic,
	}

	go func() {
		ctx := r.Context()
		if subName, cleanPrompt, isSub := services.ParseSubagentRoute(req.Prompt); isSub || req.Subagent != "" {
			targetSub := req.Subagent
			if targetSub == "" {
				targetSub = subName
			}
			if targetSub != "" && cleanPrompt != "" {
				if h.broker != nil {
					h.broker.Broadcast(SSEEvent{
						Event: "reasoning",
						Data: map[string]any{
							"job_id":   jobID,
							"subagent": targetSub,
							"chunk":    fmt.Sprintf("Activating @%s reasoning...", targetSub),
						},
					})
				}
			}
		}

		spec, result, err := h.agent.Generate(ctx, req.Prompt, opts)
		if err != nil {
			if h.broker != nil {
				h.broker.Broadcast(SSEEvent{
					Event: "error",
					Data: map[string]any{
						"job_id":  jobID,
						"code":    "GENERATION_FAILED",
						"message": err.Error(),
					},
				})
			}
			return
		}

		if h.broker != nil {
			imgID := filepath.Base(result.LocalPath)
			h.broker.Broadcast(SSEEvent{
				Event: "image_ready",
				Data: map[string]any{
					"job_id":       jobID,
					"image_id":     imgID,
					"url":          "/api/image/" + imgID,
					"prompt":       spec.EnhancedPrompt,
					"aspect_ratio": string(spec.AspectRatio),
					"seed":         spec.Seed,
				},
			})

			if result.Metadata != nil {
				if score, ok := result.Metadata["critic_score"].(float64); ok {
					critique, _ := result.Metadata["critic_notes"].(string)
					h.broker.Broadcast(SSEEvent{
						Event: "critic_evaluation",
						Data: map[string]any{
							"job_id":   jobID,
							"image_id": imgID,
							"score":    score,
							"critique": critique,
							"passed":   score >= 0.60,
						},
					})
				}
			}
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"job_id":  jobID,
		"status":  "processing",
		"message": "Generation job dispatched",
	})
}

// HandleInpaint processes multipart inpainting requests.
func (h *Handlers) HandleInpaint(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseMultipartForm(32 << 20) // 32MB max
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Invalid multipart form: " + err.Error()})
		return
	}

	prompt := r.FormValue("prompt")
	if strings.TrimSpace(prompt) == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Prompt cannot be empty"})
		return
	}

	backend := r.FormValue("backend")
	model := r.FormValue("model")
	denoiseStr := r.FormValue("denoising_strength")
	denoise := 0.70
	if denoiseStr != "" {
		if d, err := strconv.ParseFloat(denoiseStr, 64); err == nil {
			denoise = d
		}
	}

	// Check image and mask files
	imageFile, imageHeader, err := r.FormFile("image")
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Base image is required"})
		return
	}
	defer imageFile.Close()

	maskFile, _, _ := r.FormFile("mask")
	if maskFile != nil {
		defer maskFile.Close()
	}

	// Save temporary files
	tmpDir := os.TempDir()
	inputPath := filepath.Join(tmpDir, fmt.Sprintf("aris_in_%d_%s", os.Getpid(), imageHeader.Filename))
	dst, err := os.Create(inputPath)
	if err == nil {
		_, _ = io.Copy(dst, imageFile)
		dst.Close()
	}

	maskPath := ""
	if maskFile != nil {
		maskPath = filepath.Join(tmpDir, fmt.Sprintf("aris_mask_%d.png", os.Getpid()))
		mDst, err := os.Create(maskPath)
		if err == nil {
			_, _ = io.Copy(mDst, maskFile)
			mDst.Close()
		}
	}

	jobID := fmt.Sprintf("inpaint-%d", os.Getpid())

	if h.broker != nil {
		h.broker.Broadcast(SSEEvent{
			Event: "progress",
			Data: map[string]any{
				"job_id":  jobID,
				"stage":   "inpainting",
				"percent": 20,
				"message": "Dispatching inpainting transform...",
			},
		})
	}

	if h.agent != nil {
		opts := services.GenerateOptions{
			Backend:         backend,
			Model:           model,
			InputImage:      inputPath,
			MaskImage:       maskPath,
			DenoiseStrength: denoise,
			Mode:            domain.ModeInpaint,
		}
		go func() {
			spec, result, err := h.agent.Generate(r.Context(), prompt, opts)
			if err != nil {
				if h.broker != nil {
					h.broker.Broadcast(SSEEvent{
						Event: "error",
						Data: map[string]any{
							"job_id":  jobID,
							"code":    "INPAINT_FAILED",
							"message": err.Error(),
						},
					})
				}
				return
			}
			if h.broker != nil {
				imgID := filepath.Base(result.LocalPath)
				h.broker.Broadcast(SSEEvent{
					Event: "image_ready",
					Data: map[string]any{
						"job_id":       jobID,
						"image_id":     imgID,
						"url":          "/api/image/" + imgID,
						"prompt":       spec.EnhancedPrompt,
						"aspect_ratio": string(spec.AspectRatio),
						"seed":         spec.Seed,
					},
				})
			}
		}()
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"job_id":  jobID,
		"status":  "processing",
		"message": "Inpainting job accepted",
	})
}

// HandleHistory returns past generation records.
func (h *Handlers) HandleHistory(w http.ResponseWriter, r *http.Request) {
	limit := 20
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 {
			limit = val
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if val, err := strconv.Atoi(o); err == nil && val >= 0 {
			offset = val
		}
	}

	var records []domain.GenerationRecord
	if h.agent != nil {
		recs, err := h.agent.GetHistory(r.Context(), limit, offset)
		if err == nil {
			records = recs
		}
	}

	type HistoryItem struct {
		ID          string   `json:"id"`
		Prompt      string   `json:"prompt"`
		Backend     string   `json:"backend"`
		Model       string   `json:"model"`
		AspectRatio string   `json:"aspect_ratio"`
		Seed        int64    `json:"seed"`
		CriticScore float64  `json:"critic_score,omitempty"`
		CreatedAt   string   `json:"created_at"`
		ImageURL    string   `json:"image_url"`
		Width       int      `json:"width"`
		Height      int      `json:"height"`
	}

	results := make([]HistoryItem, 0, len(records))
	for _, rec := range records {
		imgID := filepath.Base(rec.ImagePath)
		ratio := "1:1"
		if rec.Width > 0 && rec.Height > 0 {
			if rec.Width == rec.Height {
				ratio = "1:1"
			} else if rec.Width > rec.Height {
				ratio = "16:9"
			} else {
				ratio = "9:16"
			}
		}
		results = append(results, HistoryItem{
			ID:          rec.ID,
			Prompt:      rec.PromptRaw,
			Backend:     rec.Backend,
			Model:       rec.Model,
			AspectRatio: ratio,
			Seed:        rec.Seed,
			CriticScore: float64(rec.Rating),
			CreatedAt:   rec.CreatedAt.Format("2006-01-02T15:04:05Z"),
			ImageURL:    "/api/image/" + imgID,
			Width:       rec.Width,
			Height:      rec.Height,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(results)
}

// HandleImage serves rendered image files from disk.
func (h *Handlers) HandleImage(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/image/")
	if id == "" || strings.Contains(id, "..") {
		http.NotFound(w, r)
		return
	}

	// Look in common output dirs or temp
	searchPaths := []string{
		id,
		filepath.Join(".", id),
		filepath.Join("output", id),
		filepath.Join(os.TempDir(), id),
	}

	var foundPath string
	for _, p := range searchPaths {
		if _, err := os.Stat(p); err == nil {
			foundPath = p
			break
		}
	}

	if foundPath == "" {
		http.NotFound(w, r)
		return
	}

	ext := filepath.Ext(foundPath)
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		mimeType = "image/png"
	}

	w.Header().Set("Content-Type", mimeType)
	w.Header().Set("Cache-Control", "public, max-age=86400")
	http.ServeFile(w, r, foundPath)
}

// HandleSubagents lists registered subagents.
func (h *Handlers) HandleSubagents(w http.ResponseWriter, r *http.Request) {
	type SubagentResponse struct {
		Name         string   `json:"name"`
		DisplayName  string   `json:"display_name"`
		Role         string   `json:"role"`
		Description  string   `json:"description"`
		Temperature  float64  `json:"temperature"`
		AllowedTools []string `json:"allowed_tools"`
	}

	var results []SubagentResponse
	if h.agent != nil && h.agent.Subagents() != nil {
		subs, err := h.agent.Subagents().ListSubagents(r.Context())
		if err == nil {
			for _, s := range subs {
				results = append(results, SubagentResponse{
					Name:         s.Name,
					DisplayName:  s.DisplayName,
					Role:         s.Role,
					Description:  s.Description,
					Temperature:  s.Temperature,
					AllowedTools: s.AllowedTools,
				})
			}
		}
	}

	if len(results) == 0 {
		// Defaults
		results = []SubagentResponse{
			{Name: "director", DisplayName: "Art Director", Role: "Composition & Lighting", Description: "Cinematography and composition reasoning"},
			{Name: "promptsmith", DisplayName: "PromptSmith", Role: "Prompt Optimization", Description: "Expands raw prompts with high-detail keywords"},
			{Name: "inpainter", DisplayName: "Inpainting Specialist", Role: "Mask Inpainting", Description: "Precision mask-guided editing and restoration"},
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(results)
}

// HandleBackends lists registered backends and models.
func (h *Handlers) HandleBackends(w http.ResponseWriter, r *http.Request) {
	type BackendResponse struct {
		Name      string   `json:"name"`
		IsDefault bool     `json:"is_default"`
		Models    []string `json:"models"`
	}

	var results []BackendResponse
	if h.agent != nil && h.agent.Registry() != nil {
		reg := h.agent.Registry()
		def := reg.GetDefault()
		defName := ""
		if def != nil {
			defName = def.Name()
		}
		for _, name := range reg.List() {
			b, _ := reg.Get(name)
			var models []string
			if b != nil {
				models = b.SupportsModels()
			}
			results = append(results, BackendResponse{
				Name:      name,
				IsDefault: name == defName,
				Models:    models,
			})
		}
	}

	if len(results) == 0 {
		results = []BackendResponse{
			{Name: "pollinations", IsDefault: true, Models: []string{"flux", "flux-realism", "any-dark"}},
			{Name: "comfyui", IsDefault: false, Models: []string{"sd_xl_base", "flux1-dev"}},
			{Name: "falai", IsDefault: false, Models: []string{"fal-ai/flux/dev", "fal-ai/flux-realism"}},
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(results)
}
