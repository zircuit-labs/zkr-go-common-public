# xerrors

The `xerrors` package provides enhanced error handling capabilities with type-safe data attachment using generics. It includes sub-packages for stack traces, contextual information, and error classification.

## Overview

This package provides:

- **Generic error wrapping** - Attach any type-safe data to errors
- **Stack traces** - Capture and preserve call stacks with joined error support
- **Contextual information** - Add key-value context data to errors with last-entry-wins semantics
- **Error classification** - Categorize errors for better handling with hierarchical override support
- **Deep unwrapping** - Extract information from nested error chains
- **Joined error support** - All subpackages preserve structure when working with `errors.Join()`

## Core Package

### ExtendedError

```go
type ExtendedError[T any] struct {
    Data T     // The attached data of type T
    err  error // The wrapped error
}
```

### Basic Usage

```go
import "github.com/zircuit-labs/zkr-go-common/xerrors"

// Extend an error with additional data
userID := 12345
err := xerrors.Extend(userID, errors.New("operation failed"))

// Extract the data later
if id, ok := xerrors.Extract[int](err); ok {
    fmt.Printf("Operation failed for user ID: %d\n", id)
}
```

### Generic Error Wrapping

```go
// Attach different types of data
type RequestInfo struct {
    Method string
    Path   string
    UserID int64
}

requestInfo := RequestInfo{
    Method: "POST",
    Path:   "/api/users",
    UserID: 12345,
}

err := xerrors.Extend(requestInfo, errors.New("validation failed"))

// Extract the request information
if info, ok := xerrors.Extract[RequestInfo](err); ok {
    log.Printf("Request failed: %s %s (user: %d)",
        info.Method, info.Path, info.UserID)
}
```

## Sub-packages

### stacktrace

Captures and manages stack traces for errors. Supports joined errors by preserving their structure while adding stacktraces to each individual error.

```go
import "github.com/zircuit-labs/zkr-go-common/xerrors/stacktrace"

// Add stack trace to existing error
err = stacktrace.Wrap(existingErr)

// Extract stack trace
trace := stacktrace.Extract(err)
if trace != nil {
    fmt.Printf("Stack trace:\n%s\n", trace.String())
}

// Works with joined errors - preserves structure
err1 := errors.New("first error")
err2 := errors.New("second error")
joined := errors.Join(err1, err2)
wrappedJoined := stacktrace.Wrap(joined) // Both individual errors get stacktraces
```

Disable stacktraces without adjusting code:

```go
stacktrace.Disabled = true
```

### errcontext

Adds contextual key-value information to errors with **last-entry-wins** semantics. When adding context with the same key, the new value overwrites the previous value. Supports joined errors by preserving structure while adding context to each individual error.

```go
import (
    "log/slog"
    "github.com/zircuit-labs/zkr-go-common/xerrors/errcontext"
)

// Add context to an error
err = errcontext.Add(err, slog.Int64("user_id", 12345))
err = errcontext.Add(err, slog.String("operation", "create_user"))
err = errcontext.Add(err, slog.String("table", "users"))

// Duplicate keys overwrite (last-entry-wins)
err = errcontext.Add(err, slog.String("operation", "update_user")) // overwrites "create_user"

// Get all context
if ctx := errcontext.Get(err); ctx != nil {
    for _, attr := range ctx.Flatten() {
        fmt.Printf("%s: %v\n", attr.Key, attr.Value)
    }
}

// Works with joined errors - preserves structure
err1 := errcontext.Add(errors.New("error 1"), slog.String("source", "db"))
err2 := errcontext.Add(errors.New("error 2"), slog.String("source", "api"))
joined := errors.Join(err1, err2)
contextualJoined := errcontext.Add(joined, slog.String("request_id", "123"))
// Each individual error gets the request_id context
```

### errclass

Provides error classification focused on whether errors are transient (retriable) or persistent (not retriable). Features hierarchical override semantics where explicit class assignment takes precedence over child error classes. For joined errors, returns the maximum class found among all nested errors. Integrates well with the retry package.

