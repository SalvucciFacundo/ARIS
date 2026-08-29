package services_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"aris/internal/adapters/image"
	"aris/internal/core/domain"
	"aris/internal/core/services"
)

// mockBackend implements ports.ImageBackend for testing.
type mockBackend struct {
	name             string
	delay            time.Duration
	failOnIndex      map[int]bool
	callCount        int64
	currentRunning   int64
	maxSeenRunning   int64
	mu               sync.Mutex
}

func newMockBackend(name string, delay time.Duration) *mockBackend {
	return &mockBackend{
		name:        name,
		delay:       delay,
		failOnIndex: make(map[int]bool),
	}
}

func (m *mockBackend) Name() string {
	return m.name
}

func (m *mockBackend) SupportsModels() []string {
	return []string{"flux", "sdxl"}
}

func (m *mockBackend) Generate(ctx context.Context, spec *domain.ImageSpec) (*domain.ImageResult, error) {
	curr := atomic.AddInt64(&m.currentRunning, 1)
	m.mu.Lock()
	if curr > m.maxSeenRunning {
		m.maxSeenRunning = curr
	}
	m.mu.Unlock()
	defer atomic.AddInt64(&m.currentRunning, -1)

	count := atomic.AddInt64(&m.callCount, 1)

	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	m.mu.Lock()
	shouldFail := m.failOnIndex[int(count)]
	m.mu.Unlock()

	if shouldFail {
		return nil, errors.New("simulated backend rate limit (HTTP 429)")
	}

	return &domain.ImageResult{
		ID:          fmt.Sprintf("res-%s-%d", m.name, spec.Seed),
		SpecID:      spec.ID,
		LocalPath:   fmt.Sprintf("/tmp/outputs/job_%d_%s.png", spec.Seed, m.name),
		SizeInBytes: 1024 * 500,
		Duration:    m.delay,
	}, nil
}

