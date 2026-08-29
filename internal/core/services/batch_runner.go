package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"aris/internal/core/domain"
	"aris/internal/core/ports"
)

// BatchConfig defines the parameters for a batch execution run.
type BatchConfig struct {
	Count          int
	SeedSweep      string
	BaseSeed       int64
	Matrix         bool
	Benchmark      bool
	Backends       []string
	Concurrency    int
	OutputDir      string
	MaxMatrixJobs  int
	Force          bool
	DryRun         bool
	Eval           bool
	Model          string
	AspectRatio    domain.AspectRatio
	NegativePrompt string
}

// BatchJob represents an atomic image generation task in a batch.
type BatchJob struct {
	ID             string             `json:"id"`
	Index          int                `json:"index"`
	Prompt         string             `json:"prompt"`
	Seed           int64              `json:"seed"`
	Backend        string             `json:"backend"`
	Model          string             `json:"model"`
	AspectRatio    domain.AspectRatio `json:"aspect_ratio"`
	NegativePrompt string             `json:"negative_prompt"`
	EnableCritic   bool               `json:"enable_critic"`
	ExtraParams    map[string]any     `json:"extra_params,omitempty"`
}

// BatchPlan defines the compiled list of jobs to be executed.
type BatchPlan struct {
	BatchID   string      `json:"batch_id"`
	OutputDir string      `json:"output_dir"`
	Config    BatchConfig `json:"config"`
	Jobs      []BatchJob  `json:"jobs"`
}

// BatchJobResult contains the execution telemetry and output of a single BatchJob.
type BatchJobResult struct {
	Job            BatchJob            `json:"job"`
	Spec           *domain.ImageSpec   `json:"spec,omitempty"`
	Result         *domain.ImageResult `json:"result,omitempty"`
	Status         string              `json:"status"` // "SUCCESS" or "FAILED"
	Error          string              `json:"error,omitempty"`
	Duration       time.Duration       `json:"duration"`
	DurationMs     int64               `json:"duration_ms"`
	ImageSizeBytes int64               `json:"image_size_bytes,omitempty"`
	ImagePath      string              `json:"image_path,omitempty"`
	Resolution     string              `json:"resolution,omitempty"`
	CriticScore    *float64            `json:"critic_score,omitempty"`
	CriticNotes    string              `json:"critic_notes,omitempty"`
}

// BatchSummary aggregates the results of an entire batch execution run.
type BatchSummary struct {
	BatchID         string           `json:"batch_id"`
	CreatedAt       time.Time        `json:"created_at"`
	TotalJobs       int              `json:"total_jobs"`
	SuccessCount    int              `json:"success_count"`
	FailedCount     int              `json:"failed_count"`
	TotalDuration   time.Duration    `json:"total_duration"`
	TotalDurationMs int64            `json:"total_duration_ms"`
	OutputDir       string           `json:"output_dir"`
	Config          BatchConfig      `json:"config"`
	Results         []BatchJobResult `json:"results"`
}

// ProgressCallback is invoked when an individual job completes.
type ProgressCallback func(result BatchJobResult, completed int, total int)

// BatchRunner coordinates worker pools, rate limiting, and telemetry collection.
type BatchRunner struct {
	registry   ports.BackendRegistry
	critic     ports.VisionCritic
	progressCb ProgressCallback

	mu         sync.Mutex
	semaphores map[string]chan struct{}
}

// NewBatchRunner creates a new BatchRunner instance.
func NewBatchRunner(registry ports.BackendRegistry, critic ports.VisionCritic) *BatchRunner {
	return &BatchRunner{
		registry:   registry,
		critic:     critic,
		semaphores: make(map[string]chan struct{}),
	}
}

// SetProgressCallback registers a listener for real-time progress events.
func (r *BatchRunner) SetProgressCallback(cb ProgressCallback) {
	r.progressCb = cb
}