```go
import "github.com/zircuit-labs/zkr-go-common/xerrors/errclass"

// Add classification to error
err = errclass.WrapAs(err, errclass.Transient)  // Temporary failure, can retry
err = errclass.WrapAs(err, errclass.Persistent) // Permanent failure, don't retry

// Check error classification
switch errclass.GetClass(err) {
case errclass.Nil:
    // No error - should not happen if err != nil
    return nil
case errclass.Transient:
    // Temporary error - can be retried
    scheduleRetry(err)
case errclass.Persistent:
    // Permanent error - don't retry
    logError(err)
    return err
case errclass.Panic:
    // Panic occurred
    handlePanic(err)
default:
    // Unknown error class
    handleUnknownError(err)
}

// Available error classes
// errclass.Nil        - No error
// errclass.Unknown    - Unknown error type
// errclass.Transient  - Temporary error (retriable)
// errclass.Persistent - Permanent error (not retriable)
// errclass.Panic      - Panic error

// Hierarchical override - explicit classification wins
dbErr := errclass.WrapAs(errors.New("timeout"), errclass.Transient)
validationErr := errclass.WrapAs(errors.New("invalid input"), errclass.Persistent)
joined := errors.Join(dbErr, validationErr)

// Returns Persistent (maximum severity)
class := errclass.GetClass(joined) // errclass.Persistent

// Explicit override takes precedence
overridden := errclass.WrapAs(joined, errclass.Unknown)
class = errclass.GetClass(overridden) // errclass.Unknown (not Persistent)
```

## Comprehensive Error Handling

### Building Rich Errors

```go
import (
    "github.com/zircuit-labs/zkr-go-common/xerrors"
    "github.com/zircuit-labs/zkr-go-common/xerrors/stacktrace"
    "github.com/zircuit-labs/zkr-go-common/xerrors/errcontext"
    "github.com/zircuit-labs/zkr-go-common/xerrors/errclass"
)

func processUser(userID int64) error {
    // Create base error
    err := errors.New("user processing failed")

    // Add stack trace
    err = stacktrace.Wrap(err)

    // Add contextual information
    err = errcontext.Add(err, slog.Int64("user_id", userID))
    err = errcontext.Add(err, slog.String("operation", "process_user"))
    err = errcontext.Add(err, slog.String("service", "user-service"))

    // Add classification
    err = errclass.WrapAs(err, errclass.Transient)

    // Add custom data
    metadata := UserProcessingMetadata{
        AttemptCount: 3,
        LastAttempt:  time.Now(),
        Reason:      "connection_timeout",
    }
    err = xerrors.Extend(metadata, err)

    return err
}
```

### Error Analysis

```go
func handleError(err error) {
    // Extract stack trace
    if trace, ok := stacktrace.Extract(err); ok {
        log.Printf("Stack trace:\n%s", trace.String())
    }

    // Get context
    if ctx := errcontext.Get(err); ctx != nil {
        log.Printf("Error context: %+v", ctx)

        // Find specific context value
        for _, attr := range ctx.Flatten() {
            if attr.Key == "user_id" {
                notifyUserOfError(attr.Value.String())
                break
            }
        }
    }

    // Extract custom metadata
    if metadata, ok := xerrors.Extract[UserProcessingMetadata](err); ok {
        if metadata.AttemptCount > 5 {
            // Escalate after too many attempts
            escalateError(err, metadata)
        }
    }

    // Handle by classification
    switch errclass.GetClass(err) {
    case errclass.Nil:
        // No error - should not happen if err != nil
        return
    case errclass.Transient:
        incrementTransientErrorMetric()
        scheduleRetry(err)
    case errclass.Persistent:
        incrementPersistentErrorMetric()
        returnInternalServerError(err)
    case errclass.Panic:
        incrementPanicMetric()
        returnInternalServerError(err)
    default:
        returnInternalServerError(err)
    }
}
```

