package gateway_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"aris/internal/adapters/gateway"
)

type MockAdapter struct {
	mu           sync.Mutex
	name         string
	started      bool
	stopped      bool
	startErr     error
	stopErr      error
	startDelay   time.Duration
	stopDelay    time.Duration
}

func (m *MockAdapter) Name() string { return m.name }

func (m *MockAdapter) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.startDelay > 0 {
		time.Sleep(m.startDelay)
	}
	if m.startErr != nil {
		return m.startErr
	}
	m.started = true
	return nil
}

func (m *MockAdapter) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.stopDelay > 0 {
		time.Sleep(m.stopDelay)
	}
	if m.stopErr != nil {
		return m.stopErr
	}
	m.stopped = true
	return nil
}

func (m *MockAdapter) IsStarted() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.started
}

func (m *MockAdapter) IsStopped() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.stopped
}

func TestMultiplexer_ConcurrentStartAndStop(t *testing.T) {
	q := gateway.NewJobQueue(2, 5)
	a1 := &MockAdapter{name: "telegram"}
	a2 := &MockAdapter{name: "discord"}

	mux := gateway.NewMultiplexer([]gateway.GatewayAdapter{a1, a2}, q)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := mux.Start(ctx); err != nil {
		t.Fatalf("Multiplexer Start failed: %v", err)
	}

	if !a1.IsStarted() || !a2.IsStarted() {
		t.Fatalf("expected all adapters to be started")
	}

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer stopCancel()

	if err := mux.Stop(stopCtx); err != nil {
		t.Fatalf("Multiplexer Stop failed: %v", err)
	}

	if !a1.IsStopped() || !a2.IsStopped() {
		t.Fatalf("expected all adapters to be stopped")
	}
}

func TestMultiplexer_NoAdaptersError(t *testing.T) {
	q := gateway.NewJobQueue(1, 2)
	mux := gateway.NewMultiplexer([]gateway.GatewayAdapter{}, q)

	ctx := context.Background()
	err := mux.Start(ctx)
	if !errors.Is(err, gateway.ErrNoAdaptersEnabled) {
		t.Fatalf("expected ErrNoAdaptersEnabled, got %v", err)
	}
}
