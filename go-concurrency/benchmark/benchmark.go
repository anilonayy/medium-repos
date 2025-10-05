package benchmark

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync/atomic"
	"time"

	"github.com/anilonayy/medium-repos/go-concurrency/fanout"
	"github.com/anilonayy/medium-repos/go-concurrency/worker"
)

type BenchmarkResult struct {
	Pattern        string
	TotalTasks     int
	SuccessCount   int
	FailureCount   int
	Duration       time.Duration
	TasksPerSecond float64
	AvgTaskTime    time.Duration
}

// SimulateWork represents a typical CPU-bound or I/O-bound task
func SimulateWork(duration time.Duration) int {
	time.Sleep(duration)
	// Simulate some computation
	result := 0
	for i := range 1000 {
		result += i
	}
	return result
}

// BenchmarkWorkerPool tests the worker pool pattern
func BenchmarkWorkerPool(taskCount, workerCount int, taskDuration time.Duration) BenchmarkResult {
	log.Printf("\n=== Worker Pool Benchmark ===")
	log.Printf("Tasks: %d, Workers: %d, Task Duration: %v", taskCount, workerCount, taskDuration)

	pool := worker.NewPool(
		"benchmark-pool",
		worker.WithWorkerSize(workerCount),
		worker.WithQueueSize(taskCount),
		worker.WithTimeout(30*time.Second),
	)

	var successCount, failureCount atomic.Int32
	start := time.Now()

	for i := range taskCount {
		taskID := i
		pool.AddJob(func(ctx context.Context) error {
			duration := taskDuration + time.Duration(rand.Intn(50))*time.Millisecond
			_ = SimulateWork(duration)

			if taskID%100 == 0 {
				log.Printf("Worker Pool: Completed task %d", taskID)
			}

			successCount.Add(1)
			return nil
		})
	}

	pool.Wait()
	elapsed := time.Since(start)

	result := BenchmarkResult{
		Pattern:        "Worker Pool",
		TotalTasks:     taskCount,
		SuccessCount:   int(successCount.Load()),
		FailureCount:   int(failureCount.Load()),
		Duration:       elapsed,
		TasksPerSecond: float64(taskCount) / elapsed.Seconds(),
		AvgTaskTime:    elapsed / time.Duration(taskCount),
	}

	return result
}

// BenchmarkFanOut tests the fan-in/fan-out pattern
func BenchmarkFanOut(taskCount, workerCount int, taskDuration time.Duration) BenchmarkResult {
	log.Printf("\n=== Fan-Out Benchmark ===")
	log.Printf("Tasks: %d, Workers: %d, Task Duration: %v", taskCount, workerCount, taskDuration)

	ctx := context.Background()

	processor := func(ctx context.Context, task fanout.Task) (any, error) {
		duration := taskDuration + time.Duration(rand.Intn(50))*time.Millisecond
		_ = SimulateWork(duration)

		if task.ID%100 == 0 {
			log.Printf("Fan-Out: Completed task %d", task.ID)
		}

		return task.ID, nil
	}

	pipeline := fanout.NewPipeline(processor, fanout.WithWorkerCount(workerCount))

	// Prepare tasks
	tasks := make([]fanout.Task, taskCount)
	for i := range taskCount {
		tasks[i] = fanout.Task{
			ID:   i,
			Data: i,
		}
	}

	start := time.Now()
	results := pipeline.Process(ctx, tasks)
	elapsed := time.Since(start)

	successCount := 0
	failureCount := 0
	for _, result := range results {
		if result.Err == nil {
			successCount++
		} else {
			failureCount++
		}
	}

	result := BenchmarkResult{
		Pattern:        "Fan-In/Fan-Out",
		TotalTasks:     taskCount,
		SuccessCount:   successCount,
		FailureCount:   failureCount,
		Duration:       elapsed,
		TasksPerSecond: float64(taskCount) / elapsed.Seconds(),
		AvgTaskTime:    elapsed / time.Duration(taskCount),
	}

	return result
}

// RunComparison executes both patterns and compares results
func RunComparison(taskCount, workerCount int, taskDuration time.Duration) {
	fmt.Println("\n========== PERFORMANCE COMPARISON ==========")
	fmt.Printf("\nConfiguration:\n")
	fmt.Printf("  - Total Tasks: %d\n", taskCount)
	fmt.Printf("  - Workers: %d\n", workerCount)
	fmt.Printf("  - Task Duration: ~%v\n\n", taskDuration)

	// Run Worker Pool benchmark
	wpResult := BenchmarkWorkerPool(taskCount, workerCount, taskDuration)

	// Small pause between tests
	time.Sleep(500 * time.Millisecond)

	// Run Fan-Out benchmark
	foResult := BenchmarkFanOut(taskCount, workerCount, taskDuration)

	// Display comparison
	displayComparison(wpResult, foResult)
}

