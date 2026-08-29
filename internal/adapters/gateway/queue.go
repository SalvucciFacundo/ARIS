package gateway

import (
	"context"
	"errors"
	"sync"
)

// ErrQueueFull is returned when the job queue buffer has reached its maximum configured capacity.
var ErrQueueFull = errors.New("generation queue is full")

// ErrQueueClosed is returned when submitting a job to a stopped queue.
var ErrQueueClosed = errors.New("generation queue is closed")

// Job encapsulates an asynchronous work unit executed by the gateway worker pool.
type Job struct {
	Ctx      context.Context
	Task     func(ctx context.Context) error
	ReplyErr func(error)
}

// JobQueue manages concurrency and bounded buffering of image generation tasks.
type JobQueue struct {
	concurrency int
	maxQueue    int
	queue       chan Job
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	mu          sync.RWMutex
	running     bool
}

// NewJobQueue creates a JobQueue with the specified concurrency limit and max queue depth.
func NewJobQueue(concurrency, maxQueue int) *JobQueue {
	if concurrency <= 0 {
		concurrency = 1
	}
	if maxQueue <= 0 {
		maxQueue = 10
	}
	return &JobQueue{
		concurrency: concurrency,
		maxQueue:    maxQueue,
		queue:       make(chan Job, maxQueue),
	}
}

// Start spawns the concurrency worker routines.
func (q *JobQueue) Start(ctx context.Context) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.running {
		return
	}

	q.ctx, q.cancel = context.WithCancel(ctx)
	q.running = true

	for i := 0; i < q.concurrency; i++ {
		q.wg.Add(1)
		go q.worker()
	}
}

// Submit attempts to enqueue a job. Returns ErrQueueFull if the queue is saturated.
func (q *JobQueue) Submit(job Job) error {
	q.mu.RLock()
	if !q.running {
		q.mu.RUnlock()
		return ErrQueueClosed
	}
	q.mu.RUnlock()

	select {
	case q.queue <- job:
		return nil
	default:
		return ErrQueueFull
	}
}

// Pending returns the number of jobs currently waiting in the queue buffer.
func (q *JobQueue) Pending() int {
	return len(q.queue)
}

// Capacity returns the maximum buffer depth of the queue.
func (q *JobQueue) Capacity() int {
	return q.maxQueue
}

// Stop gracefully shuts down the queue and waits for in-flight jobs to complete.
func (q *JobQueue) Stop() error {
	q.mu.Lock()
	if !q.running {
		q.mu.Unlock()
		return nil
	}
	q.running = false
	if q.cancel != nil {
		q.cancel()
	}
	q.mu.Unlock()

	q.wg.Wait()
	return nil
}

func (q *JobQueue) worker() {
	defer q.wg.Done()

	for {
		select {
		case <-q.ctx.Done():
			return
		case job, ok := <-q.queue:
			if !ok {
				return
			}

			// Check if job's context was already cancelled before starting
			if job.Ctx != nil && job.Ctx.Err() != nil {
				if job.ReplyErr != nil {
					job.ReplyErr(job.Ctx.Err())
				}
				continue
			}

			if job.Task != nil {
				taskCtx := job.Ctx
				if taskCtx == nil {
					taskCtx = q.ctx
				}
				if err := job.Task(taskCtx); err != nil && job.ReplyErr != nil {
					job.ReplyErr(err)
				}
			}
		}
	}
}
