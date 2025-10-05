package fanout

import (
	"context"
	"fmt"
	"sync"
)

type (
	// Task represents a unit of work to be processed
	Task struct {
		ID   int
		Data any
	}

	// Result holds the outcome of a processed task
	Result struct {
		TaskID int
		Output any
		Err    error
	}

	// Processor defines how to handle a task
	Processor func(context.Context, Task) (any, error)

	// Pipeline manages the fan-out and fan-in operations
	Pipeline struct {
		workerCount int
		processor   Processor
	}

	PipelineOption func(*Pipeline)
)

// WithWorkerCount sets the number of workers for fan-out
func WithWorkerCount(count int) PipelineOption {
	return func(p *Pipeline) {
		p.workerCount = count
	}
}

// NewPipeline creates a new fan-in/fan-out pipeline
func NewPipeline(processor Processor, opts ...PipelineOption) *Pipeline {
	p := &Pipeline{
		workerCount: 5,
		processor:   processor,
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

// Process executes tasks using the fan-out/fan-in pattern
func (p *Pipeline) Process(ctx context.Context, tasks []Task) []Result {
	taskChan := make(chan Task, len(tasks))
	resultChan := make(chan Result, len(tasks))

	// Fan-out: distribute tasks to workers
	var wg sync.WaitGroup
	for range p.workerCount {
		wg.Add(1)
		go p.worker(ctx, &wg, taskChan, resultChan)
	}

	// Send tasks to workers
	go func() {
		for _, task := range tasks {
			select {
			case taskChan <- task:
			case <-ctx.Done():
				close(taskChan)
				return
			}
		}
		close(taskChan)
	}()

	// Wait for workers and close result channel
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Fan-in: collect results
	return collectResults(resultChan)
}

// ProcessStream handles streaming tasks and returns a result channel
func (p *Pipeline) ProcessStream(ctx context.Context, taskChan <-chan Task) <-chan Result {
	resultChan := make(chan Result)

	var wg sync.WaitGroup
	for range p.workerCount {
		wg.Add(1)
		go p.worker(ctx, &wg, taskChan, resultChan)
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	return resultChan
}

func (p *Pipeline) worker(ctx context.Context, wg *sync.WaitGroup, tasks <-chan Task, results chan<- Result) {
	defer wg.Done()

	for task := range tasks {
		select {
		case <-ctx.Done():
			results <- Result{
				TaskID: task.ID,
				Err:    fmt.Errorf("context cancelled: %w", ctx.Err()),
			}
			return
		default:
			output, err := p.processor(ctx, task)
			results <- Result{
				TaskID: task.ID,
				Output: output,
				Err:    err,
			}
		}
	}
}

func collectResults(resultChan <-chan Result) []Result {
	results := make([]Result, 0)
	for result := range resultChan {
		results = append(results, result)
	}
	return results
}

// Multiplex combines multiple result channels into one (fan-in)
func Multiplex(ctx context.Context, channels ...<-chan Result) <-chan Result {
	out := make(chan Result)

	var wg sync.WaitGroup
	wg.Add(len(channels))

	for _, ch := range channels {
		go func(c <-chan Result) {
			defer wg.Done()
			for result := range c {
				select {
				case out <- result:
				case <-ctx.Done():
					return
				}
			}
		}(ch)
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}

// Split divides a task channel into multiple channels (fan-out)
func Split(ctx context.Context, input <-chan Task, n int) []<-chan Task {
	outputs := make([]chan Task, n)
	for i := range outputs {
		outputs[i] = make(chan Task)
	}

	go func() {
		defer func() {
			for _, ch := range outputs {
				close(ch)
			}
		}()

		i := 0
		for task := range input {
			select {
			case outputs[i%n] <- task:
				i++
			case <-ctx.Done():
				return
			}
		}
	}()

	// Convert to read-only channels
	result := make([]<-chan Task, n)
	for i := range outputs {
		result[i] = outputs[i]
	}
	return result
}
