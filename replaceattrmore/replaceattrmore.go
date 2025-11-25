package replaceattrmore

import (
	"context"
	"log/slog"
)

// ReplaceAttrMoreFunc is like slog's ReplaceAttr but allows returning multiple attributes
// from a single input attribute, enabling 1-to-many transformations.
type ReplaceAttrMoreFunc func(groups []string, a slog.Attr) []slog.Attr

// Handler wraps any slog.Handler and applies ReplaceAttrMoreFunc
// transformations before passing the record to the wrapped handler.
type Handler struct {
	next    slog.Handler
	replace ReplaceAttrMoreFunc
	attrs   []slog.Attr
	groups  []string
}

// Compile-time interface assertion
var _ slog.Handler = (*Handler)(nil)

// New creates a handler wrapper that can expand single attributes
// into multiple attributes before passing to the next handler.
func New(next slog.Handler, replace ReplaceAttrMoreFunc) *Handler {
	return &Handler{
		next:    next,
		replace: replace,
	}
}

func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	// Collect all attributes from the record
	var allAttrs []slog.Attr

	// Add pre-configured attributes from WithAttrs
	allAttrs = append(allAttrs, h.attrs...)

	// Add record attributes
	r.Attrs(func(a slog.Attr) bool {
		allAttrs = append(allAttrs, a)
		return true
	})

	// Transform attributes using ReplaceAttrMoreFunc
	var transformedAttrs []slog.Attr
	for _, attr := range allAttrs {
		if h.replace != nil {
			resolved := slog.Attr{Key: attr.Key, Value: attr.Value.Resolve()}
			expanded := h.replace(h.groups, resolved)
			transformedAttrs = append(transformedAttrs, expanded...)
		} else {
			transformedAttrs = append(transformedAttrs, attr)
		}
	}

	// Create new record with transformed attributes
	newRecord := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)
	newRecord.AddAttrs(transformedAttrs...)

	return h.next.Handle(ctx, newRecord)
}

// WithAttrs returns a new Handler with the given attributes added.
// Note: Transforming in WithAttrs uses the group set at call time;
// later WithGroup calls won't influence these attrs.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	// Transform the new attrs through ReplaceAttrMoreFunc
	var transformedAttrs []slog.Attr
	for _, attr := range attrs {
		if h.replace != nil {
			resolved := slog.Attr{Key: attr.Key, Value: attr.Value.Resolve()}
			expanded := h.replace(h.groups, resolved)
			transformedAttrs = append(transformedAttrs, expanded...)
		} else {
			transformedAttrs = append(transformedAttrs, attr)
		}
	}

	return &Handler{
		next:    h.next,
		replace: h.replace,
		attrs:   append(h.attrs, transformedAttrs...),
		groups:  h.groups,
	}
}

func (h *Handler) WithGroup(name string) slog.Handler {
	return &Handler{
		next:    h.next.WithGroup(name),
		replace: h.replace,
		attrs:   h.attrs,
		groups:  append(h.groups, name),
	}
}