func (r *BatchRunner) getSemaphore(backend string) chan struct{} {
	r.mu.Lock()
	defer r.mu.Unlock()

	if sem, ok := r.semaphores[backend]; ok {
		return sem
	}

	// ComfyUI (local GPU) defaults to concurrency of 1 to avoid VRAM exhaustion
	capacity := 100
	if backend == "comfyui" {
		capacity = 1
	}

	sem := make(chan struct{}, capacity)
	r.semaphores[backend] = sem
	return sem
}

// ParseSeedSweep parses a string in the format "<start>-<end>" into a slice of sequential seeds.
func ParseSeedSweep(sweep string) ([]int64, error) {
	sweep = strings.TrimSpace(sweep)
	parts := strings.Split(sweep, "-")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid seed sweep format %q (expected <start>-<end>)", sweep)
	}

	start, err := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid start seed %q: %w", parts[0], err)
	}

	end, err := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid end seed %q: %w", parts[1], err)
	}

	if start < 0 || end < 0 {
		return nil, fmt.Errorf("seed values must be non-negative (got %d to %d)", start, end)
	}

	if start > end {
		return nil, fmt.Errorf("start seed (%d) must be less than or equal to end seed (%d)", start, end)
	}

	count := int(end - start + 1)
	seeds := make([]int64, count)
	for i := 0; i < count; i++ {
		seeds[i] = start + int64(i)
	}
	return seeds, nil
}

// BuildPlan constructs a BatchPlan from prompts, seeds, and backends.
func BuildPlan(cfg BatchConfig, prompts []string, seeds []int64, backends []string) (*BatchPlan, error) {
	if len(prompts) == 0 {
		return nil, fmt.Errorf("prompts list cannot be empty")
	}
	if len(seeds) == 0 {
		return nil, fmt.Errorf("seeds list cannot be empty")
	}
	if len(backends) == 0 {
		return nil, fmt.Errorf("backends list cannot be empty")
	}

	randBytes := make([]byte, 4)
	_, _ = rand.Read(randBytes)
	shortID := hex.EncodeToString(randBytes)
	batchID := fmt.Sprintf("batch_%s_%s", time.Now().Format("20060102_150405"), shortID)

	var jobs []BatchJob
	idx := 1

	for _, prompt := range prompts {
		for _, seed := range seeds {
			for _, backend := range backends {
				job := BatchJob{
					ID:             fmt.Sprintf("job-%03d", idx),
					Index:          idx,
					Prompt:         prompt,
					Seed:           seed,
					Backend:        backend,
					Model:          cfg.Model,
					AspectRatio:    cfg.AspectRatio,
					NegativePrompt: cfg.NegativePrompt,
					EnableCritic:   cfg.Eval,
				}
				jobs = append(jobs, job)
				idx++
			}
		}
	}

	return &BatchPlan{
		BatchID:   batchID,
		OutputDir: cfg.OutputDir,
		Config:    cfg,
		Jobs:      jobs,
	}, nil
}

