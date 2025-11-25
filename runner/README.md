# runner

Boilerplate abstraction for standardized services that provides common initialization patterns, configuration management, logging setup, DataDog integration, and graceful shutdown handling.

## Overview

The `runner` package eliminates boilerplate code from service `main()` functions by providing:

- Standardized service initialization
- Configuration loading and management
- Structured logging setup with service identity
- DataDog APM tracing and profiling integration
- Signal handling for graceful shutdown
- Singleton service support with distributed locking
- Panic recovery and proper exit codes

## Core Types

### Runnable Function

```go
type Runnable func(cfg *config.Configuration, tm Runner, logger *slog.Logger) error
```

Your service logic is implemented as a `Runnable` function that receives:

- `cfg` - Loaded configuration
- `tm` - Task manager for running background services
- `logger` - Configured structured logger

### Runner Interface

```go
type Runner interface {
    Run(tasks ...task.Task)           // Run tasks that stop when any task stops
    RunTerminable(tasks ...task.Task) // Run tasks that can be terminated independently
    Cleanup(f func())                 // Register cleanup functions
    Context() context.Context         // Get the cancellation context
}
```

## Usage

### Basic Service

```go
package main

import (
    "embed"
    "log/slog"

    "github.com/zircuit-labs/zkr-go-common/config"
    "github.com/zircuit-labs/zkr-go-common/runner"
)

//go:embed config
var configFS embed.FS

func main() {
    runner.Run("my-service", configFS, runService)
}

func runService(cfg *config.Configuration, tm runner.Runner, logger *slog.Logger) error {
    logger.Info("Service starting")

    // Your service logic here
    // Use tm.Run() to start background tasks
    // Use cfg to access configuration

    logger.Info("Service initialized successfully")
    return nil
}
```

### Singleton Service

For services that should only run one instance at a time:

```go
func main() {
    runner.Run("unique-service", configFS, runService, runner.AsSingleton())
}

func runService(cfg *config.Configuration, tm runner.Runner, logger *slog.Logger) error {
    // This will only run if this instance acquires the distributed lock
    logger.Info("Running as singleton instance")

    // Singleton service logic here
    return nil
}
```

### Service Name Configuration

```go
// Use provided name instead of DD_SERVICE environment variable
func main() {
    runner.Run("my-service", configFS, runService, runner.UseProvidedName())
}

// Or combine options
func main() {
    runner.Run("unique-service", configFS, runService,
        runner.AsSingleton(),
        runner.UseProvidedName(),
    )
}
```

## Configuration

### Runner Configuration

The runner looks for configuration under the `runner` path in the embedded config file `./data/settings.toml`:

```toml
# data/settings.toml
[runner]
log_level = "info"  # debug, info, warn, error
```

### Environment Variables

Key environment variables the runner recognizes:

```bash
# Service identification (used by DataDog and logging)
export DD_SERVICE=my-service
export DD_ENV=production
export DD_VERSION=1.0.0

# DataDog APM (enables profiling and tracing when set)
export DD_APM_ENABLED=true

# Logging
export LOG_LEVEL=info
```

## Features

### DataDog Integration

When `DD_APM_ENABLED` is set, the runner automatically:

- Starts DataDog profiler with CPU, heap, block, mutex, and goroutine profiles
- Initializes APM tracing
- Tags profiles with service info, version, and git commit

```go
// Profiler configuration includes:
profiler.WithService(identity.ServiceName)
profiler.WithVersion(version.Info.Version)
profiler.WithTags(
    fmt.Sprintf("instance:%s", identity.InstanceID),
    fmt.Sprintf("git_commit:%s", version.Info.GitCommit),
)
```

### Panic Recovery

The runner wraps your service in panic recovery. Panics are caught and logged with stack traces, and the service exits with code 2 (standard Go panic exit code).

### Signal Handling

Automatically handles OS signals for graceful shutdown:

- SIGTERM, SIGINT trigger graceful shutdown

### Singleton Support

For services that should run only one instance:

```go
runner.Run("worker", configFS, runService, runner.AsSingleton())
```

Uses NATS-based distributed locking:

- Acquires lock before starting service
- Releases lock on shutdown
- Uses fencing to prevent split-brain scenarios
- Automatically extends lock while running

## Error Handling and Exit Codes

The runner uses specific exit codes:

- **0** - Normal exit
- **1** - Error exit (service returned an error)
- **2** - Panic exit (service panicked)


## Task Management

The provided `Runner` interface simplifies task management:

```go
func runService(cfg *config.Configuration, tm runner.Runner, logger *slog.Logger) error {
    // Run persistent tasks (stop when any stops)
    httpServer := createHTTPServer()
    worker := createWorker()
    tm.Run(httpServer, worker)

    // Run terminable tasks (can be stopped independently)
    healthCheck := createHealthCheck()
    tm.RunTerminable(healthCheck)

    // Register cleanup
    tm.Cleanup(func() {
        logger.Info("Cleaning up resources")
        closeDatabase()
    })

    return nil
}
```

## Best Practices

1. **Embed configuration** - Use `embed.FS` for configuration files
2. **Use structured logging** - The provided logger is pre-configured
3. **Handle errors properly** - Return errors from your Runnable for proper exit codes
4. **Register cleanup** - Use `tm.Cleanup()` for resource cleanup
5. **Leverage configuration** - Use the config system for all settings
6. **Monitor with DataDog** - Enable APM in production environments
7. **Use singleton carefully** - Only for services that truly need single instances

## Dependencies

- Zircuit's `config` package for configuration management
- Zircuit's `log` package for structured logging
- Zircuit's `task` package for task management
- Zircuit's `version` package for version information
- Zircuit's `xerrors` packages for error handling
- Zircuit's `calm` package for panic recovery
- DataDog Go libraries for APM and profiling