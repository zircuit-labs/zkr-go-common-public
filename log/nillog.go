package log

import (
	"context"
	"log/slog"
)

// NewNilLogger creates a logger that discards all logs.
func NewNilLogger() *slog.Logger {
	return slog.New(&NilHandler{})
}

type NilHandler struct{}

// Enabled returns false for all levels, effectively disabling all logs.
func (h *NilHandler) Enabled(context.Context, slog.Level) bool {
	return false
}

// Handle does nothing.
func (h *NilHandler) Handle(context.Context, slog.Record) error {
	return nil
}

// WithAttrs does nothing.
func (h *NilHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

// WithGroup does nothing.
func (h *NilHandler) WithGroup(name string) slog.Handler {
	return h
}
