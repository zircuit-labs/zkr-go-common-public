// Package errclass provides functions for simple error classification.
package errclass

import (
	"log/slog"

	"github.com/zircuit-labs/zkr-go-common/xerrors"
)

// Class represents a represents a type of error.
type Class int

// These are the allowed error classifications.
// The values are arbitrary but provide a strict ordering,
// where the higher the value, the more severe the error.
// When determining the class of a joined error, the highest
// class is returned.
const (
	Nil     Class = -1
	Unknown Class = 0

	Transient  Class = 100
	Persistent Class = 110

	Panic Class = 900
)

// String implements stringer interface.
func (c Class) String() string {
	switch c {
	case Nil:
		return "nil"
	case Panic:
		return "panic"
	case Transient:
		return "transient"
	case Persistent:
		return "persistent"
	default:
		return "unknown"
	}
}

// LogValue implements slog.LogValuer for Class.
// It returns the class in a structured format.
func (c Class) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("class", c.String()),
	)
}

// WrapAs extends an error with the given class data.
func WrapAs(err error, class Class) error {
	if err == nil {
		return nil
	}
	return xerrors.Extend(class, err)
}

// GetClass extracts the Class from an error.
// If the error directly has a class (e.g., from WrapAs), that class is returned.
// Otherwise, for joined errors, it recursively checks direct children and returns
// the maximum class found. This preserves hierarchical override semantics where
// explicitly wrapped joined errors take precedence over their contents.
func GetClass(err error) Class {
	if err == nil {
		return Nil
	}

	// Check if this error is DIRECTLY an ExtendedError with a Class
	// (not using Extract which would traverse into joined children)
	if extended, ok := err.(xerrors.ExtendedError[Class]); ok { //nolint:errorlint // intentionally not using errors.As
		return extended.Data
	}

	// Check if this is a joined error
	// A joined error implements Unwrap() []error
	type multiError interface {
		Unwrap() []error
	}

	if _, isJoined := err.(multiError); isJoined {
		// For joined errors, recursively check each child
		directChildren := xerrors.Unjoin(err)
		maxClass := Nil
		for _, child := range directChildren {
			childClass := GetClass(child)
			if childClass > maxClass {
				maxClass = childClass
			}
		}

		// If still no class found, return Unknown
		if maxClass == Nil {
			return Unknown
		}

		return maxClass
	}

	// Not a joined error and not directly ExtendedError[Class]
	// Use Extract to find class in the error chain
	if class, ok := xerrors.Extract[Class](err); ok {
		return class
	}

	return Unknown
}
