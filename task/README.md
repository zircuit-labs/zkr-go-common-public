# task

The `task` package provides abstractions for managing multiple goroutines in the form of tasks, making it easy to coordinate background services and handle graceful shutdowns.

## Overview

This package simplifies the management of concurrent background services by providing:

- A standardized `Task` interface for background services
- A `Manager` for coordinating multiple tasks
- Automatic cleanup and graceful shutdown handling
- Integration with context cancellation

## Core Types

### Task Interface

```go
type Task interface {
    // Run executes the work of this service and blocks until
    // the context is cancelled or an error occurs
    Run(context.Context) error

    // Name provides a human-friendly name for use in logging
    Name() string
}
```

### Manager

The `Manager` coordinates multiple tasks, ensuring they all stop when any one of them stops or encounters an error.

```go
type Manager struct {
    // ... internal fields
}
```

## Usage

### Basic Task Implementation

```go
import (
    "context"
    "log"
    "time"
    "github.com/zircuit-labs/zkr-go-common/task"
)

// MyService implements the Task interface
type MyService struct {
    name string
}

func (s *MyService) Name() string {
    return s.name
}

func (s *MyService) Run(ctx context.Context) error {
    ticker := time.NewTicker(time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-ticker.C:
            // Do work here
            log.Println("Service is running...")
        }
    }
}
```

### Using the Manager

```go
import (
    "log"
    "github.com/zircuit-labs/zkr-go-common/task"
)

func main() {
    // Create a manager with logging
    manager := task.NewManager(task.WithLogger(logger))

    // Create tasks
    service1 := &MyService{name: "service-1"}
    service2 := &MyService{name: "service-2"}

    // Run long-running tasks (when any stops, all stop)
    manager.Run(service1)

    // Run terminable tasks (can exit without stopping others)
    manager.RunTerminable(service2)

    // Register cleanup functions
    manager.Cleanup(func() {
        log.Println("Cleaning up resources...")
    })

    // Wait for all tasks (blocks until completion or error)
    if err := manager.Wait(); err != nil {
        log.Printf("Manager stopped with error: %v", err)
    }
}
```

### Manager Options

```go
// Create manager with custom logger
manager := task.NewManager(task.WithLogger(customLogger))

// Create manager with default nil logger (no logging)
manager := task.NewManager()
```

## Sub-packages

The task package includes several specialized sub-packages:

### polling

For tasks that need to poll at regular intervals.

### ossignal

For handling OS signals in tasks.

### sighup

For handling SIGHUP signals specifically.

## Methods

### Run vs RunTerminable

The Manager provides two methods for starting tasks:

- **`Run(tasks ...Task)`** - For long-running tasks that should cause all other tasks to stop when they complete
- **`RunTerminable(tasks ...Task)`** - For tasks that can complete normally without affecting other tasks

```go
// Long-running service that should stop everything when it exits
manager.Run(httpServer, databaseService)

// One-time initialization task that can complete without stopping others
manager.RunTerminable(migrationTask, setupTask)
```

## Features

### Graceful Shutdown

- Tasks started with `Run()` automatically cancel all other tasks when they stop
- Tasks started with `RunTerminable()` can exit without affecting other tasks
- Context cancellation propagates to all running tasks
- Cleanup functions are executed in reverse order of registration

### Error Handling

- If any task returns an error, all other tasks are cancelled
- The first error encountered is returned to the caller
- Uses `errgroup` internally for coordinated error handling

### Logging

- Optional structured logging support via `slog.Logger`
- Task names are included in log messages for easier debugging
- Uses nil logger by default (no logging overhead)

## Examples

### HTTP Server Task

```go
type HTTPServerTask struct {
    server *http.Server
}

func (h *HTTPServerTask) Name() string {
    return "http-server"
}

func (h *HTTPServerTask) Run(ctx context.Context) error {
    errChan := make(chan error, 1)

    go func() {
        errChan <- h.server.ListenAndServe()
    }()

    select {
    case <-ctx.Done():
        // Graceful shutdown
        shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()
        return h.server.Shutdown(shutdownCtx)
    case err := <-errChan:
        return err
    }
}
```

### Database Connection Task

```go
type DBTask struct {
    db *sql.DB
}

func (d *DBTask) Name() string {
    return "database"
}

func (d *DBTask) Run(ctx context.Context) error {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return d.db.Close()
        case <-ticker.C:
            if err := d.db.PingContext(ctx); err != nil {
                return fmt.Errorf("database health check failed: %w", err)
            }
        }
    }
}
```

## Integration

The task package integrates well with other packages in this repository:

- Uses `calm/errgroup` for error coordination
- Supports `log` package for structured logging
- Can be used with `runner` for service standardization
- Works with `http/echotask` for HTTP services

## Best Practices

1. **Always respect context cancellation** - Check `ctx.Done()` in your task loops
2. **Provide meaningful task names** - Helps with debugging and monitoring
3. **Handle cleanup properly** - Use `Cleanup` for resource cleanup
4. **Keep tasks focused** - Each task should have a single responsibility
5. **Use timeouts for external operations** - Don't let tasks hang indefinitely
