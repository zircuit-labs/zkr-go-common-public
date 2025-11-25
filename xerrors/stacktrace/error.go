package stacktrace

import (
	"errors"
	"sync/atomic"

	"github.com/zircuit-labs/zkr-go-common/xerrors"
)

const (
	// depth of stack to ignore so that callers of Wrap don't see the call to Wrap itself.
	wrapStackDepth = 4 // Updated to account for the additional wrapSingleError call
)

// Disabled disables stacktrace collection in Wrap when set to true.
var Disabled atomic.Bool

// Wrap extends an error by including a stack trace at the point where this was called.
// If the error already contains a stack trace, it is not wrapped again.
// For joined errors, the wrap is applied to each individual error.
func Wrap(err error) error {
	// no-op if disabled or the error is nil
	if Disabled.Load() || err == nil {
		return err
	}

	// Check if this is a joined error
	if joinedErrors := xerrors.Unjoin(err); len(joinedErrors) > 1 {
		// Apply wrap to each direct child error (recursion happens naturally)
		wrappedErrors := make([]error, len(joinedErrors))
		for i, e := range joinedErrors {
			wrappedErrors[i] = Wrap(e) // Recursive call to preserve structure
		}
		return errors.Join(wrappedErrors...)
	}

	// Handle single error
	return wrapSingleError(err)
}

// wrapSingleError wraps a single error with a stack trace if it doesn't already have one
func wrapSingleError(err error) error {
	if _, ok := xerrors.Extract[StackTrace](err); !ok {
		return xerrors.Extend(GetStack(wrapStackDepth, true), err)
	}
	return err
}

// Extract returns the StackTrace embedded in the error if it exists.
func Extract(err error) StackTrace {
	st, ok := xerrors.Extract[StackTrace](err)
	if !ok {
		return nil
	}
	return st
}
