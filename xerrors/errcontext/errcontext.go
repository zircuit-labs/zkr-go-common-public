package errcontext

import (
	"errors"
	"log/slog"
	"maps"
	"sort"

	"github.com/zircuit-labs/zkr-go-common/xerrors"
)

type Context map[string]slog.Value

// Flatten converts c to slice of slog.Attr
func (c Context) Flatten() []slog.Attr {
	attrs := make([]slog.Attr, 0, len(c))
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		attrs = append(attrs, slog.Attr{Key: key, Value: c[key]})
	}
	return attrs
}

// LogValue implements slog.LogValuer for Context.
// It returns the context as a flat object with direct key-value pairs.
func (c Context) LogValue() slog.Value {
	if len(c) == 0 {
		return slog.Value{}
	}

	attrs := c.Flatten()
	return slog.GroupValue(attrs...)
}

// Add wraps the given error with log attributes for greater context.
// If the error already has context, the new context replaces any existing keys (last-entry-wins)
// and the error wrapped again with the new context.
// For joined errors, the context is applied to each individual error.
func Add(err error, context ...slog.Attr) error {
	if err == nil {
		return nil
	}

	// Check if this is a joined error
	if joinedErrors := xerrors.Unjoin(err); len(joinedErrors) > 1 {
		// Apply context to each direct child error (recursion happens naturally)
		contextualizedErrors := make([]error, len(joinedErrors))
		for i, e := range joinedErrors {
			contextualizedErrors[i] = Add(e, context...) // Recursive call to preserve structure
		}
		return errors.Join(contextualizedErrors...)
	}

	// Handle single error
	return addContextToSingleError(err, context...)
}

// addContextToSingleError adds context to a single error with last-entry-wins behavior
func addContextToSingleError(err error, context ...slog.Attr) error {
	var newContext Context

	if oldAttrs := Get(err); oldAttrs != nil {
		newContext = maps.Clone(oldAttrs)
	} else {
		newContext = make(Context, len(context))
	}

	// Add new context, replacing any duplicate keys (last-entry-wins)
	for _, attr := range context {
		newContext[attr.Key] = attr.Value
	}

	return xerrors.Extend(newContext, err)
}

// Get returns the newest Context map attached to the given error.
// Only the newest context is returned.
func Get(err error) Context {
	if err == nil {
		return nil
	}

	if context, ok := xerrors.Extract[Context](err); ok {
		return context
	}
	return nil
}
