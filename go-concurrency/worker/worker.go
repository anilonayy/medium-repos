package worker

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

type (
	// Pool represents a worker pool that executes jobs concurrently
	Pool struct {
		name     string
		workers  int
		timeout  time.Duration
		delay    time.Duration
		queue    chan Job
		shutdown chan struct{}
		sync.WaitGroup
	}

	// Job represents a task to be executed by the worker pool
	Job struct {
		task func(ctx context.Context) error
	}

	// Option is a functional option for configuring the Pool
	Option func(*Pool)
)

// WithWorkerSize sets the number of concurrent workers
func WithWorkerSize(workers int) Option {
	return func(p *Pool) {
		p.workers = workers
	}
}

// WithQueueSize sets the size of the job queue
func WithQueueSize(size int) Option {
	return func(p *Pool) {
		p.queue = make(chan Job, size)
	}
}

// WithTimeout sets the maximum execution time for each job
func WithTimeout(timeout time.Duration) Option {
	return func(p *Pool) {
		p.timeout = timeout
	}
}

// WithDelay sets a delay between job executions
func WithDelay(delay time.Duration) Option {
	return func(p *Pool) {
		p.delay = delay
	}
}

// NewPool creates and starts a new worker pool
func NewPool(name string, opts ...Option) *Pool {
	pool := &Pool{
		name:     name,
		workers:  10,
		queue:    make(chan Job, 100),
		timeout:  60 * time.Second,
		delay:    0,
		shutdown: make(chan struct{}),
	}

	for _, opt := range opts {
		opt(pool)
	}

	// Start the workers
	for range pool.workers {
		go pool.worker()
	}

	log.Printf("[%s] Worker pool started with %d workers", pool.name, pool.workers)

	return pool
}

// AddJob adds a job to the worker pool queue
func (p *Pool) AddJob(task func(ctx context.Context) error) {
	p.Add(1)
	p.queue <- Job{task: task}
}

// worker is the goroutine that processes jobs from the queue
func (p *Pool) worker() {
	for {
		select {
		case job, ok := <-p.queue:
			if !ok {
				return
			}

			// Optional delay between jobs
			if p.delay > 0 {
				time.Sleep(p.delay)
			}

			// Execute the job with timeout
			p.executeJob(job)

		case <-p.shutdown:
			return
		}
	}
}

// executeJob runs a job with timeout and panic recovery
func (p *Pool) executeJob(job Job) {
	defer p.Done()

	ctx, cancel := context.WithTimeout(context.Background(), p.timeout)
	defer cancel()

	// Recover from panics
	defer func() {
		if err := recover(); err != nil {
			log.Printf("[%s] Worker panic: %v", p.name, err)
		}
	}()

	// Execute the job
	if err := job.task(ctx); err != nil {
		log.Printf("[%s] Job error: %v", p.name, err)
		return
	}

	// Check for timeout
	if ctx.Err() == context.DeadlineExceeded {
		log.Printf("[%s] Job timed out after %v", p.name, p.timeout)
	}
}

// Shutdown gracefully shuts down the worker pool
// It stops accepting new jobs and waits for existing jobs to complete
func (p *Pool) Shutdown(ctx context.Context) error {
	log.Printf("[%s] Shutting down worker pool...", p.name)

	// Signal workers to stop
	close(p.shutdown)

	// Close the queue to prevent new jobs
	close(p.queue)

	// Wait for all jobs to complete with a timeout
	done := make(chan struct{})
	go func() {
		p.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Printf("[%s] Worker pool shut down successfully", p.name)
		return nil
	case <-ctx.Done():
		return fmt.Errorf("[%s] shutdown timeout: %w", p.name, ctx.Err())
	}
}
