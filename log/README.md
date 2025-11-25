# log

Zircuit's Go logger that uses `slog.Logger` to output structured JSON logs with special parsing for errors, particularly those wrapped through the `xerrors` package.

## Overview

This package provides:

- Structured JSON logging with Go's standard `slog.Logger`
- Automatic error enrichment with stack traces, context, and classification
- Support for joined errors (flattened output with per-leaf detail)
- Service identity management via the `identity` subpackage
- Testing support using t.Output() from Go 1.25
- Configurable log levels
- Support for both JSON and text output styles

**NOTE on JSON log output**: Dots in error detail keys are replaced with underscores for better JSON parser compatibility

## Features

### Structured JSON Output

All logs are output as structured JSON for easy parsing and analysis:

```json
{
  "level": "info",
  "service": "my-service-abc123",
  "msg": "Server started",
  "time": "2023-01-01T12:00:00Z"
}
```

### Error Enhancement

Automatically extracts and includes error information:

- Stack traces from `xerrors/stacktrace`
- Error context from `xerrors/errcontext`
- Error classification from `xerrors/errclass`

## Usage

### Basic Setup

```go
import (
    "log/slog"
    "github.com/zircuit-labs/zkr-go-common/log"
    "github.com/zircuit-labs/zkr-go-common/log/identity"
)

func main() {
    // Optional: set log level
    err := log.SetLogLevel("info")
    // check error

    // Optional: set service identity
    identity.SetServiceName("my-service")
    serviceName, instanceID := identity.WhoAmI()

    // Create a logger with service identity
    logger, err := log.NewLogger(
        log.WithServiceName(serviceName),
        log.WithInstanceID(instanceID),
    )
    if err != nil {
        panic(err)
    }

    logger.Info("Service starting")
    logger.Error("Something went wrong", slog.String("component", "database"))
}
```

### Service Identity

```go
import (
    "github.com/zircuit-labs/zkr-go-common/log/identity"
)

// Set service name once (protected by sync.Once)
identity.SetServiceName("api-server")

// Get service identity
serviceName, instanceID := identity.WhoAmI()
fmt.Printf("Service: %s, Instance: %s\n", serviceName, instanceID)

// Or use with logger
logger, err := log.NewLogger(
    log.WithServiceName(serviceName),
    log.WithInstanceID(instanceID),
)
// check err
```

### Log Levels

```go
// Set log level by string
log.SetLogLevel("debug")  // debug, info, warn, error
log.SetLogLevel("info")
log.SetLogLevel("warn")
log.SetLogLevel("error")

// Get current log level
level := log.GetLogLevel()
```

### Error Logging with Additional Detail

```go
import (
    "errors"
    "log/slog"

    "github.com/zircuit-labs/zkr-go-common/xerrors/stacktrace"
    "github.com/zircuit-labs/zkr-go-common/xerrors/errcontext"
    "github.com/zircuit-labs/zkr-go-common/xerrors/errclass"
)

func processData() error {
    // Create error with stack trace
    err := stacktrace.Wrap(errors.New("database connection failed"))

    // Add context
    err = errcontext.Add(err,
        slog.String("table", "users"),
        slog.String("operation", "select"),
    )

    // Add classification
    err = errclass.WrapAs(err, errclass.Transient)

    return err
}

func processMultipleOperations() error {
    // Create individual errors with context
    err1 := errcontext.Add(
        errclass.WrapAs(errors.New("validation failed"), errclass.Persistent),
        slog.String("field", "email"),
    )

    err2 := errcontext.Add(
        errclass.WrapAs(errors.New("database timeout"), errclass.Transient),
        slog.String("table", "users"),
    )

    // Join errors - output is flattened for readability with per-leaf detail preserved
    return errors.Join(err1, err2)
}

func main() {
    logger, err := log.NewLogger()
    // check err

    if err := processData(); err != nil {
        // Error information is automatically extracted and logged
        logger.Error("Processing failed", log.ErrAttr(err))
        // Output includes: stack trace, context data, and error class
    }

    if err := processMultipleOperations(); err != nil {
        // Joined errors are flattened in output; per-leaf detail is preserved
        logger.Error("Multiple operations failed", log.ErrAttr(err))
        // Output includes context and classification from all individual errors
    }
}
```

### Testing Support

```go
import (
    "testing"
    "github.com/zircuit-labs/zkr-go-common/log"
)

func TestMyFunction(t *testing.T) {
    // Get a test logger that outputs to t.Log
    logger := log.NewTestLogger(t)

    // Use logger in your test
    logger.Info("Test starting")

    // Test your code...
}
```

### Nil Logger

For cases where logging is not needed:

```go
// Create a nil logger (no output)
logger := log.NewNilLogger()

// Safe to use, but produces no output
logger.Info("This won't be logged")
```

## Configuration

### Environment Variables

This package does not automatically read environment variables. If desired, wire them in your app:

```go
if err := log.SetLogLevel(os.Getenv("LOG_LEVEL")); err != nil {
    panic(err)
}
logger, err := log.NewLogger(
    log.WithServiceName(os.Getenv("SERVICE_NAME")),
)
```

### Programmatic Configuration

