package errcontext

import (
	"log/slog"

	"github.com/zircuit-labs/zkr-go-common/xerrors"
)

type Context []slog.Attr

// Add wraps the given error with log attributes for greater context.
// If the error already has context, the new context is appended to the existing context
// and the error wrapped again with the new context.
func Add(err error, context ...slog.Attr) error {
	if err == nil {
		return nil
	}

	var newContext Context

	if oldContext := Get(err); oldContext != nil {
		newContext = append(oldContext, context...)
	} else {
		newContext = append(make(Context, 0, len(context)), context...)
	}
	return xerrors.Extend(newContext, err)
}

// Get returns the context of the given error.
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
