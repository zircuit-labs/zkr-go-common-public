package replaceattrmore_test

import (
	"bytes"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/zircuit-labs/zkr-go-common/replaceattrmore"
)

var timeRegex = regexp.MustCompile(`"time":"[^"]+`)

// normalizeTime replaces time values with a fixed time for consistent testing
func normalizeTime(log string) string {
	return timeRegex.ReplaceAllString(log, `"time":"2021-01-01T00:00:00Z`)
}

func TestHandler_BasicFunctionality(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	// Create base JSONHandler
	jsonHandler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	// Create a simple replace function that converts "test" attributes to uppercase
	replaceFunc := func(groups []string, a slog.Attr) []slog.Attr {
		if a.Key == "test" && a.Value.Kind() == slog.KindString {
			return []slog.Attr{
				slog.String("test", strings.ToUpper(a.Value.String())),
			}
		}
		return []slog.Attr{a}
	}

	// Chain handlers
	handler := replaceattrmore.New(jsonHandler, replaceFunc)
	logger := slog.New(handler)

	// Log with test attribute
	logger.Info("test message", slog.String("test", "hello world"))

	expectedJSON := `
	{
		"time":"2021-01-01T00:00:00Z",
		"level":"INFO",
		"msg":"test message",
		"test":"HELLO WORLD"
	}`

	actualLogJSON := normalizeTime(buf.String())
	assert.JSONEq(t, expectedJSON, actualLogJSON)
}

func TestHandler_OneToManyTransformation(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	jsonHandler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	// Replace function that expands integer attributes into value, hex, and parity
	replaceFunc := func(groups []string, a slog.Attr) []slog.Attr {
		if a.Value.Kind() == slog.KindInt64 {
			val := a.Value.Int64()
			parity := "even"
			if val%2 != 0 {
				parity = "odd"
			}
			return []slog.Attr{
				a, // Keep original int attribute
				slog.String(a.Key+"_hex", fmt.Sprintf("0x%x", val)),
				slog.String(a.Key+"_parity", parity),
			}
		}
		return []slog.Attr{a}
	}

	handler := replaceattrmore.New(jsonHandler, replaceFunc)
	logger := slog.New(handler)
	logger.Info("number analysis",
		slog.Int("value", 42),
		slog.Int("count", 7))

	expectedJSON := `
	{
		"time":"2021-01-01T00:00:00Z",
		"level":"INFO",
		"msg":"number analysis",
		"value":42,
		"value_hex":"0x2a",
		"value_parity":"even",
		"count":7,
		"count_hex":"0x7",
		"count_parity":"odd"
	}`
	actualLogJSON := normalizeTime(buf.String())
	assert.JSONEq(t, expectedJSON, actualLogJSON)
}

func TestHandler_WithAttrs(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	jsonHandler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	replaceFunc := func(groups []string, a slog.Attr) []slog.Attr {
		if a.Key == "prefix" {
			return []slog.Attr{
				slog.String("prefix_value", a.Value.String()),
				slog.String("prefix_type", "custom"),
			}
		}
		return []slog.Attr{a}
	}

	handler := replaceattrmore.New(jsonHandler, replaceFunc)
	logger := slog.New(handler)

	// Add attributes via WithAttrs
	loggerWithAttrs := logger.With(slog.String("prefix", "test"))
	loggerWithAttrs.Info("test message")

	expectedJSON := `{
		"time":"2021-01-01T00:00:00Z",
		"level":"INFO",
		"msg":"test message",
		"prefix_value":"test",
		"prefix_type":"custom"
	}`
	actualLogJSON := normalizeTime(buf.String())
	assert.JSONEq(t, expectedJSON, actualLogJSON)
}

