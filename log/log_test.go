package log_test

import (
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/rzajac/zltest"
	slogzerolog "github.com/samber/slog-zerolog/v2"
	"github.com/stretchr/testify/assert"
	"github.com/zircuit-labs/zkr-go-common/log"
	"github.com/zircuit-labs/zkr-go-common/xerrors/errclass"
	"github.com/zircuit-labs/zkr-go-common/xerrors/errcontext"
	"github.com/zircuit-labs/zkr-go-common/xerrors/stacktrace"
)

var sourceRegex = regexp.MustCompile(`,?"source":"[^"]+"`)

func init() {
	// set up the logger with a fixed timestamp
	testTime := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	zerolog.TimestampFunc = func() time.Time { return testTime }
}

// TestErrorLog validates that an error is logged correctly when not a joined error.
// WARNING: This test is extremely fragile if line numbers in this file change.
func TestErrorLog(t *testing.T) {
	t.Parallel()

	zlogTester := zltest.New(t)
	zlogger := zlogTester.Logger().With().Timestamp().Logger()

	// use the custom converter, as that is what we are testing
	logger := slog.New(slogzerolog.Option{
		Converter: log.CustomSlogConverter,
		Logger:    &zlogger,
	}.NewZerologHandler())

	// create a test error with all the bells and whistles
	err := fmt.Errorf("test error")
	err = stacktrace.Wrap(err) // this line number must match in the expected log below
	err = errclass.WrapAs(err, errclass.Transient)
	err = errcontext.Add(err, slog.Bool("example_bool", true), slog.Int("example_int", 42))

	// log the error
	logger.Error("example error log", log.ErrAttr(err))

	// check the log output matches what we expect
	expectedLog := `
	{
		"time":"2021-01-01T00:00:00Z",
		"level": "error",
		"error": "test error",
		"error_context": {
			"class": "transient",
			"error": "test error",
			"example_bool": true,
			"example_int": 42,
			"stacktrace": [
				{
					"func": "github.com/zircuit-labs/zkr-go-common/log_test.TestErrorLog",
					"line": "45"
				}
			]
		},
		"message": "example error log"
	}
	`
	actualLog := zlogTester.LastEntry().String()
	assert.JSONEq(t, expectedLog, removeStackSourceFields(actualLog))
}

// TestErrorLog validates that an error is logged correctly when it is a joined error.
// WARNING: This test is extremely fragile if line numbers in this file change.
func TestErrorLogJoined(t *testing.T) {
	t.Parallel()

	zlogTester := zltest.New(t)
	zlogger := zlogTester.Logger().With().Timestamp().Logger()

	// use the custom converter, as that is what we are testing
	logger := slog.New(slogzerolog.Option{
		Converter: log.CustomSlogConverter,
		Logger:    &zlogger,
	}.NewZerologHandler())

	// create a test error with all the bells and whistles
	errA := fmt.Errorf("test error A")
	errA = stacktrace.Wrap(errA) // this line number must match in the expected log below
	errA = errclass.WrapAs(errA, errclass.Transient)
	errA = errcontext.Add(errA, slog.Bool("example_bool", true), slog.Int("example_int", 42))

	// create a second test error with all the bells and whistles
	errB := fmt.Errorf("test error B")
	errB = stacktrace.Wrap(errB)
	errB = errclass.WrapAs(errB, errclass.Persistent)
	errB = errcontext.Add(errB, slog.Duration("example_duration", 5*time.Second))

	// join the errors
	err := errors.Join(errA, errB)

	// log the joined error
	logger.Error("example error log", log.ErrAttr(err))

	// check the log output matches what we expect
	// NOTE: the real stacktrace contains a "source" field that has the full path to the file,
	// which changes based on the environment. We remove this field from the expected log,
	// and the actual log for comparison.
	expectedLog := `
	{
		"level":"error",
		"error":["test error A","test error B"],
		"error_context":{
			"error_0":{
				"class":"transient",
				"error":"test error A",
				"example_bool":true,
				"example_int":42,
				"stacktrace":[
					{
						"func":"github.com/zircuit-labs/zkr-go-common/log_test.TestErrorLogJoined",
						"line":"93"
					}
				]
			},
			"error_1":{
				"class":"persistent",
				"error":"test error B",
				"example_duration":5000000000,
				"stacktrace":[
					{
						"func":"github.com/zircuit-labs/zkr-go-common/log_test.TestErrorLogJoined",
						"line":"99"
					}
				]
			}
		},
		"time":"2021-01-01T00:00:00Z",
		"message":"example error log"
	}
	`

	actualLog := zlogTester.LastEntry().String()
	assert.JSONEq(t, expectedLog, removeStackSourceFields(actualLog))
}

