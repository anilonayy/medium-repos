package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/anilonayy/medium-repos/go-concurrency/benchmark"
	"github.com/anilonayy/medium-repos/go-concurrency/fanout"
	"github.com/anilonayy/medium-repos/go-concurrency/worker"
)

func main() {
	log.Println("=== Go Concurrency Patterns Demo ===")
	fmt.Println("1. Worker Pool Pattern")
	fmt.Println("2. Fan-In/Fan-Out Pattern")
	fmt.Println("3. Performance Comparison")
	fmt.Println("4. Multiple Scenario Benchmark")
	fmt.Print("\nEnter your choice (1-4): ")

	var choice int
	fmt.Scanln(&choice)

	switch choice {
	case 1:
		runWorkerPoolExample()
	case 2:
		runFanOutExample()
	case 3:
		runPerformanceComparison()
	case 4:
		runMultipleScenarios()
	default:
		log.Println("Invalid choice, running Worker Pool example by default")
		runWorkerPoolExample()
	}
}

func runWorkerPoolExample() {
	log.Println("\n=== Worker Pool Pattern ===")

	// Create a worker pool with 5 workers
	pool := worker.NewPool(
		"example-pool",
		worker.WithWorkerSize(5),
		worker.WithQueueSize(100),
		worker.WithTimeout(10*time.Second),
	)

	// Add some example jobs
	for i := 1; i <= 20; i++ {
		jobID := i
		pool.AddJob(func(ctx context.Context) error {
			// Simulate some work
			duration := time.Duration(rand.Intn(3)+1) * time.Second

			select {
			case <-time.After(duration):
				log.Printf("Job %d completed after %v", jobID, duration)
				return nil
			case <-ctx.Done():
				return fmt.Errorf("job %d cancelled: %w", jobID, ctx.Err())
			}
		})
	}

	pool.AddJob(func(ctx context.Context) error {
		time.Sleep(3 * time.Second) // This will timeout
		return nil
	})

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("\nReceived interrupt signal, shutting down...")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := pool.Shutdown(shutdownCtx); err != nil {
			log.Printf("Shutdown error: %v", err)
		}

		os.Exit(0)
	}()

	// Wait for all jobs to complete
	pool.Wait()
	log.Println("All jobs completed!")
}

func runFanOutExample() {
	log.Println("\n=== Fan-In/Fan-Out Pattern ===")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Define a processor that simulates data transformation
	processor := func(ctx context.Context, task fanout.Task) (any, error) {
		// Simulate processing time
		duration := time.Duration(rand.Intn(2)+1) * time.Second
		time.Sleep(duration)

		// Example: multiply the input by 2
		if num, ok := task.Data.(int); ok {
			result := num * 2
			log.Printf("Task %d: processed %d -> %d (took %v)", task.ID, num, result, duration)
			return result, nil
		}

		return nil, fmt.Errorf("invalid task data type")
	}

	// Create a pipeline with 3 workers
	pipeline := fanout.NewPipeline(processor, fanout.WithWorkerCount(3))

	// Prepare tasks
	tasks := make([]fanout.Task, 15)
	for i := range 15 {
		tasks[i] = fanout.Task{
			ID:   i + 1,
			Data: (i + 1) * 10,
		}
	}

	log.Printf("Processing %d tasks with fan-out pattern...\n", len(tasks))
	start := time.Now()

	// Process tasks
	results := pipeline.Process(ctx, tasks)

	elapsed := time.Since(start)
	log.Printf("\nAll tasks completed in %v\n", elapsed)

	// Display results
	successCount := 0
	for _, result := range results {
		if result.Err == nil {
			successCount++
		} else {
			log.Printf("Task %d failed: %v", result.TaskID, result.Err)
		}
	}

	log.Printf("Success: %d/%d tasks", successCount, len(tasks))

	// Demonstrate streaming with fan-in
	log.Println("\n=== Streaming Example with Fan-In ===")
	demoStreamingFanIn(ctx)
}

func demoStreamingFanIn(ctx context.Context) {
	processor := func(ctx context.Context, task fanout.Task) (any, error) {
		time.Sleep(time.Duration(rand.Intn(500)) * time.Millisecond)
		return fmt.Sprintf("processed-%v", task.Data), nil
	}

	pipeline := fanout.NewPipeline(processor, fanout.WithWorkerCount(4))

	// Create task channel
	taskChan := make(chan fanout.Task)
	go func() {
		defer close(taskChan)
		for i := range 10 {
			taskChan <- fanout.Task{
				ID:   i + 1,
				Data: fmt.Sprintf("item-%d", i+1),
			}
		}
	}()

	// Process streaming tasks
	resultChan := pipeline.ProcessStream(ctx, taskChan)

	// Collect results
	count := 0
	for result := range resultChan {
		if result.Err != nil {
			log.Printf("Stream task %d failed: %v", result.TaskID, result.Err)
		} else {
			log.Printf("Stream task %d: %v", result.TaskID, result.Output)
			count++
		}
	}

	log.Printf("Streaming completed: %d tasks processed", count)
}

func runPerformanceComparison() {
	fmt.Println("\n========== PERFORMANCE COMPARISON SETUP ==========")
	fmt.Println("\nSelect workload configuration:")
	fmt.Println("1. Light (1000 tasks, 10 workers, ~10ms per task)")
	fmt.Println("2. Medium (5000 tasks, 25 workers, ~20ms per task)")
	fmt.Println("3. Heavy (10000 tasks, 50 workers, ~50ms per task)")
	fmt.Println("4. Very Heavy (20000 tasks, 100 workers, ~100ms per task)")
	fmt.Println("5. Custom")
	fmt.Print("\nEnter your choice (1-5): ")

	var choice int
	fmt.Scanln(&choice)

	var taskCount, workerCount int
	var taskDuration time.Duration

	switch choice {
	case 1:
		taskCount, workerCount, taskDuration = 1000, 10, 10*time.Millisecond
	case 2:
		taskCount, workerCount, taskDuration = 5000, 25, 20*time.Millisecond
	case 3:
		taskCount, workerCount, taskDuration = 10000, 50, 50*time.Millisecond
	case 4:
		taskCount, workerCount, taskDuration = 20000, 100, 100*time.Millisecond
	case 5:
		fmt.Print("Enter number of tasks: ")
		fmt.Scanln(&taskCount)
		fmt.Print("Enter number of workers: ")
		fmt.Scanln(&workerCount)
		var ms int
		fmt.Print("Enter task duration in milliseconds: ")
		fmt.Scanln(&ms)
		taskDuration = time.Duration(ms) * time.Millisecond
	default:
		log.Println("Invalid choice, using medium configuration")
		taskCount, workerCount, taskDuration = 5000, 25, 20*time.Millisecond
	}

	benchmark.RunComparison(taskCount, workerCount, taskDuration)
}

func runMultipleScenarios() {
	fmt.Println("\n========== RUNNING MULTIPLE SCENARIO BENCHMARK ==========")
	fmt.Println("\nThis will test both patterns across different workloads:")
	fmt.Println("  - Light Load (1000 tasks)")
	fmt.Println("  - Medium Load (5000 tasks)")
	fmt.Println("  - Heavy Load (10000 tasks)")
	fmt.Println("  - Very Heavy Load (20000 tasks)")
	fmt.Println("\nPress Enter to continue...")
	fmt.Scanln()

	benchmark.RunMultipleScenarios()
}
