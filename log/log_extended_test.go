package log_test

import (
	"bytes"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zircuit-labs/zkr-go-common/log"
	"github.com/zircuit-labs/zkr-go-common/version"
)

var timeRegex = regexp.MustCompile(`time=\S+`)

// Helper functions for text output comparison
func normalizeTextTime(log string) string {
	// Replace time=<timestamp> with time=2021-01-01T00:00:00Z
	return timeRegex.ReplaceAllString(log, "time=2021-01-01T00:00:00Z")
}

func comparableTextLog(s string) string {
	return normalizeTextTime(s)
}

func TestNewLogger_WithServiceName(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger, err := log.NewLogger(log.WithWriter(&buf), log.WithServiceName("my-service"))
	require.NoError(t, err)

	logger.Info("test message")

	expectedLog := `{
		"time": "2021-01-01T00:00:00Z",
		"level": "info",
		"msg": "test message",
		"service": "my-service"
	}`

	actualLogJSON := buf.String()
	cleanedActual := comparableLog(actualLogJSON)
	assert.JSONEq(t, expectedLog, cleanedActual)
}

func TestNewLogger_WithInstanceID(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger, err := log.NewLogger(log.WithWriter(&buf), log.WithInstanceID("instance-456"))
	require.NoError(t, err)

	logger.Info("test message")

	expectedLog := `{
		"time": "2021-01-01T00:00:00Z",
		"level": "info",
		"msg": "test message",
		"instance": "instance-456"
	}`

	actualLogJSON := buf.String()
	cleanedActual := comparableLog(actualLogJSON)
	assert.JSONEq(t, expectedLog, cleanedActual)
}

func TestNewLogger_WithVersion(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	versionInfo := &version.VersionInformation{
		Version:   "v2.1.0",
		Date:      time.Date(2023, 6, 15, 10, 30, 0, 0, time.UTC),
		GitCommit: "def456",
	}

	logger, err := log.NewLogger(log.WithWriter(&buf), log.WithVersion(versionInfo))
	require.NoError(t, err)

	logger.Info("version test")

	expectedLog := `{
		"time": "2021-01-01T00:00:00Z",
		"level": "info",
		"msg": "version test",
		"git_commit": "def456",
		"git_commit_time": "2023-06-15T10:30:00Z",
		"version": "v2.1.0"
	}`

	actualLogJSON := buf.String()
	cleanedActual := comparableLog(actualLogJSON)
	assert.JSONEq(t, expectedLog, cleanedActual)
}

func TestNewLogger_WithVersion_PartialInfo(t *testing.T) {
	t.Parallel()

	t.Run("OnlyVersion", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		versionInfo := &version.VersionInformation{Version: "v1.0.0"}

		logger, err := log.NewLogger(log.WithWriter(&buf), log.WithVersion(versionInfo))
		require.NoError(t, err)

		logger.Info("partial version")

		expectedLog := `{
			"time": "2021-01-01T00:00:00Z",
			"level": "info",
			"msg": "partial version",
			"git_commit": "unknown",
			"version": "v1.0.0"
		}`

		actualLogJSON := buf.String()
		cleanedActual := comparableLog(actualLogJSON)
		assert.JSONEq(t, expectedLog, cleanedActual)
	})

	t.Run("OnlyCommit", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		versionInfo := &version.VersionInformation{
			GitCommit: "abc123",
		}

		logger, err := log.NewLogger(log.WithWriter(&buf), log.WithVersion(versionInfo))
		require.NoError(t, err)

		logger.Info("commit only")

		expectedLog := `{
			"time": "2021-01-01T00:00:00Z",
			"level": "info",
			"msg": "commit only",
			"git_commit": "abc123"
		}`

		actualLogJSON := buf.String()
		cleanedActual := comparableLog(actualLogJSON)
		assert.JSONEq(t, expectedLog, cleanedActual)
	})
}

func TestNewLogger_WithLogStyle_Text(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger, err := log.NewLogger(
		log.WithWriter(&buf),
		log.WithLogStyle(log.LogStyleText),
		log.WithServiceName("text-service"),
	)
	require.NoError(t, err)

	logger.Info("text output test")

	expectedOutput := "time=2021-01-01T00:00:00Z level=info msg=\"text output test\" service=text-service\n"
	actualOutput := buf.String()
	cleanedActual := comparableTextLog(actualOutput)
	assert.Equal(t, expectedOutput, cleanedActual)
}

func TestNewLogger_WithLogStyle_Text_WithError(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger, err := log.NewLogger(
		log.WithWriter(&buf),
		log.WithLogStyle(log.LogStyleText),
	)
	require.NoError(t, err)

	testErr := errors.New("text error")
	logger.Error("text error test", log.ErrAttr(testErr))

	actualOutput := buf.String()
	cleanedActual := comparableTextLog(actualOutput)

	// Text format should contain key information
	assert.Contains(t, cleanedActual, "level=error")
	assert.Contains(t, cleanedActual, "msg=\"text error test\"")
	assert.Contains(t, cleanedActual, "error=\"text error\"")
	assert.Contains(t, cleanedActual, "time=2021-01-01T00:00:00Z")
}

func TestNewLogger_WithWriter_Nil(t *testing.T) {
	t.Parallel()

	logger, err := log.NewLogger(log.WithWriter(nil))
	require.NoError(t, err)

	// Should not panic with nil writer (uses io.Discard)
	require.NotPanics(t, func() {
		logger.Info("discarded message")
		logger.Error("discarded error")
	})
}