func displayComparison(wp, fo BenchmarkResult) {
	fmt.Println("\n========== RESULTS ==========")

	fmt.Printf("\n%-20s %-20s %-20s\n", "Metric", "Worker Pool", "Fan-In/Fan-Out")
	fmt.Println("------------------------------------------------------------")

	fmt.Printf("%-20s %-20d %-20d\n", "Total Tasks", wp.TotalTasks, fo.TotalTasks)
	fmt.Printf("%-20s %-20d %-20d\n", "Success Count", wp.SuccessCount, fo.SuccessCount)
	fmt.Printf("%-20s %-20d %-20d\n", "Failure Count", wp.FailureCount, fo.FailureCount)
	fmt.Printf("%-20s %-20s %-20s\n", "Total Duration", wp.Duration.Round(time.Millisecond), fo.Duration.Round(time.Millisecond))
	fmt.Printf("%-20s %-20.2f %-20.2f\n", "Tasks/Second", wp.TasksPerSecond, fo.TasksPerSecond)
	fmt.Printf("%-20s %-20s %-20s\n", "Avg Task Time", wp.AvgTaskTime.Round(time.Millisecond), fo.AvgTaskTime.Round(time.Millisecond))

	fmt.Println("\n========== ANALYSIS ==========")

	if wp.Duration < fo.Duration {
		improvement := ((fo.Duration.Seconds() - wp.Duration.Seconds()) / fo.Duration.Seconds()) * 100
		fmt.Printf("Winner: Worker Pool\n")
		fmt.Printf("Faster by: %.2f%% (%.2fs)\n", improvement, fo.Duration.Seconds()-wp.Duration.Seconds())
	} else if fo.Duration < wp.Duration {
		improvement := ((wp.Duration.Seconds() - fo.Duration.Seconds()) / wp.Duration.Seconds()) * 100
		fmt.Printf("Winner: Fan-In/Fan-Out\n")
		fmt.Printf("Faster by: %.2f%% (%.2fs)\n", improvement, wp.Duration.Seconds()-fo.Duration.Seconds())
	} else {
		fmt.Printf("Result: Both patterns performed equally\n")
	}

	fmt.Println("\nKey Observations:")
	fmt.Printf("  - Worker Pool throughput: %.2f tasks/sec\n", wp.TasksPerSecond)
	fmt.Printf("  - Fan-Out throughput: %.2f tasks/sec\n", fo.TasksPerSecond)

	if wp.TasksPerSecond > fo.TasksPerSecond {
		diff := ((wp.TasksPerSecond - fo.TasksPerSecond) / fo.TasksPerSecond) * 100
		fmt.Printf("  - Worker Pool has %.2f%% higher throughput\n", diff)
	} else if fo.TasksPerSecond > wp.TasksPerSecond {
		diff := ((fo.TasksPerSecond - wp.TasksPerSecond) / wp.TasksPerSecond) * 100
		fmt.Printf("  - Fan-Out has %.2f%% higher throughput\n", diff)
	}
}

// RunMultipleScenarios tests different workload scenarios
func RunMultipleScenarios() {
	scenarios := []struct {
		name         string
		taskCount    int
		workerCount  int
		taskDuration time.Duration
	}{
		{"Light Load", 1000, 10, 10 * time.Millisecond},
		{"Medium Load", 5000, 25, 20 * time.Millisecond},
		{"Heavy Load", 10000, 50, 50 * time.Millisecond},
		{"Very Heavy Load", 20000, 100, 100 * time.Millisecond},
	}

	for _, scenario := range scenarios {
		fmt.Printf("\n\n============================================================\n")
		fmt.Printf("Scenario: %s\n", scenario.name)
		fmt.Printf("============================================================\n")

		RunComparison(scenario.taskCount, scenario.workerCount, scenario.taskDuration)

		if scenario.name != "Very Heavy Load" {
			time.Sleep(1 * time.Second)
		}
	}

	fmt.Println("\n\n========== BENCHMARK COMPLETED ==========")
}
