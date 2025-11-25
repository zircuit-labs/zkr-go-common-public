package log_test

import (
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/zircuit-labs/zkr-go-common/log"
	"github.com/zircuit-labs/zkr-go-common/xerrors/errclass"
	"github.com/zircuit-labs/zkr-go-common/xerrors/errcontext"
	"github.com/zircuit-labs/zkr-go-common/xerrors/stacktrace"
)

// Test helper types
type customError struct {
	msg string
}

func (e customError) Error() string {
	return e.msg
}

func TestNewLoggableErrorHandler_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("NonErrorAttribute", func(t *testing.T) {
		t.Parallel()
		logger, buf := newTestLogger(t)

		// Log non-error attribute with "error" key
		logger.Info("test", slog.String("error", "not an error"))

		expectedLog := `{
			"time": "2021-01-01T00:00:00Z",
			"level": "info",
			"msg": "test",
			"error": "not an error",
			"service": "test-service"
		}`

		actualLogJSON := buf.String()
		cleanedActual := comparableLog(actualLogJSON)
		assert.JSONEq(t, expectedLog, cleanedActual)
	})

	t.Run("OtherAttributeTypes", func(t *testing.T) {
		t.Parallel()
		logger, buf := newTestLogger(t)

		logger.Info("test",
			slog.Int("count", 42),
			slog.Bool("flag", true),
			slog.Duration("timeout", 5*time.Second),
		)

		expectedLog := `{
			"time": "2021-01-01T00:00:00Z",
			"level": "info",
			"msg": "test",
			"count": 42,
			"flag": true,
			"timeout": 5000000000,
			"service": "test-service"
		}`

		actualLogJSON := buf.String()
		cleanedActual := comparableLog(actualLogJSON)
		assert.JSONEq(t, expectedLog, cleanedActual)
	})
}

func TestLoggableError_ComplexChains(t *testing.T) {
	t.Parallel()

	t.Run("MultipleWrappedErrors", func(t *testing.T) {
		t.Parallel()
		logger, buf := newTestLogger(t)

		// Create a complex error chain
		baseErr := errors.New("root cause")
		wrappedErr := stacktrace.Wrap(baseErr)
		contextErr := errcontext.Add(wrappedErr, slog.String("operation", "file_read"))
		classErr := errclass.WrapAs(contextErr, errclass.Transient)
		finalErr := stacktrace.Wrap(classErr) // Wrap again

		logger.Error("complex error chain", log.ErrAttr(finalErr))

		output := buf.String()
		cleanedActual := comparableLog(output)

		// Should contain the base error message
		assert.Contains(t, cleanedActual, "root cause")
		// Should contain context information
		assert.Contains(t, cleanedActual, "operation")
		assert.Contains(t, cleanedActual, "file_read")
		// Should contain error class
		assert.Contains(t, cleanedActual, "transient")
		// Should contain stack trace information
		assert.Contains(t, cleanedActual, "github_com/zircuit-labs/zkr-go-common/xerrors_ExtendedError")
	})

	t.Run("ErrorWithDurationInContext", func(t *testing.T) {
		t.Parallel()
		logger, buf := newTestLogger(t)

		baseErr := errors.New("timeout error")
		contextErr := errcontext.Add(baseErr,
			slog.Duration("timeout", 30*time.Second),
			slog.String("endpoint", "api.example.com"),
		)

		logger.Error("request failed", log.ErrAttr(contextErr))

		expectedLog := `{
			"time": "2021-01-01T00:00:00Z",
			"level": "error",
			"error": "timeout error",
			"error_detail": {
				"github_com/zircuit-labs/zkr-go-common/xerrors_ExtendedError[github_com/zircuit-labs/zkr-go-common/xerrors/errcontext_Context]": {
					"timeout": "30s",
					"endpoint": "api.example.com"
				}
			},
			"msg": "request failed",
			"service": "test-service"
		}`

		actualLogJSON := buf.String()
		cleanedActual := comparableLog(actualLogJSON)
		assert.JSONEq(t, expectedLog, cleanedActual)
	})
}

