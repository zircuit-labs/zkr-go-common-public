// Package xerrors provides a generic implementation of error wrapping, allowing any data type to be captured alongside an error.
package xerrors

import (
	"errors"
	"log/slog"
)

// ExtendedError is a custom error type that contains additional data using generics.
type ExtendedError[T any] struct {
	Data T
	err  error
}

// Error meets the error interface by returning the Error() of the underlying.
func (e ExtendedError[T]) Error() string {
	return e.err.Error()
}

// Unwrap returns the underlying wrapped error.
func (e ExtendedError[T]) Unwrap() error {
	return e.err
}

// LogValue implements slog.LogValuer for ExtendedError.
// It returns the extended data. The error message is handled at a higher level to avoid redundancy.
func (e ExtendedError[T]) LogValue() slog.Value {
	// Check if the data itself implements LogValuer
	if logValuer, ok := any(e.Data).(slog.LogValuer); ok {
		return logValuer.LogValue()
	}
	return slog.AnyValue(e.Data)
}

// Extend creates an ExtendedError wrapping an original error with additional data.
func Extend[T any](data T, err error) error {
	if err == nil {
		return nil
	}
	return ExtendedError[T]{Data: data, err: err}
}

// Extract returns the desired data if possible, even in cases of deeply nested wrapping.
// NOTE: If an error is extended multiple times with the same data type, only the first match is returned.
func Extract[T any](err error) (T, bool) {
	var extendedError ExtendedError[T]
	ok := errors.As(err, &extendedError)
	return extendedError.Data, ok
}

// Unjoin returns the underlying errors if the error was joined with errors.Join.
func Unjoin(err error) []error {
	if err == nil {
		return nil
	}
	if joinedErrs, ok := err.(interface{ Unwrap() []error }); ok {
		return joinedErrs.Unwrap()
	}
	return []error{err}
}

// Flatten recursively extracts all individual errors from a joined error tree.
// Unlike Unjoin which returns only direct children, Flatten returns all leaf errors.
func Flatten(err error) []error {
	if err == nil {
		return nil
	}

	// Check if this implements Unwrap() []error (joined error)
	if joinedErrs, ok := err.(interface{ Unwrap() []error }); ok {
		var allErrors []error
		for _, e := range joinedErrs.Unwrap() {
			// Recursively extract from nested joins
			allErrors = append(allErrors, Flatten(e)...)
		}
		return allErrors
	}

	// Check if this is a wrapper that might contain a joined error
	if unwrapped := errors.Unwrap(err); unwrapped != nil {
		// Check if the unwrapped error is a joined error
		if joinedErrors := Flatten(unwrapped); len(joinedErrors) > 1 {
			// The wrapped error was a join, return its flattened errors
			return joinedErrors
		}
	}

	// Not a joined error, return as single item
	return []error{err}
}
