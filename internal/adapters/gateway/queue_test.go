package gateway_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"aris/internal/adapters/gateway"
)

func TestJobQueue_ConcurrencyAndQueueLimit(t *testing.T) {
	concurrency := 2
	maxQueue := 3

	q := gateway.NewJobQueue(concurrency, maxQueue)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	q.Start(ctx)
	defer q.Stop()

	var activeCount int32
	var maxObservedActive int32
	var completedCount int32

	blockCh := make(chan struct{})
	defer func() {
		select {
		case <-blockCh:
		default:
			close(blockCh)
		}
	}()

	task := func(ctx context.Context) error {
		current := atomic.AddInt32(&activeCount, 1)
		for {
			max := atomic.LoadInt32(&maxObservedActive)
			if current > max {
				if atomic.CompareAndSwapInt32(&maxObservedActive, max, current) {
					break
				}
			} else {
				break
			}
		}

		<-blockCh
		atomic.AddInt32(&activeCount, -1)
		atomic.AddInt32(&completedCount, 1)
		return nil
	}

	// Submit concurrency (2) jobs that will be picked up by workers
	for i := 0; i < concurrency; i++ {
		if err := q.Submit(gateway.Job{Ctx: ctx, Task: task}); err != nil {
			t.Fatalf("concurrency job %d submit failed unexpectedly: %v", i, err)
		}
	}

	// Wait briefly for workers to consume the 2 jobs from the queue channel into active execution
	deadline := time.Now().Add(1 * time.Second)
	for atomic.LoadInt32(&activeCount) < int32(concurrency) {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %d workers to become active", concurrency)
		}
		time.Sleep(5 * time.Millisecond)
	}

	// Now fill the maxQueue (3) buffer
	for i := 0; i < maxQueue; i++ {
		if err := q.Submit(gateway.Job{Ctx: ctx, Task: task}); err != nil {
			t.Fatalf("buffered job %d submit failed unexpectedly: %v", i, err)
		}
	}

	// Submitting the next job must return ErrQueueFull since buffer is completely full
	err := q.Submit(gateway.Job{
		Ctx:  ctx,
		Task: task,
	})
	if !errors.Is(err, gateway.ErrQueueFull) {
		t.Fatalf("expected ErrQueueFull, got %v", err)
	}

	// Unblock all running tasks
	close(blockCh)

	// Wait for the 5 submitted jobs to finish (2 active + 3 in queue)
	drainDeadline := time.Now().Add(2 * time.Second)
	for atomic.LoadInt32(&completedCount) < 5 {
		if time.Now().After(drainDeadline) {
			t.Fatalf("timed out waiting for jobs to complete, completed=%d", completedCount)
		}
		time.Sleep(10 * time.Millisecond)
	}

	if maxObservedActive > int32(concurrency) {
		t.Errorf("max active concurrency exceeded: observed %d, max allowed %d", maxObservedActive, concurrency)
	}
}

func TestJobQueue_ContextCancellation(t *testing.T) {
	q := gateway.NewJobQueue(1, 2)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	q.Start(ctx)
	defer q.Stop()

	jobCtx, jobCancel := context.WithCancel(context.Background())
	jobCancel() // Cancel immediately

	var executed bool
	err := q.Submit(gateway.Job{
		Ctx: jobCtx,
		Task: func(c context.Context) error {
			executed = true
			return nil
		},
	})
	if err != nil {
		t.Fatalf("submit failed: %v", err)
	}

	time.Sleep(50 * time.Millisecond)
	if executed {
		t.Errorf("expected cancelled job not to execute task")
	}
}

func TestJobQueue_RaceUnderLoad(t *testing.T) {
	q := gateway.NewJobQueue(4, 20)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	q.Start(ctx)
	defer q.Stop()

	var wg sync.WaitGroup
	var accepted int64
	var rejected int64

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := q.Submit(gateway.Job{
				Ctx: ctx,
				Task: func(c context.Context) error {
					time.Sleep(2 * time.Millisecond)
					return nil
				},
			})
			if err != nil {
				if errors.Is(err, gateway.ErrQueueFull) {
					atomic.AddInt64(&rejected, 1)
				}
			} else {
				atomic.AddInt64(&accepted, 1)
			}
		}()
	}

	wg.Wait()
	if accepted == 0 {
		t.Errorf("expected some jobs to be accepted")
	}
}
