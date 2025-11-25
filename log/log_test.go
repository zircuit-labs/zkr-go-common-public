package log_test

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zircuit-labs/zkr-go-common/log"
	"github.com/zircuit-labs/zkr-go-common/xerrors/errclass"
	"github.com/zircuit-labs/zkr-go-common/xerrors/errcontext"
	"github.com/zircuit-labs/zkr-go-common/xerrors/stacktrace"
)

var (
	sourceRegex = regexp.MustCompile(`,?"source":"[^"]+"`)
	lineRegex   = regexp.MustCompile(`"line":\d+`)
	errTest     = errors.New("test error")
)

// newTestLogger creates a test logger with consistent options
func newTestLogger(t *testing.T) (*slog.Logger, *bytes.Buffer) {
	t.Helper()
	var buf bytes.Buffer
	logger, err := log.NewLogger(log.WithWriter(&buf), log.WithServiceName("test-service"))
	require.NoError(t, err)
	return logger, &buf
}

// Helper functions to create a deeper call stack for testing
// This simulates a more realistic error scenario where an error
// originates from deep within business logic
func createComplexError() error {
	return processRequest()
}

func processRequest() error {
	return validateInput()
}

func validateInput() error {
	return businessLogic()
}

func businessLogic() error {
	return stacktrace.Wrap(errTest)
}

// TestErrorLog validates that an error created through a deeper call stack is logged correctly.
// This demonstrates how complex errors with stacktraces, classes, and context
// can be created deep in business logic and bubble up while preserving all their information.
// NOTE: The line numbers are not being tested here, so we set them to 0 for maintainability of this test.
// In practice, the real line numbers will appear. This is tested in the stacktrace package.
func TestErrorLog(t *testing.T) {
	t.Parallel()

	// Create a test logger
	logger, buf := newTestLogger(t)

	// create a test error with all the bells and whistles through a deeper call stack
	// This demonstrates how errors can be created deep in the call stack and bubble up
	// while preserving all their extended information (class, context, stacktrace)
	err := createComplexError()
	err = errclass.WrapAs(err, errclass.Transient)
	err = errcontext.Add(
		err,
		slog.Bool("example_bool", true),
		slog.Int("example_int", 42),
		slog.Duration("example_dur", 5*time.Minute+15*time.Second),
	)

	// log the error
	logger.Error("example error log", log.ErrAttr(err))

	expectedLog := `
	{
		"time":"2021-01-01T00:00:00Z",
		"level": "error",
		"error": "test error",
		"error_detail": {
			"github_com/zircuit-labs/zkr-go-common/xerrors_ExtendedError[github_com/zircuit-labs/zkr-go-common/xerrors/errclass_Class]": {
				"class": "transient"
			},
			"github_com/zircuit-labs/zkr-go-common/xerrors_ExtendedError[github_com/zircuit-labs/zkr-go-common/xerrors/errcontext_Context]": {
				"example_bool": true,
				"example_int": 42,
				"example_dur": "5m15s"
			},
			"github_com/zircuit-labs/zkr-go-common/xerrors_ExtendedError[github_com/zircuit-labs/zkr-go-common/xerrors/stacktrace_StackTrace]": [
				{
					"func": "github.com/zircuit-labs/zkr-go-common/log_test.businessLogic",
					"line": 0
				},
				{
					"func": "github.com/zircuit-labs/zkr-go-common/log_test.validateInput",
					"line": 0
				},
				{
					"func": "github.com/zircuit-labs/zkr-go-common/log_test.processRequest",
					"line": 0
				},
				{
					"func": "github.com/zircuit-labs/zkr-go-common/log_test.createComplexError",
					"line": 0
				},
				{
					"func": "github.com/zircuit-labs/zkr-go-common/log_test.TestErrorLog",
					"line": 0
				}
			]
		},
		"msg": "example error log",
		"service": "test-service"
	}
	`
	actualLogJSON := buf.String()
	cleanedActual := comparableLog(actualLogJSON)
	assert.JSONEq(t, expectedLog, cleanedActual)
}