func TestHandler_WithGroup(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	jsonHandler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	replaceFunc := func(groups []string, a slog.Attr) []slog.Attr {
		// Add group information only for the id attribute to avoid duplicates
		if len(groups) > 0 && a.Key == "id" {
			groupPath := strings.Join(groups, ".")
			return []slog.Attr{
				a,
				slog.String("attr_group", groupPath),
			}
		}
		return []slog.Attr{a}
	}

	handler := replaceattrmore.New(jsonHandler, replaceFunc)
	logger := slog.New(handler)

	// Use WithGroup
	groupLogger := logger.WithGroup("request")
	groupLogger.Info("grouped message", slog.String("id", "123"))

	expectedJSON := `
	{
		"time":"2021-01-01T00:00:00Z",
		"level":"INFO",
		"msg":"grouped message",
		"request": {
			"id":"123",
			"attr_group":"request"
		}
	}`
	actualLogJSON := normalizeTime(buf.String())
	assert.JSONEq(t, expectedJSON, actualLogJSON)
}

func TestHandler_NilReplaceFunc(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	jsonHandler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	// Create handler with nil replace function
	handler := replaceattrmore.New(jsonHandler, nil)
	logger := slog.New(handler)

	// Test WithAttrs with nil replace function
	loggerWithAttrs := logger.With(slog.String("preset", "value"))
	loggerWithAttrs.Info("test message", slog.String("key", "value"))

	// Should pass through unchanged
	expectedJSON := `
	{
		"time":"2021-01-01T00:00:00Z",
		"level":"INFO",
		"msg":"test message",
		"preset":"value",
		"key":"value"
	}`
	actualLogJSON := normalizeTime(buf.String())
	assert.JSONEq(t, expectedJSON, actualLogJSON)
}

func TestHandler_Enabled(t *testing.T) {
	t.Parallel()

	jsonHandler := slog.NewJSONHandler(&bytes.Buffer{}, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	})

	handler := replaceattrmore.New(jsonHandler, nil)

	ctx := t.Context()

	assert.False(t, handler.Enabled(ctx, slog.LevelDebug))
	assert.False(t, handler.Enabled(ctx, slog.LevelInfo))
	assert.True(t, handler.Enabled(ctx, slog.LevelWarn))
	assert.True(t, handler.Enabled(ctx, slog.LevelError))
}

func TestHandler_ComplexTransformation(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	jsonHandler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	// Complex replace function that handles multiple attribute types
	replaceFunc := func(groups []string, a slog.Attr) []slog.Attr {
		switch a.Key {
		case "error":
			// Expand error into error and error_type
			return []slog.Attr{
				slog.String("error", a.Value.String()),
				slog.String("error_type", "application"),
				slog.Bool("has_error", true),
			}
		case "duration":
			// Convert duration to both string and milliseconds
			if a.Value.Kind() == slog.KindDuration {
				dur := a.Value.Duration()
				return []slog.Attr{
					slog.String("duration", dur.String()),
					slog.Float64("duration_ms", float64(dur.Nanoseconds())/1e6),
				}
			}
		case "count":
			// Add categorization based on count value
			if a.Value.Kind() == slog.KindInt64 {
				count := a.Value.Int64()
				category := "low"
				if count > 100 {
					category = "high"
				} else if count > 10 {
					category = "medium"
				}
				return []slog.Attr{
					a,
					slog.String("count_category", category),
				}
			}
		}
		return []slog.Attr{a}
	}

	handler := replaceattrmore.New(jsonHandler, replaceFunc)
	logger := slog.New(handler)

	logger.Error("complex log",
		slog.String("error", "connection failed"),
		slog.Duration("duration", time.Millisecond*150),
		slog.Int("count", 42))

	expectedJSON := `
	{
		"time":"2021-01-01T00:00:00Z",
		"level":"ERROR",
		"msg":"complex log",
		"error":"connection failed",
		"error_type":"application",
		"has_error":true,
		"duration":"150ms",
		"duration_ms":150,
		"count":42,
		"count_category":"medium"
	}`
	actualLogJSON := normalizeTime(buf.String())
	assert.JSONEq(t, expectedJSON, actualLogJSON)
}