func TestNewLogger_AllOptions(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	versionInfo := &version.VersionInformation{
		Version:   "v3.0.0",
		Date:      time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		GitCommit: "full123",
	}

	logger, err := log.NewLogger(
		log.WithWriter(&buf),
		log.WithServiceName("full-service"),
		log.WithInstanceID("full-instance"),
		log.WithVersion(versionInfo),
		log.WithLogStyle(log.LogStyleJSON),
	)
	require.NoError(t, err)

	logger.Warn("all options test")

	expectedLog := `{
		"time": "2021-01-01T00:00:00Z",
		"level": "warn",
		"msg": "all options test",
		"service": "full-service",
		"instance": "full-instance",
		"git_commit": "full123",
		"git_commit_time": "2024-01-01T12:00:00Z",
		"version": "v3.0.0"
	}`

	actualLogJSON := buf.String()
	cleanedActual := comparableLog(actualLogJSON)
	assert.JSONEq(t, expectedLog, cleanedActual)
}

func TestNewLogger_InvalidLogStyle(t *testing.T) {
	t.Parallel()

	logger, err := log.NewLogger(log.WithLogStyle(log.LogStyle(999)))
	assert.Error(t, err)
	assert.Nil(t, logger)
	assert.Contains(t, err.Error(), "unsupported log style option: 999")
}

func TestNewNilLogger(t *testing.T) {
	t.Parallel()

	logger := log.NewNilLogger()
	require.NotNil(t, logger)

	// Should not panic and should not output anything
	require.NotPanics(t, func() {
		logger.Debug("nil debug")
		logger.Info("nil info")
		logger.Warn("nil warn")
		logger.Error("nil error")
	})
}

func TestNewTestLogger(t *testing.T) {
	t.Parallel()

	logger := log.NewTestLogger(t)
	require.NotNil(t, logger)

	// Should not panic
	require.NotPanics(t, func() {
		logger.Info("test logger info")
		logger.Error("test logger error")
	})
}

func TestErrAttr_NilError(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger, err := log.NewLogger(log.WithWriter(&buf))
	require.NoError(t, err)

	logger.Error("nil error test", log.ErrAttr(nil))

	expectedLog := `{
		"time": "2021-01-01T00:00:00Z",
		"level": "error",
		"error": null,
		"msg": "nil error test"
	}`

	actualLogJSON := buf.String()
	cleanedActual := comparableLog(actualLogJSON)
	assert.JSONEq(t, expectedLog, cleanedActual)
}

func TestLoggable_Function(t *testing.T) {
	t.Parallel()

	t.Run("WrapError", func(t *testing.T) {
		t.Parallel()
		originalErr := errors.New("wrapped error")
		loggableErr := log.Loggable(originalErr)

		assert.Equal(t, "wrapped error", loggableErr.Error())
		assert.Equal(t, originalErr, loggableErr.Unwrap())
	})

	t.Run("WrapNilError", func(t *testing.T) {
		t.Parallel()
		loggableErr := log.Loggable(nil)

		assert.Equal(t, "", loggableErr.Error())
		assert.Nil(t, loggableErr.Unwrap())
	})
}

func TestSetLogLevel_EdgeCases(t *testing.T) { //nolint:paralleltest // This test cannot be parallel since changes global state
	// Save and restore original level
	originalLevel := log.GetLogLevel()
	t.Cleanup(func() {
		_ = log.SetLogLevel(originalLevel)
	})

	t.Run("ValidLevels", func(t *testing.T) { //nolint:paralleltest // This test cannot be parallel since changes global state
		levels := []string{"debug", "info", "warn", "error"}
		for _, level := range levels {
			err := log.SetLogLevel(level)
			assert.NoError(t, err)
			assert.Equal(t, level, log.GetLogLevel())
		}
	})

	t.Run("EmptyString", func(t *testing.T) { //nolint:paralleltest // This test cannot be parallel since changes global state
		err := log.SetLogLevel("")
		assert.NoError(t, err)
	})

	t.Run("InvalidLevel", func(t *testing.T) { //nolint:paralleltest // This test cannot be parallel since changes global state
		err := log.SetLogLevel("invalid")
		assert.Error(t, err)
	})

	t.Run("CaseInsensitive", func(t *testing.T) { //nolint:paralleltest // This test cannot be parallel since changes global state
		err := log.SetLogLevel("INFO")
		assert.NoError(t, err)
		assert.Equal(t, "info", log.GetLogLevel())
	})
}

func TestLogLevel_OutputFiltering(t *testing.T) { //nolint:paralleltest // This test cannot be parallel since changes global state
	// Save and restore original level
	originalLevel := log.GetLogLevel()
	t.Cleanup(func() {
		_ = log.SetLogLevel(originalLevel)
	})

	t.Run("WarnLevel_FiltersDebugAndInfo", func(t *testing.T) {
		var buf bytes.Buffer
		logger, err := log.NewLogger(log.WithWriter(&buf))
		require.NoError(t, err)

		_ = log.SetLogLevel("warn")

		logger.Debug("debug message")
		logger.Info("info message")
		logger.Warn("warn message")
		logger.Error("error message")

		output := buf.String()
		// Should only contain warn and error
		assert.NotContains(t, output, "debug message")
		assert.NotContains(t, output, "info message")
		assert.Contains(t, output, "warn message")
		assert.Contains(t, output, "error message")
	})
}