// TestLogErrorSimple validates that a simple error is logged correctly.
func TestLogErrorSimple(t *testing.T) {
	t.Parallel()

	// Create a test logger
	logger, buf := newTestLogger(t)

	// log a basic error
	logger.Error("example error log", log.ErrAttr(errTest))

	expectedLog := `
	{
		"time":"2021-01-01T00:00:00Z",
		"level": "error",
		"error": "test error",
		"msg": "example error log",
		"service": "test-service"
	}
	`
	actualLogJSON := buf.String()
	cleanedActual := comparableLog(actualLogJSON)
	assert.JSONEq(t, expectedLog, cleanedActual)
}

// TestErrorLogNil validates that a nil error is logged correctly.
func TestErrorLogNil(t *testing.T) {
	t.Parallel()

	// Create a test logger
	logger, buf := newTestLogger(t)

	// log a nil error
	logger.Error("example error log", log.ErrAttr(nil))

	expectedLog := `
	{
		"time":"2021-01-01T00:00:00Z",
		"level": "error",
		"error": null,
		"msg": "example error log",
		"service": "test-service"
	}
	`
	actualLogJSON := buf.String()
	cleanedActual := comparableLog(actualLogJSON)
	assert.JSONEq(t, expectedLog, cleanedActual)
}

func TestLogErrorWrapped(t *testing.T) {
	t.Parallel()

	// Create a test logger
	logger, buf := newTestLogger(t)

	// Create a test error that has no extra bells and whistles.
	err := fmt.Errorf("failed to open file %s: %w", "example.txt", errTest)
	err = fmt.Errorf("further wrapped: %w", err)

	// log the error
	logger.Error("example error log", log.ErrAttr(err))

	expectedLog := `
	{
		"time":"2021-01-01T00:00:00Z",
		"level": "error",
		"error": "further wrapped: failed to open file example.txt: test error",
		"msg": "example error log",
		"service": "test-service"
	}
	`
	actualLogJSON := buf.String()
	cleanedActual := comparableLog(actualLogJSON)
	assert.JSONEq(t, expectedLog, cleanedActual)
}

// TestErrorLogNotAnError validates that a non-error with error key is logged correctly.
func TestErrorLogNotAnError(t *testing.T) {
	t.Parallel()

	// Create a test logger
	logger, buf := newTestLogger(t)

	// log a non-error value using the error key
	logger.Error("example error log", slog.Int("error", 42))

	expectedLog := `
	{
		"time":"2021-01-01T00:00:00Z",
		"level": "error",
		"error": 42,
		"msg": "example error log",
		"service": "test-service"
	}
	`
	actualLogJSON := buf.String()
	cleanedActual := comparableLog(actualLogJSON)
	assert.JSONEq(t, expectedLog, cleanedActual)
}

// TestLogLevel ensures that the log level can be dynamically set
func TestLogLevel(t *testing.T) { //nolint:paralleltest // test uses package-level variable to control log level
	// Get the current log level and restore it after the test
	originalLevel := log.GetLogLevel()
	t.Cleanup(func() {
		_ = log.SetLogLevel(originalLevel)
	})

	// Verify the default level is info
	require.Equal(t, strings.ToLower(slog.LevelInfo.String()), log.GetLogLevel())

	// Create a test logger
	logger, buf := newTestLogger(t)

	// Log one each of debug, info, warn, and error level logs
	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")
	logger.Error("error message")

	// Parse the logged output - should contain info, warn, error but not debug
	output := buf.String()
	assert.NotContains(t, output, "debug message")
	assert.Contains(t, output, "info message")
	assert.Contains(t, output, "warn message")
	assert.Contains(t, output, "error message")

	// Set the log level to ERROR
	err := log.SetLogLevel(slog.LevelError.String())
	assert.NoError(t, err)
	require.Equal(t, strings.ToLower(slog.LevelError.String()), log.GetLogLevel())

	// Reset the buffer
	buf.Reset()

	// Log the same messages again
	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")
	logger.Error("error message")

	// Only the error should be logged now
	output = buf.String()
	assert.NotContains(t, output, "debug message")
	assert.NotContains(t, output, "info message")
	assert.NotContains(t, output, "warn message")
	assert.Contains(t, output, "error message")

	// Set the log level to blank - should have no affect
	err = log.SetLogLevel("")
	assert.NoError(t, err)
	require.Equal(t, strings.ToLower(slog.LevelError.String()), log.GetLogLevel())
}

