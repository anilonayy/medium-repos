# Go Concurrency Patterns

Practical examples of Go concurrency patterns for real-world job processing.

## Patterns

### 1. Worker Pool
Fixed pool of workers processing jobs from a queue.

**Features:**
- Configurable worker count and queue size
- Job timeout support
- Graceful shutdown
- Panic recovery
- Custom context support

### 2. Fan-In/Fan-Out
Distribute work across multiple goroutines and collect results.

**Features:**
- Dynamic work distribution
- Result aggregation
- Streaming support
- Channel multiplexing
- Channel splitting

## Quick Start

### Worker Pool Example

```go
pool := worker.NewPool(
    "my-pool",
    worker.WithWorkerSize(5),
    worker.WithQueueSize(100),
    worker.WithTimeout(10*time.Second),
)

for i := 0; i < 10; i++ {
    jobID := i
    pool.AddJob(func(ctx context.Context) error {
        // Do work here
        log.Printf("Processing job %d", jobID)
        time.Sleep(1 * time.Second)
        return nil
    })
}

pool.Wait()
```

### Fan-In/Fan-Out Example

```go
processor := func(ctx context.Context, task fanout.Task) (interface{}, error) {
    result := fmt.Sprintf("Processed: %v", task.Data)
    return result, nil
}

pipeline := fanout.NewPipeline(processor, fanout.WithWorkerCount(5))

tasks := []fanout.Task{
    {ID: 1, Data: "task-1"},
    {ID: 2, Data: "task-2"},
    {ID: 3, Data: "task-3"},
}

results := pipeline.Process(ctx, tasks)

for _, result := range results {
    if result.Err != nil {
        log.Printf("Task %d failed: %v", result.TaskID, result.Err)
    } else {
        log.Printf("Task %d: %v", result.TaskID, result.Output)
    }
}
```

### Streaming Example

```go
taskChan := make(chan fanout.Task)

go func() {
    defer close(taskChan)
    for i := 0; i < 100; i++ {
        taskChan <- fanout.Task{ID: i, Data: i}
    }
}()

resultChan := pipeline.ProcessStream(ctx, taskChan)

for result := range resultChan {
    log.Printf("Result: %v", result.Output)
}
```

## Configuration

### Worker Pool Options

```go
worker.WithWorkerSize(10)                    // Number of concurrent workers
worker.WithQueueSize(100)                    // Job queue buffer size
worker.WithTimeout(30 * time.Second)         // Max execution time per job
worker.WithDelay(100 * time.Millisecond)     // Delay between job executions
```

### Fan-Out Options

```go
fanout.WithWorkerCount(10)  // Number of concurrent workers
```

## Advanced Usage

### Graceful Shutdown

```go
shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

if err := pool.Shutdown(shutdownCtx); err != nil {
    log.Printf("Shutdown error: %v", err)
}
```

### Job with Custom Context

```go
customCtx := context.WithValue(context.Background(), "key", "value")
pool.AddJobWithContext(customCtx, func(ctx context.Context) error {
    value := ctx.Value("key")
    // Use the custom context
    return nil
})
```

## Running Examples

```bash
cd go-concurrency
go run main.go
```

**Menu options:**
1. Worker Pool Pattern demo
2. Fan-In/Fan-Out Pattern demo
3. Performance Comparison (Light: 1K tasks, Medium: 5K tasks, Heavy: 10K tasks, Very Heavy: 20K tasks)
4. Multiple Scenario Benchmark (runs all load scenarios)

Press `Ctrl+C` to test graceful shutdown in worker pool demo.

## When to Use

### Worker Pool
- Need controlled resource usage
- Independent jobs that can be processed in any order
- Long-running services with continuous job processing
- Want to limit concurrency

### Fan-In/Fan-Out
- Processing large batches quickly
- Tasks can be split and processed independently
- Need to aggregate results from multiple sources
- One-time batch operations

## Performance Notes

Both patterns perform similarly with typical differences under 10%. Choose based on your use case:

**Worker Pool:**
- Consistent overhead from channel buffering
- Better for continuous job processing
- Lower memory footprint for long-running services

**Fan-In/Fan-Out:**
- Slightly faster for batch processing (5-10% typical)
- Better throughput for parallel task distribution
- More suitable for one-time batch operations

## License

MIT