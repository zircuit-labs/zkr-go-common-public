# http

The `http` package provides HTTP server functionality as a task, built on top of the Echo web framework with integrated observability, health checks, and middleware.

## Overview

This package enables you to:

- Run HTTP servers as tasks with graceful shutdown
- Integrate with DataDog APM tracing
- Include Prometheus metrics collection
- Provide standardized health check endpoints
- Use response caching middleware
- Handle panics gracefully with stack traces

## Sub-packages

### echotask

The main HTTP server implementation that wraps Echo framework as a task.

### port

Utilities for port management and configuration.

## Core Components

### EchoTask

The main HTTP server task that implements the `task.Task` interface.

```go
type EchoTask struct {
    // Internal implementation
}

// Implements task.Task interface
func (e *EchoTask) Run(ctx context.Context) error
func (e *EchoTask) Name() string
```

### RouteRegistration

Interface for registering HTTP routes:

```go
type RouteRegistration interface {
    RegisterRoutes(RouteRegistrant) error
}

type RouteRegistrant interface {
    GET(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
    POST(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
    PUT(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
    DELETE(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
    // ... other HTTP methods
}
```

## Usage

### Basic HTTP Server

```go
import (
    "log"
    "github.com/labstack/echo/v4"
    "github.com/zircuit-labs/zkr-go-common/http/echotask"
    "github.com/zircuit-labs/zkr-go-common/task"
)

type MyAPI struct{}

func (api *MyAPI) RegisterRoutes(r echotask.RouteRegistrant) error {
    r.GET("/hello", func(c echo.Context) error {
        return c.JSON(200, map[string]string{"message": "Hello, World!"})
    })

    r.POST("/users", func(c echo.Context) error {
        // Handle user creation
        return c.JSON(201, map[string]string{"status": "created"})
    })

    return nil
}

func main() {
    // Create the HTTP task
    api := &MyAPI{}
    httpTask, err := echotask.New(
        echotask.WithPort(8080),
        echotask.WithRouteRegistration(api),
    )
    if err != nil {
        panic(err)
    }

    // Run with task manager
    manager := task.NewManager()
    manager.Run(httpTask) // start as a long-running task
    if err := manager.Wait(); err != nil {
        log.Printf("Server stopped: %v", err)
    }
}
```

### Configuration Options

```go
// Basic configuration
httpTask, err := echotask.New(
    echotask.WithPort(8080),
    echotask.WithHost("0.0.0.0"),
    echotask.WithRouteRegistration(api),
)

// With custom timeouts
httpTask, err := echotask.New(
    echotask.WithPort(8080),
    echotask.WithReadTimeout(30 * time.Second),
    echotask.WithWriteTimeout(30 * time.Second),
    echotask.WithRouteRegistration(api),
)

// With custom logger
logger := slog.Default()
httpTask, err := echotask.New(
    echotask.WithPort(8080),
    echotask.WithLogger(logger),
    echotask.WithRouteRegistration(api),
)
```

### Built-in Endpoints

The HTTP server automatically includes:

- **Health Check**: `GET /healthcheck` - Returns service health status
- **Metrics**: `GET /metrics` - Prometheus metrics endpoint

### Middleware Integration

The server comes with several built-in middleware:

#### DataDog APM Tracing

Automatically enabled when DataDog environment is detected:

```go
// Tracing is automatically configured based on environment variables
// DD_SERVICE, DD_ENV, DD_VERSION, etc.
```

#### Prometheus Metrics

```go
// Metrics are automatically collected for all endpoints
// Available at /metrics endpoint
```

#### Recovery Middleware

```go
// Panics are automatically recovered and logged with stack traces
// Uses xerrors/stacktrace for enhanced error information
```

#### Cache Middleware

```go
import "github.com/zircuit-labs/zkr-go-common/http/echotask/cache"

// In your route registration
r.GET("/expensive-operation",
    cache.Middleware(cache.WithTTL(5*time.Minute))(
        func(c echo.Context) error {
            // Expensive operation here
            return c.JSON(200, result)
        },
    ),
)
```

### Health Check System

```go
import "github.com/zircuit-labs/zkr-go-common/http/echotask/healthcheck"

// Custom health check
type DatabaseHealthCheck struct {
    db *sql.DB
}

func (h *DatabaseHealthCheck) Name() string {
    return "database"
}

func (h *DatabaseHealthCheck) Check(ctx context.Context) error {
    return h.db.PingContext(ctx)
}

// Register health check
httpTask, err := echotask.New(
    echotask.WithPort(8080),
    echotask.WithHealthCheck(&DatabaseHealthCheck{db: db}),
    echotask.WithRouteRegistration(api),
)
```

## Port Management

The `port` sub-package provides utilities for port handling:

```go
import "github.com/zircuit-labs/zkr-go-common/http/port"

// Get next available port
availablePort, err := port.GetAvailable()

// Check if port is in use
inUse := port.IsInUse(8080)

// Find port in range
port, err := port.FindInRange(8000, 9000)
```

## Configuration Integration

Integrates with the `config` package for configuration management:

```go
type ServerConfig struct {
    Port         int           `koanf:"port"`
    Host         string        `koanf:"host"`
    ReadTimeout  time.Duration `koanf:"read_timeout"`
    WriteTimeout time.Duration `koanf:"write_timeout"`
}

config := &ServerConfig{}
if err := config.Load("server.toml"); err != nil {
    return err
}

httpTask, err := echotask.New(
    echotask.WithPort(config.Port),
    echotask.WithHost(config.Host),
    echotask.WithReadTimeout(config.ReadTimeout),
    echotask.WithWriteTimeout(config.WriteTimeout),
    echotask.WithRouteRegistration(api),
)
```

## Error Handling

The HTTP server provides comprehensive error handling:

```go
// Custom error handler
func customErrorHandler(err error, c echo.Context) {
    // Log error with context
    logger.Error("HTTP error",
        slog.String("method", c.Request().Method),
        slog.String("path", c.Request().URL.Path),
        slog.Any("error", err),
    )

    // Return appropriate response
    if he, ok := err.(*echo.HTTPError); ok {
        c.JSON(he.Code, map[string]interface{}{
            "error": he.Message,
        })
    } else {
        c.JSON(500, map[string]interface{}{
            "error": "Internal server error",
        })
    }
}

httpTask, err := echotask.New(
    echotask.WithPort(8080),
    echotask.WithErrorHandler(customErrorHandler),
    echotask.WithRouteRegistration(api),
)
```

## Best Practices

1. **Use RouteRegistration interface** - Keeps route definitions organized and testable
2. **Implement health checks** - Include health checks for external dependencies
3. **Use structured logging** - Log requests and errors with context
4. **Handle graceful shutdown** - The task framework handles this automatically
5. **Configure timeouts** - Set appropriate read/write timeouts for your use case
6. **Use middleware wisely** - Add caching, rate limiting, and authentication as needed
7. **Monitor metrics** - Use the `/metrics` endpoint for observability

## Dependencies

- `github.com/labstack/echo/v4` - Web framework
- `github.com/DataDog/dd-trace-go` - APM tracing
- `github.com/labstack/echo-contrib/echoprometheus` - Metrics
- Zircuit's `task` package for task management
- Zircuit's `log` package for structured logging
- Zircuit's `xerrors` packages for error handling
