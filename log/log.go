package log

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/zircuit-labs/zkr-go-common/version"
)

const (
	ErrorKey = "error"
)

type LogStyle int

const (
	LogStyleJSON = iota
	LogStyleText
)

var logLevel = &slog.LevelVar{}

func SetLogLevel(level string) error {
	if level != "" {
		return logLevel.UnmarshalText([]byte(level))
	}
	return nil
}

func GetLogLevel() string {
	return strings.ToLower(logLevel.Level().String())
}

// ErrAttr is a helper for logging error values using LoggableError wrapper.
// It wraps the error with LoggableError to enable custom logging behavior via LogValuer interface.
func ErrAttr(err error) slog.Attr {
	if err == nil {
		return slog.Any(ErrorKey, nil)
	}
	return slog.Any(ErrorKey, Loggable(err))
}

// NewNilLogger creates a logger that discards all logs.
func NewNilLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

// NewTestLogger creates a new logger for testing.
// NOTE: Since this logger uses the testing t.Log method,
// it will only log when the test fails. Additionally,
// it will cause a panic if the logger is called after the
// test has completed. This can be helpful for finding race conditions.
func NewTestLogger(t *testing.T) *slog.Logger {
	t.Helper()
	log, err := NewLogger(WithWriter(t.Output()), WithLogStyle(LogStyleJSON))
	if err != nil {
		t.Fatalf("NewLogger failed: %v", err)
	}
	return log
}

type options struct {
	writer      io.Writer
	instanceID  string
	serviceName string
	versionInfo *version.VersionInformation
	logStyle    LogStyle
}

// Option configures logger creation
type Option func(*options)

// WithWriter configures the logger to write to the specified io.Writer
func WithWriter(w io.Writer) Option {
	return func(opts *options) {
		if w == nil {
			opts.writer = io.Discard
			return
		}
		opts.writer = w
	}
}

// WithInstanceID configures the logger to emit the instance field with every log
func WithInstanceID(id string) Option {
	return func(opts *options) {
		opts.instanceID = id
	}
}

// WithServiceName configures the logger to emit the service field with every log
func WithServiceName(name string) Option {
	return func(opts *options) {
		opts.serviceName = name
	}
}

// WithVersion configures the logger to emit the version information with every log
func WithVersion(versionInfo *version.VersionInformation) Option {
	return func(opts *options) {
		opts.versionInfo = versionInfo
	}
}

// WithLogStyle configures the logger to use the given supported style of logging
// Ideally this would allow for any slog.Handler however that is not possible at this time
func WithLogStyle(logStyle LogStyle) Option {
	return func(opts *options) {
		opts.logStyle = logStyle
	}
}

// NewLogger creates a new logger using replaceattrmore.Handler chained with slog.JSONHandler.
// This approach leverages all of slog's built-in functionality while providing custom
// LoggableError flattening. Use ErrAttr() when logging errors with this logger.
func NewLogger(opts ...Option) (*slog.Logger, error) {
	// Parse cfg
	cfg := options{
		writer:   os.Stdout,
		logStyle: LogStyleJSON,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	// Create base log handler with lowercase level formatting and key sanitization as required
	logHandler, err := formatHandler(cfg.logStyle, cfg.writer)
	if err != nil {
		return nil, err
	}

	// Chain with loggable error handler for error flattening
	handler := NewLoggableErrorHandler(logHandler)

	// Add Optional Attributes
	attrs := []slog.Attr{}
	if cfg.serviceName != "" {
		attrs = append(attrs, slog.String("service", cfg.serviceName))
	}
	if cfg.instanceID != "" {
		attrs = append(attrs, slog.String("instance", cfg.instanceID))
	}
	if cfg.versionInfo != nil {
		if c := cfg.versionInfo.Commit(); c != "" {
			attrs = append(attrs, slog.String("git_commit", c))
		}
		if !cfg.versionInfo.Date.IsZero() {
			attrs = append(attrs, slog.Time("git_commit_time", cfg.versionInfo.Date))
		}
		if v := cfg.versionInfo.Version; v != "" {
			attrs = append(attrs, slog.String("version", v))
		}
	}

	return slog.New(handler.WithAttrs(attrs)), nil
}

func formatHandler(logStyle LogStyle, writer io.Writer) (slog.Handler, error) {
	handlerOptions := &slog.HandlerOptions{
		Level: logLevel,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Convert level to lowercase to match our expected format
			if a.Key == slog.LevelKey {
				if lvl, ok := a.Value.Any().(slog.Level); ok {
					a.Value = slog.StringValue(strings.ToLower(lvl.String()))
				} else {
					// Fallback if another handler set a string or other kind
					a.Value = slog.StringValue(strings.ToLower(a.Value.String()))
				}
			}
			return a
		},
	}

	switch logStyle {
	case LogStyleJSON:
		return slog.NewJSONHandler(writer, handlerOptions), nil
	case LogStyleText:
		return slog.NewTextHandler(writer, handlerOptions), nil
	default:
		return nil, fmt.Errorf("unsupported log style option: %v", logStyle)
	}
}