// TestLogErrorWithGroup validates that a simple error is logged correctly when using grouped logging
func TestLogErrorWithGroup(t *testing.T) {
	t.Parallel()

	// Create a test logger with group
	logger, buf := newTestLogger(t)
	logger = logger.WithGroup("error_group")

	// Create a test error that has no extra bells and whistles.
	err := errors.New("grouped error")

	// log the error
	logger.Error("example grouped error log", log.ErrAttr(err))

	expectedLog := `
	{
		"level": "error",
		"error_group": {
			"error": "grouped error",
			"service": "test-service"
		},
		"msg": "example grouped error log",
		"time": "2021-01-01T00:00:00Z"
	}
	`
	actualLogJSON := buf.String()
	cleanedActual := comparableLog(actualLogJSON)
	assert.JSONEq(t, expectedLog, cleanedActual)
}

func removeStackSourceFields(log string) string {
	return sourceRegex.ReplaceAllString(log, "")
}

func removeStackLineNumbers(log string) string {
	return lineRegex.ReplaceAllString(log, `"line":0`)
}

func normalizeTime(log string) string {
	timeRegex := regexp.MustCompile(`"time":"[^"]+`)
	return timeRegex.ReplaceAllString(log, `"time":"2021-01-01T00:00:00Z`)
}

func comparableLog(s string) string {
	s = removeStackSourceFields(s)
	s = normalizeTime(s)
	s = removeStackLineNumbers(s)
	return s
}

// TestLogErrorJoined validates that errors joined via errors.Join are logged correctly.
func TestLogErrorJoined(t *testing.T) {
	t.Parallel()

	// Create a test logger
	logger, buf := newTestLogger(t)

	// Create a complex web of errors
	errA := errors.New("test error A")
	errB := errors.New("test error B")
	errC := errors.New("test error C")
	errD := errors.New("test error D")
	errE := errors.New("test error E")

	errAB := errors.Join(errA, stacktrace.Wrap(errB))
	errCD := stacktrace.Wrap(errors.Join(errC, stacktrace.Wrap(errD)))
	errCDE := errors.Join(errE, errCD)

	errABCDE := errors.Join(errAB, errCDE)

	// log the error
	logger.Error("example joined error log", log.ErrAttr(errABCDE))

	expectedLog := `
	{
		"level": "error",
		"error": "test error A; test error B; test error E; test error C; test error D",
		"errors": ["test error A","test error B","test error E","test error C","test error D"],
		"error_detail": {
			"error_0":{
				"error": "test error A"
			},
			"error_1":{
				"error": "test error B",
				"error_detail": {
					"github_com/zircuit-labs/zkr-go-common/xerrors_ExtendedError[github_com/zircuit-labs/zkr-go-common/xerrors/stacktrace_StackTrace]": [
						{
							"func": "github.com/zircuit-labs/zkr-go-common/log_test.TestLogErrorJoined",
							"line": 0
						}
					]
				}
			},
			"error_2":{
				"error": "test error E"
			},
			"error_3":{
				"error": "test error C",
				"error_detail": {
					"github_com/zircuit-labs/zkr-go-common/xerrors_ExtendedError[github_com/zircuit-labs/zkr-go-common/xerrors/stacktrace_StackTrace]": [
						{
							"func": "github.com/zircuit-labs/zkr-go-common/xerrors/stacktrace.Wrap",
							"line": 0
						},
						{
							"func": "github.com/zircuit-labs/zkr-go-common/log_test.TestLogErrorJoined",
							"line": 0
						}
					]
				}
			},
			"error_4":{
				"error": "test error D",
				"error_detail": {
					"github_com/zircuit-labs/zkr-go-common/xerrors_ExtendedError[github_com/zircuit-labs/zkr-go-common/xerrors/stacktrace_StackTrace]": [
						{
							"func": "github.com/zircuit-labs/zkr-go-common/log_test.TestLogErrorJoined",
							"line": 0
						}
					]
				}
			}
		},
		"msg": "example joined error log",
		"time": "2021-01-01T00:00:00Z",
		"service": "test-service"
	}
	`
	actualLogJSON := buf.String()
	cleanedActual := comparableLog(actualLogJSON)
	assert.JSONEq(t, expectedLog, cleanedActual)
}
