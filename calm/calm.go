// Package calm allows users to call a function and capture any panic as an error with stack trace instead.
package calm

import (
	"fmt"

	"github.com/zircuit-labs/zkr-go-common/xerrors"
	"github.com/zircuit-labs/zkr-go-common/xerrors/errclass"
	"github.com/zircuit-labs/zkr-go-common/xerrors/stacktrace"
)

const (
	// depth of stack to ignore so that the stack trace from recovered panic
	// does not include the deferred recovery function itself.
	panicStackDepth = 3
)

// Unpanic executes the given function catching any panic and returning it as an error with stack trace.
// WARNING: It is not possible to recover from a panic in a goroutine spawned by `f()`. Users should ensure
// that any goroutines created by `f()` are likewise guarded against panics.
func Unpanic(f func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			r := fmt.Errorf("panic: %v", r)
			r = xerrors.Extend(stacktrace.GetStack(panicStackDepth, true), r)
			err = errclass.WrapAs(r, errclass.Panic)
		}
	}()

	return f()
}