// TestErrorDuplicateContext validates that duplicate context keys are handled correctly.
func TestErrorDuplicateContext(t *testing.T) {
	t.Parallel()

	zlogTester := zltest.New(t)
	zlogger := zlogTester.Logger().With().Timestamp().Logger()

	// use the custom converter, as that is what we are testing
	logger := slog.New(slogzerolog.Option{
		Converter: log.CustomSlogConverter,
		Logger:    &zlogger,
	}.NewZerologHandler())

	// Create a test error and add context that has the same key but different value.
	// While the error contains both values, only the last one is logged.
	err := fmt.Errorf("test error")
	err = errcontext.Add(err, slog.String("key", "value1"))
	err = errcontext.Add(err, slog.String("key", "value2"))

	// log the error
	logger.Error("example error log", log.ErrAttr(err))

	// check the log output matches what we expect
	expectedLog := `
	{
		"time":"2021-01-01T00:00:00Z",
		"level": "error",
		"error": "test error",
		"error_context": {
			"error": "test error",
			"key": "value2"
		},
		"message": "example error log"
	}
	`
	actualLog := zlogTester.LastEntry().String()
	assert.JSONEq(t, expectedLog, actualLog)
}

// TestLogErrorSimple validates that a simple error is logged correctly.
func TestLogErrorSimple(t *testing.T) {
	t.Parallel()

	zlogTester := zltest.New(t)
	zlogger := zlogTester.Logger().With().Timestamp().Logger()

	// use the custom converter, as that is what we are testing
	logger := slog.New(slogzerolog.Option{
		Converter: log.CustomSlogConverter,
		Logger:    &zlogger,
	}.NewZerologHandler())

	// Create a test error that has no extra bells and whistles.
	err := fmt.Errorf("test error")

	// log the error
	logger.Error("example error log", log.ErrAttr(err))

	// check the log output matches what we expect
	expectedLog := `
	{
		"time":"2021-01-01T00:00:00Z",
		"level": "error",
		"error": "test error",
		"message": "example error log"
	}
	`
	actualLog := zlogTester.LastEntry().String()
	assert.JSONEq(t, expectedLog, actualLog)
}

// TestLogCompoundError validates that a joined error consisting of
// a simple error and a complex error is logged correctly.
func TestLogCompoundError(t *testing.T) {
	t.Parallel()

	zlogTester := zltest.New(t)
	zlogger := zlogTester.Logger().With().Timestamp().Logger()

	// use the custom converter, as that is what we are testing
	logger := slog.New(slogzerolog.Option{
		Converter: log.CustomSlogConverter,
		Logger:    &zlogger,
	}.NewZerologHandler())

	// Create one test error that has no extra bells and whistles,
	// and another that does.
	errA := fmt.Errorf("test error A")
	errB := fmt.Errorf("test error B")
	errB = errclass.WrapAs(errB, errclass.Persistent)
	// Join the errors together.
	err := errors.Join(errA, errB)

	// log the error
	logger.Error("example error log", log.ErrAttr(err))

	// check the log output matches what we expect
	expectedLog := `
	{
		"time": "2021-01-01T00:00:00Z",
		"level": "error",
		"message": "example error log",
		"error": [
			"test error A",
			"test error B"
		],
		"error_context": {
			"error_0": {
				"error": "test error A"
			},
			"error_1": {
				"class": "persistent",
				"error": "test error B"
			}
		}
	}
	`
	actualLog := zlogTester.LastEntry().String()
	assert.JSONEq(t, expectedLog, actualLog)
}

// TestErrorLogNil validates that a nil error is logged correctly.
func TestErrorLogNil(t *testing.T) {
	t.Parallel()

	zlogTester := zltest.New(t)
	zlogger := zlogTester.Logger().With().Timestamp().Logger()

	// use the custom converter, as that is what we are testing
	logger := slog.New(slogzerolog.Option{
		Converter: log.CustomSlogConverter,
		Logger:    &zlogger,
	}.NewZerologHandler())

	// log a nil error
	logger.Error("example error log", log.ErrAttr(nil))

	// check the log output matches what we expect
	expectedLog := `
	{
		"time":"2021-01-01T00:00:00Z",
		"level": "error",
		"error": null,
		"message": "example error log"
	}
	`
	actualLog := zlogTester.LastEntry().String()
	assert.JSONEq(t, expectedLog, actualLog)
}

// TestErrorLogNotAnError validates that a non-error with error key is logged correctly.
func TestErrorLogNotAnError(t *testing.T) {
	t.Parallel()

	zlogTester := zltest.New(t)
	zlogger := zlogTester.Logger().With().Timestamp().Logger()

	// use the custom converter, as that is what we are testing
	logger := slog.New(slogzerolog.Option{
		Converter: log.CustomSlogConverter,
		Logger:    &zlogger,
	}.NewZerologHandler())

	// log a nil error
	logger.Error("example error log", slog.Int("error", 42))

	// check the log output matches what we expect
	expectedLog := `
	{
		"time":"2021-01-01T00:00:00Z",
		"level": "error",
		"error": 42,
		"message": "example error log"
	}
	`
	actualLog := zlogTester.LastEntry().String()
	assert.JSONEq(t, expectedLog, actualLog)
}

func removeStackSourceFields(log string) string {
	return sourceRegex.ReplaceAllString(log, "")
}
