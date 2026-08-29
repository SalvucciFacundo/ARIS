package gateway

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
)

// ErrNoAdaptersEnabled is returned when the Multiplexer is started with zero active messaging adapters.
var ErrNoAdaptersEnabled = errors.New("no gateway adapters are enabled")

// Multiplexer coordinates the concurrent lifecycle of multiple GatewayAdapters and the central JobQueue.
type Multiplexer struct {
	adapters []GatewayAdapter
	queue    *JobQueue
	mu       sync.Mutex
	running  bool
}

// NewMultiplexer creates a new Gateway Multiplexer instance.
func NewMultiplexer(adapters []GatewayAdapter, queue *JobQueue) *Multiplexer {
	return &Multiplexer{
		adapters: adapters,
		queue:    queue,
	}
}

// Start boots the JobQueue and launches all configured gateway adapters concurrently.
func (m *Multiplexer) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return nil
	}

	if len(m.adapters) == 0 {
		return ErrNoAdaptersEnabled
	}

	if m.queue != nil {
		m.queue.Start(ctx)
	}

	var wg sync.WaitGroup
	errCh := make(chan error, len(m.adapters))

	for _, adapter := range m.adapters {
		wg.Add(1)
		go func(a GatewayAdapter) {
			defer wg.Done()
			log.Printf("🚀 Starting Gateway adapter [%s]...", a.Name())
			if err := a.Start(ctx); err != nil {
				errCh <- fmt.Errorf("adapter [%s] failed to start: %w", a.Name(), err)
			}
		}(adapter)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			return err
		}
	}

	m.running = true
	log.Printf("🌐 ARIS Gateway Multiplexer is active with %d adapters.", len(m.adapters))
	return nil
}

// Stop gracefully shuts down all adapters and the worker queue.
func (m *Multiplexer) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return nil
	}

	log.Printf("🛑 Shutting down ARIS Gateway Multiplexer...")

	var wg sync.WaitGroup
	for _, adapter := range m.adapters {
		wg.Add(1)
		go func(a GatewayAdapter) {
			defer wg.Done()
			if err := a.Stop(ctx); err != nil {
				log.Printf("⚠️ Error stopping adapter [%s]: %v", a.Name(), err)
			}
		}(adapter)
	}

	wg.Wait()

	if m.queue != nil {
		_ = m.queue.Stop()
	}

	m.running = false
	log.Printf("✅ Gateway Multiplexer successfully stopped.")
	return nil
}