```go
import (
    "github.com/zircuit-labs/zkr-go-common/log"
    "github.com/zircuit-labs/zkr-go-common/log/identity"
    "github.com/zircuit-labs/zkr-go-common/version"
)

// Configure service identity (can only be set once)
identity.SetServiceName("payment-service")

// Configure log level
log.SetLogLevel("info")


// Create logger with custom options
serviceName, instanceID := identity.WhoAmI()
logger, err := log.NewLogger(
    log.WithServiceName(serviceName),
    log.WithInstanceID(instanceID),
    log.WithVersion(&version.Info),
    log.WithLogStyle(log.LogStyleJSON), // or LogStyleText
)
// check error
```

## Integration with xerrors

The logger automatically extracts information from any error class that implements `slog.LogValuer`, such as those in the `xerrors` package.

```go
import (
    "errors"
    "fmt"
    "log/slog"
    "github.com/zircuit-labs/zkr-go-common/log"
    "github.com/zircuit-labs/zkr-go-common/xerrors/errclass"
    "github.com/zircuit-labs/zkr-go-common/xerrors/errcontext"
    "github.com/zircuit-labs/zkr-go-common/xerrors/stacktrace"
)

func example() {
    logger, _ := log.NewLogger()

    // Create a rich error
    err := fmt.Errorf("operation failed")
    err = stacktrace.Wrap(err)
    err = errcontext.Add(err, slog.Int64("user_id", 12345))
    err = errclass.WrapAs(err, errclass.Transient)

    // Log it - all error information is automatically extracted
    logger.Error("Request failed", log.ErrAttr(err))
}

func exampleJoinedErrors() {
    logger, _ := log.NewLogger()

    // Create multiple rich errors
    authErr := stacktrace.Wrap(
        errcontext.Add(
            errclass.WrapAs(errors.New("authentication failed"), errclass.Persistent),
            slog.String("service", "auth"),
            slog.Int64("user_id", 12345),
        ),
    )

    dbErr := stacktrace.Wrap(
        errcontext.Add(
            errclass.WrapAs(errors.New("database connection failed"), errclass.Transient),
            slog.String("service", "database"),
            slog.String("table", "users"),
        ),
    )

    // Join errors - flattened in output with all error information preserved
    combinedErr := errors.Join(authErr, dbErr)

    // Log joined error - extracts information from all individual errors
    logger.Error("Request failed", log.ErrAttr(combinedErr))
    // Output includes stack traces, context data, and error classes from both errors
}
```

## Output Format

### Standard Log Entry

```json
{
  "level": "info",
  "time": "2021-01-01T00:00:00Z",
  "msg": "operation failed",
  "user_id": 12345
}
```

### Error Log Entry with xerrors

```json
{
  "time": "2021-01-01T00:00:00Z",
  "level": "error",
  "error": "operation failed",
  "error_detail": {
    "github_com/zircuit-labs/zkr-go-common/xerrors_ExtendedError[github_com/zircuit-labs/zkr-go-common/xerrors/errclass_Class]": {
      "class": "transient"
    },
    "github_com/zircuit-labs/zkr-go-common/xerrors_ExtendedError[github_com/zircuit-labs/zkr-go-common/xerrors/errcontext_Context]": {
      "user_id": 12345
    },
    "github_com/zircuit-labs/zkr-go-common/xerrors_ExtendedError[github_com/zircuit-labs/zkr-go-common/xerrors/stacktrace_StackTrace]": [
      {
        "func": "github.com/zircuit-labs/zkr-go-common/log_test.TestErrorLog",
        "line": 45
      }
    ]
  },
  "msg": "Request failed",
  "service": "my-service"
}
```

## Constants

The package defines keys used in structured logging:

```go
const (
    ErrorKey = "error" // Key used by ErrAttr
)
```

## Best Practices

1. **Use structured fields** - Prefer `slog.String()`, `slog.Int()` etc. over string formatting
2. **Log at appropriate levels** - Use debug for development, info for important events, warn for recoverable issues, error for failures
3. **Include context** - Add relevant context fields to help with debugging
4. **Use xerrors integration** - Leverage automatic error enrichment for better observability

### Joined Error Output Format

When errors are joined using `errors.Join()`, the logger outputs a flattened view with backward-compatible fields:

```json
{
  "level": "error",
  "time": "2021-01-01T00:00:00Z",
  "error": "auth failed; db connection failed",
  "errors": ["auth failed", "db connection failed"],
  "error_detail": {
    "error_0": {
      "error": "auth failed",
      "error_detail": {
        "github_com/zircuit-labs/zkr-go-common/xerrors_ExtendedError[...]": "..."
      }
    },
    "error_1": {
      "error": "db connection failed",
      "error_detail": {
        "github_com/zircuit-labs/zkr-go-common/xerrors_ExtendedError[...]": "..."
      }
    }
  },
  "msg": "Multiple operations failed",
  "service": "my-service"
}
```

- `error` field contains semicolon-separated error messages (backward compatibility)
- `errors` field contains array of individual error messages (programmatic access)
- `error_detail` preserves all extended information from each individual error

### Special Considerations

When implementing new errors that support `errors.Join`, consider both usage and logging.

The logger flattens joined errors in JSON output for readability while preserving each leaf error's detail (stacktrace/context/class) under `error_detail`. Applying a class to a parent joined error may not appear in the flattened output; prefer setting classes on the leaf errors.

The `stacktrace` and `errcontext` errors do not share these problems since the information carried there is only attached to the leaf errors.

## Dependencies

- `log/slog` - Go's structured logging
- `github.com/rs/xid` - Unique ID generation
- Zircuit's `xerrors` packages for error handling