## Integration with Logging

The xerrors package integrates seamlessly with the logging package:

```go
import (
    "log/slog"
    "github.com/zircuit-labs/zkr-go-common/log"
    "github.com/zircuit-labs/zkr-go-common/xerrors/errcontext"
    "github.com/zircuit-labs/zkr-go-common/xerrors/errclass"
    "github.com/zircuit-labs/zkr-go-common/xerrors/stacktrace"
)

func logError(logger *slog.Logger, err error) {
    // Base error information
    attrs := []slog.Attr{
        slog.String("error", err.Error()),
    }

    // Add stack trace if available
    if trace, ok := stacktrace.Extract(err); ok {
        attrs = append(attrs, slog.String("stacktrace", trace.String()))
    }

    // Add error classification
    if class := errclass.GetClass(err); class != errclass.Nil {
        attrs = append(attrs, slog.String("error_class", string(class)))
    }

    // Add context information
    if ctx := errcontext.Get(err); ctx != nil {
        attrs = append(attrs, ctx.Flatten()...)
    }

    logger.Error("Operation failed", attrs...)
}
```

## Error Chain Utilities

### Working with Joined Errors

```go
// Handle joined errors
err1 := errors.New("first error")
err2 := errors.New("second error")
joined := errors.Join(err1, err2)

// Unjoin to get direct children only
errs := xerrors.Unjoin(joined)
for i, err := range errs {
    fmt.Printf("Direct child %d: %s\n", i+1, err.Error())
}

// Flatten to get ALL individual errors from nested joins
nested := errors.Join(joined, errors.New("third error"))
allErrs := xerrors.Flatten(nested)
for i, err := range allErrs {
    fmt.Printf("Individual error %d: %s\n", i+1, err.Error())
}
// Output: "first error", "second error", "third error"
```

## Best Practices

1. **Add stack traces early** - Wrap errors with `stacktrace.Wrap()` as close to the source as possible
2. **Use context appropriately** - Add relevant debugging information with `errcontext`
3. **Classify errors consistently** - Use `errclass` for systematic error handling
4. **Log enhanced errors** - Take advantage of the rich error information in logs
5. **Leverage joined error support** - All subpackages preserve structure when working with `errors.Join()`
6. **Understand hierarchical semantics** - Explicit wrapping with `errclass.WrapAs()` takes precedence over child classifications
7. **Use `Flatten()` for deep analysis** - When you need all individual errors from complex joined structures
8. **Context key management** - Be aware of last-entry-wins behavior when adding context with duplicate keys

## Performance Considerations

- **Stack trace capture** has overhead - use judiciously in hot paths
- **Context addition** is lightweight - safe for frequent use
- **Error classification** is minimal overhead
- **Generic wrapping** has no runtime type overhead due to Go's generics implementation

## Advanced Features

### Joined Error Support

All xerrors subpackages have been enhanced with sophisticated joined error support:

- **Structure Preservation**: When wrapping joined errors, the original structure is preserved
- **Hierarchical Processing**: Each subpackage applies its transformations recursively to nested errors
- **Smart Extraction**: Functions like `GetClass()` intelligently traverse joined error trees

### Context Management

The `errcontext` package uses map-based storage with last-entry-wins semantics:

- Duplicate keys are overwritten with the latest value
- Efficient storage and retrieval of contextual information
- Seamless integration with structured logging

### Classification Override

The `errclass` package implements hierarchical override semantics:

- Explicit class assignments take precedence over child error classes
- Maximum class detection across joined error trees
- Smart handling of wrapped joined errors

## Dependencies

The xerrors package and its sub-packages have minimal dependencies:

- Go standard library (`errors`, `fmt`, `runtime` for stack traces, `log/slog` for structured logging)
- Internal dependencies between sub-packages (e.g., errcontext uses base xerrors)

This makes it safe to use as a foundational error handling library without introducing heavy dependencies.