func TestParseSeedSweep(t *testing.T) {
	tests := []struct {
		input    string
		expected []int64
		wantErr  bool
	}{
		{
			input:    "100-105",
			expected: []int64{100, 101, 102, 103, 104, 105},
			wantErr:  false,
		},
		{
			input:    "500-500",
			expected: []int64{500},
			wantErr:  false,
		},
		{
			input:    "200-100", // Inverted
			expected: nil,
			wantErr:  true,
		},
		{
			input:    "foo-bar", // Non numeric
			expected: nil,
			wantErr:  true,
		},
		{
			input:    "100", // Missing delimiter
			expected: nil,
			wantErr:  true,
		},
		{
			input:    "-5-10", // Negative
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := services.ParseSeedSweep(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseSeedSweep(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if !tt.wantErr {
				if len(got) != len(tt.expected) {
					t.Fatalf("expected length %d, got %d", len(tt.expected), len(got))
				}
				for i, v := range tt.expected {
					if got[i] != v {
						t.Errorf("at index %d: expected %d, got %d", i, v, got[i])
					}
				}
			}
		})
	}
}

func TestBuildPlan(t *testing.T) {
	cfg := services.BatchConfig{
		Count:       2,
		Backends:    []string{"mock1", "mock2"},
		Model:       "flux",
		AspectRatio: domain.RatioLandscape,
		Concurrency: 4,
	}

	prompts := []string{"prompt A", "prompt B"}
	seeds := []int64{100, 101}
	backends := []string{"mock1", "mock2"}

	plan, err := services.BuildPlan(cfg, prompts, seeds, backends)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Total jobs: 2 prompts * 2 seeds * 2 backends = 8 jobs
	if len(plan.Jobs) != 8 {
		t.Fatalf("expected 8 jobs, got %d", len(plan.Jobs))
	}

	// Verify plan details
	firstJob := plan.Jobs[0]
	if firstJob.Prompt != "prompt A" || firstJob.Seed != 100 || firstJob.Backend != "mock1" {
		t.Errorf("unexpected first job: %+v", firstJob)
	}
}

func TestBatchRunner_Execute_Success(t *testing.T) {
	reg := image.NewRegistry()
	b1 := newMockBackend("mock1", 10*time.Millisecond)
	b2 := newMockBackend("mock2", 10*time.Millisecond)
	_ = reg.Register(b1)
	_ = reg.Register(b2)

	runner := services.NewBatchRunner(reg, nil)

	cfg := services.BatchConfig{
		Concurrency: 4,
		Backends:    []string{"mock1", "mock2"},
	}
	prompts := []string{"prompt 1", "prompt 2"}
	seeds := []int64{42, 43}
	plan, err := services.BuildPlan(cfg, prompts, seeds, []string{"mock1", "mock2"})
	if err != nil {
		t.Fatalf("BuildPlan error: %v", err)
	}

	var progressCalls int64
	runner.SetProgressCallback(func(result services.BatchJobResult, completed, total int) {
		atomic.AddInt64(&progressCalls, 1)
	})

	summary, err := runner.Execute(context.Background(), plan)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if summary.TotalJobs != 8 {
		t.Errorf("expected 8 total jobs, got %d", summary.TotalJobs)
	}
	if summary.SuccessCount != 8 {
		t.Errorf("expected 8 successes, got %d", summary.SuccessCount)
	}
	if summary.FailedCount != 0 {
		t.Errorf("expected 0 failures, got %d", summary.FailedCount)
	}
	if atomic.LoadInt64(&progressCalls) != 8 {
		t.Errorf("expected 8 progress callbacks, got %d", progressCalls)
	}
}

func TestBatchRunner_Execute_FailSoft(t *testing.T) {
	reg := image.NewRegistry()
	b1 := newMockBackend("mock1", 5*time.Millisecond)
	// Fail on job index 2 and 4 for mock1
	b1.failOnIndex[2] = true
	b1.failOnIndex[4] = true
	_ = reg.Register(b1)

	runner := services.NewBatchRunner(reg, nil)

	cfg := services.BatchConfig{
		Concurrency: 2,
		Backends:    []string{"mock1"},
	}
	prompts := []string{"p1", "p2", "p3", "p4"}
	seeds := []int64{10}
	plan, err := services.BuildPlan(cfg, prompts, seeds, []string{"mock1"})
	if err != nil {
		t.Fatalf("BuildPlan error: %v", err)
	}

	summary, err := runner.Execute(context.Background(), plan)
	if err != nil {
		t.Fatalf("Execute should not return error on fail-soft jobs: %v", err)
	}

	if summary.TotalJobs != 4 {
		t.Errorf("expected 4 total jobs, got %d", summary.TotalJobs)
	}
	if summary.SuccessCount != 2 {
		t.Errorf("expected 2 successes, got %d", summary.SuccessCount)
	}
	if summary.FailedCount != 2 {
		t.Errorf("expected 2 failures, got %d", summary.FailedCount)
	}

	// Verify failed results have Error set
	for _, res := range summary.Results {
		if res.Status == "FAILED" && res.Error == "" {
			t.Errorf("expected non-empty Error for failed job: %+v", res)
		}
	}
}

func TestBatchRunner_Execute_BackendThrottling(t *testing.T) {
	reg := image.NewRegistry()
	// comfyui should be throttled to concurrency 1 by default
	comfy := newMockBackend("comfyui", 30*time.Millisecond)
	_ = reg.Register(comfy)

	runner := services.NewBatchRunner(reg, nil)

	cfg := services.BatchConfig{
		Concurrency: 8, // high global concurrency
		Backends:    []string{"comfyui"},
	}
	prompts := []string{"p1", "p2", "p3", "p4"}
	seeds := []int64{1, 2}
	plan, _ := services.BuildPlan(cfg, prompts, seeds, []string{"comfyui"})

	summary, err := runner.Execute(context.Background(), plan)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if summary.SuccessCount != 8 {
		t.Fatalf("expected 8 successes, got %d", summary.SuccessCount)
	}

	// Check max concurrent executions seen by comfyui
	if comfy.maxSeenRunning > 1 {
		t.Errorf("comfyui expected max concurrency of 1, but observed %d", comfy.maxSeenRunning)
	}
}

func TestBatchRunner_Execute_ContextCancellation(t *testing.T) {
	reg := image.NewRegistry()
	b1 := newMockBackend("mock1", 100*time.Millisecond)
	_ = reg.Register(b1)

	runner := services.NewBatchRunner(reg, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	cfg := services.BatchConfig{
		Concurrency: 2,
		Backends:    []string{"mock1"},
	}
	prompts := []string{"p1", "p2", "p3", "p4", "p5", "p6"}
	seeds := []int64{1}
	plan, _ := services.BuildPlan(cfg, prompts, seeds, []string{"mock1"})

	summary, err := runner.Execute(ctx, plan)
	if err == nil {
		t.Fatal("expected context cancellation error, got nil")
	}

	// Summary should still contain partial jobs that were processed or attempted
	if summary == nil {
		t.Fatal("expected non-nil summary even on cancellation")
	}
}