// Execute processes all planned jobs using a worker pool with backend throttling.
func (r *BatchRunner) Execute(ctx context.Context, plan *BatchPlan) (*BatchSummary, error) {
	if plan == nil || len(plan.Jobs) == 0 {
		return nil, fmt.Errorf("batch plan is empty")
	}

	concurrency := plan.Config.Concurrency
	if concurrency <= 0 {
		concurrency = 2
	}
	if concurrency > len(plan.Jobs) {
		concurrency = len(plan.Jobs)
	}

	jobsCh := make(chan BatchJob, len(plan.Jobs))
	resultsCh := make(chan BatchJobResult, len(plan.Jobs))

	for _, j := range plan.Jobs {
		jobsCh <- j
	}
	close(jobsCh)

	startTime := time.Now()
	var wg sync.WaitGroup

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobsCh {
				select {
				case <-ctx.Done():
					resultsCh <- BatchJobResult{
						Job:        job,
						Status:     "FAILED",
						Error:      fmt.Sprintf("cancelled: %v", ctx.Err()),
						Duration:   0,
						DurationMs: 0,
					}
					continue
				default:
				}

				res := r.executeJob(ctx, job)
				resultsCh <- res
			}
		}()
	}

	wg.Wait()
	close(resultsCh)

	var results []BatchJobResult
	successCount := 0
	failedCount := 0
	completed := 0

	for res := range resultsCh {
		completed++
		if res.Status == "SUCCESS" {
			successCount++
		} else {
			failedCount++
		}
		results = append(results, res)
		if r.progressCb != nil {
			r.progressCb(res, completed, len(plan.Jobs))
		}
	}

	// Sort results deterministically by job index
	sort.Slice(results, func(i, j int) bool {
		return results[i].Job.Index < results[j].Job.Index
	})

	totalDuration := time.Since(startTime)

	summary := &BatchSummary{
		BatchID:         plan.BatchID,
		CreatedAt:       startTime,
		TotalJobs:       len(plan.Jobs),
		SuccessCount:    successCount,
		FailedCount:     failedCount,
		TotalDuration:   totalDuration,
		TotalDurationMs: totalDuration.Milliseconds(),
		OutputDir:       plan.OutputDir,
		Config:          plan.Config,
		Results:         results,
	}

	if ctx.Err() != nil {
		return summary, ctx.Err()
	}

	return summary, nil
}

func (r *BatchRunner) executeJob(ctx context.Context, job BatchJob) BatchJobResult {
	start := time.Now()

	// Acquire backend semaphore to enforce concurrency limits (e.g. VRAM locking)
	sem := r.getSemaphore(job.Backend)
	select {
	case sem <- struct{}{}:
		defer func() { <-sem }()
	case <-ctx.Done():
		return BatchJobResult{
			Job:        job,
			Status:     "FAILED",
			Error:      fmt.Sprintf("context cancelled while waiting for backend %s: %v", job.Backend, ctx.Err()),
			Duration:   time.Since(start),
			DurationMs: time.Since(start).Milliseconds(),
		}
	}

	backend, err := r.registry.Get(job.Backend)
	if err != nil {
		return BatchJobResult{
			Job:        job,
			Status:     "FAILED",
			Error:      fmt.Sprintf("backend %q unavailable: %v", job.Backend, err),
			Duration:   time.Since(start),
			DurationMs: time.Since(start).Milliseconds(),
		}
	}

	w, h := job.AspectRatio.Dimensions(1024)
	if job.AspectRatio == "" {
		w, h = 1024, 1024
	}

	spec := &domain.ImageSpec{
		ID:             fmt.Sprintf("%s-%d", job.ID, job.Seed),
		RawPrompt:      job.Prompt,
		EnhancedPrompt: job.Prompt,
		NegativePrompt: job.NegativePrompt,
		AspectRatio:    job.AspectRatio,
		Width:          w,
		Height:         h,
		Seed:           job.Seed,
		Backend:        job.Backend,
		Model:          job.Model,
		CreatedAt:      time.Now(),
	}
	spec.ApplyDefaults()

	result, err := backend.Generate(ctx, spec)
	duration := time.Since(start)

	if err != nil {
		return BatchJobResult{
			Job:        job,
			Spec:       spec,
			Status:     "FAILED",
			Error:      err.Error(),
			Duration:   duration,
			DurationMs: duration.Milliseconds(),
		}
	}

	jobRes := BatchJobResult{
		Job:            job,
		Spec:           spec,
		Result:         result,
		Status:         "SUCCESS",
		Duration:       duration,
		DurationMs:     duration.Milliseconds(),
		ImageSizeBytes: result.SizeInBytes,
		ImagePath:      result.LocalPath,
		Resolution:     fmt.Sprintf("%dx%d", spec.Width, spec.Height),
	}

	// Vision Critic Evaluation if enabled
	if job.EnableCritic && r.critic != nil {
		score, notes, cErr := r.critic.Evaluate(ctx, result.LocalPath, spec)
		if cErr == nil {
			jobRes.CriticScore = &score
			jobRes.CriticNotes = notes
		}
	}

	return jobRes
}