func TestJoinedErrors_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("SingleErrorInJoin", func(t *testing.T) {
		t.Parallel()
		logger, buf := newTestLogger(t)

		// errors.Join with single error should be treated as regular error
		singleErr := errors.New("single error")
		joinedErr := errors.Join(singleErr)

		logger.Error("single joined error", log.ErrAttr(joinedErr))

		output := buf.String()
		cleanedActual := comparableLog(output)

		// Should contain the error message
		assert.Contains(t, cleanedActual, "single error")
		// Should NOT have joined error format (no "errors" array)
		assert.NotContains(t, cleanedActual, `"errors":[`)
	})

	t.Run("JoinedErrorsWithDifferentTypes", func(t *testing.T) {
		t.Parallel()
		logger, buf := newTestLogger(t)

		// Mix of simple and extended errors
		simpleErr := errors.New("simple error")
		extendedErr := errclass.WrapAs(
			errcontext.Add(
				stacktrace.Wrap(errors.New("extended error")),
				slog.String("component", "database"),
			),
			errclass.Persistent,
		)

		joinedErr := errors.Join(simpleErr, extendedErr)
		logger.Error("mixed error types", log.ErrAttr(joinedErr))

		output := buf.String()
		cleanedActual := comparableLog(output)

		// Should contain both error messages
		assert.Contains(t, cleanedActual, "simple error")
		assert.Contains(t, cleanedActual, "extended error")
		// Should have joined error format
		assert.Contains(t, cleanedActual, `"errors":[`)
		// Should contain extended error details for the second error
		assert.Contains(t, cleanedActual, "component")
		assert.Contains(t, cleanedActual, "database")
		assert.Contains(t, cleanedActual, "persistent")
	})

	t.Run("NilErrorsInJoin", func(t *testing.T) {
		t.Parallel()
		logger, buf := newTestLogger(t)

		// errors.Join filters out nil errors
		err1 := errors.New("real error")
		joinedErr := errors.Join(err1, nil, nil)

		logger.Error("joined with nils", log.ErrAttr(joinedErr))

		output := buf.String()
		cleanedActual := comparableLog(output)

		// Should only contain the non-nil error
		assert.Contains(t, cleanedActual, "real error")
		// Should be treated as single error since nils are filtered out
		assert.NotContains(t, cleanedActual, `"errors":[`)
	})
}

func TestSlogValueToAny_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("NestedLogValuer", func(t *testing.T) {
		t.Parallel()
		logger, buf := newTestLogger(t)

		// Create an error that implements LogValuer and returns another LogValuer
		baseErr := errors.New("nested logvaluer")
		contextErr := errcontext.Add(baseErr, slog.String("level", "nested"))
		stackErr := stacktrace.Wrap(contextErr)

		logger.Error("nested logvaluer test", log.ErrAttr(stackErr))

		output := buf.String()
		cleanedActual := comparableLog(output)

		// Should properly handle nested LogValuer implementations
		assert.Contains(t, cleanedActual, "nested logvaluer")
		assert.Contains(t, cleanedActual, "level")
		assert.Contains(t, cleanedActual, "nested")
	})

	t.Run("ErrorWithTimeValues", func(t *testing.T) {
		t.Parallel()
		logger, buf := newTestLogger(t)

		baseErr := errors.New("time error")
		timeValue := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
		contextErr := errcontext.Add(baseErr,
			slog.Time("occurred_at", timeValue),
			slog.String("timezone", "UTC"),
		)

		logger.Error("time value test", log.ErrAttr(contextErr))

		output := buf.String()
		cleanedActual := comparableLog(output)

		// Should contain the time value and other context
		assert.Contains(t, cleanedActual, "time error")
		assert.Contains(t, cleanedActual, "occurred_at")
		assert.Contains(t, cleanedActual, "timezone")
		assert.Contains(t, cleanedActual, "UTC")
	})
}

func TestGetTypePath_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("CustomStruct", func(t *testing.T) {
		t.Parallel()
		logger, buf := newTestLogger(t)

		customErr := customError{msg: "custom error"}

		logger.Error("custom struct error", log.ErrAttr(customErr))

		output := buf.String()
		cleanedActual := comparableLog(output)

		// Should handle custom types gracefully
		assert.Contains(t, cleanedActual, "custom error")
	})

	t.Run("PointerToError", func(t *testing.T) {
		t.Parallel()
		logger, buf := newTestLogger(t)

		// Test with pointer to error type
		baseErr := errors.New("pointer error")
		contextErr := errcontext.Add(baseErr, slog.String("type", "pointer"))

		logger.Error("pointer error test", log.ErrAttr(contextErr))

		output := buf.String()
		cleanedActual := comparableLog(output)

		// Should handle pointer types correctly
		assert.Contains(t, cleanedActual, "pointer error")
		assert.Contains(t, cleanedActual, "type")
		assert.Contains(t, cleanedActual, "pointer")
	})
}
