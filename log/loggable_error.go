package log

import (
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"strings"

	"github.com/zircuit-labs/zkr-go-common/log/sanitizejson"
	"github.com/zircuit-labs/zkr-go-common/replaceattrmore"
	"github.com/zircuit-labs/zkr-go-common/xerrors"
)

// collectLogValuerAttrs walks an error chain and collects slog.LogValuer data as sanitized attributes.
func collectLogValuerAttrs(err error) []slog.Attr {
	var attrs []slog.Attr
	for e := err; e != nil; e = errors.Unwrap(e) {
		if lv, ok := e.(slog.LogValuer); ok {
			typePath := getTypePath(e)
			logValue := lv.LogValue()

			if logValue.Kind() == slog.KindGroup {
				// If it's already a group, create a nested group with the type path
				// Convert group values to proper format using slogValueToAny
				groupAttrs := logValue.Group()
				convertedAttrs := make([]slog.Attr, len(groupAttrs))
				for i, attr := range groupAttrs {
					convertedAttrs[i] = slog.Any(attr.Key, slogValueToAny(attr.Value))
				}
				attrs = append(attrs, slog.GroupAttrs(typePath, convertedAttrs...))
			} else {
				// For non-group values, create a single attribute
				attrs = append(attrs, slog.Any(typePath, slogValueToAny(logValue)))
			}
		}
	}

	// Apply sanitization to all attributes
	return sanitizejson.SanitizeAttrs(attrs)
}

// getTypePath extracts a stable type path for logging keys, handling pointer types correctly
func getTypePath(err error) string {
	errType := reflect.TypeOf(err)
	if errType.Kind() == reflect.Ptr {
		errType = errType.Elem()
	}

	pkg, name := errType.PkgPath(), errType.Name()
	if pkg == "" || name == "" {
		// Fallback for unnamed/builtin or otherwise unnameable types
		return errType.String()
	}
	return pkg + "." + name
}

// LoggableError wraps an error to provide custom logging behavior
type LoggableError struct {
	err error
}

// Error implements the error interface.
func (le LoggableError) Error() string {
	if le.err == nil {
		return ""
	}
	return le.err.Error()
}

// Unwrap returns the wrapped error.
func (le LoggableError) Unwrap() error {
	return le.err
}

// Loggable wraps an error to enable custom logging behavior.
func Loggable(err error) LoggableError {
	return LoggableError{err: err}
}

// NewLoggableErrorHandler creates a chained handler using replaceattrmore.Handler
// to flatten LoggableError structures with any underlying slog.Handler
func NewLoggableErrorHandler(next slog.Handler) slog.Handler {
	replaceFunc := func(groups []string, a slog.Attr) []slog.Attr {
		a.Value = a.Value.Resolve()
		// Handle LoggableError flattening
		if a.Key == ErrorKey && a.Value.Kind() == slog.KindAny {
			if loggableErr, ok := a.Value.Any().(LoggableError); ok {
				return flattenLoggableError(loggableErr)
			}
		}
		// Return unchanged for all other attributes
		return []slog.Attr{a}
	}

	return replaceattrmore.New(next, replaceFunc)
}

// flattenLoggableError converts LoggableError to flat error + error_detail structure
func flattenLoggableError(loggableErr LoggableError) []slog.Attr {
	// Check if this is a joined error (implements Unwrap() []error)
	if joinedErrors := xerrors.Flatten(loggableErr.err); len(joinedErrors) > 1 {
		// Handle joined errors specially (only if we have multiple errors)
		return flattenJoinedErrors(joinedErrors)
	}

	// Original single error handling
	attrs := []slog.Attr{
		slog.String(ErrorKey, loggableErr.Error()),
	}

	// Collect error_detail as attributes from the error chain
	if errorDetailAttrs := collectLogValuerAttrs(loggableErr.err); len(errorDetailAttrs) > 0 {
		attrs = append(attrs, slog.GroupAttrs("error_detail", errorDetailAttrs...))
	}

	return attrs
}

// flattenJoinedErrors creates attributes for joined errors
func flattenJoinedErrors(errs []error) []slog.Attr {
	// Create array of error messages
	errorMessages := make([]string, len(errs))
	for i, err := range errs {
		errorMessages[i] = err.Error()
	}

	attrs := []slog.Attr{
		slog.String(ErrorKey, strings.Join(errorMessages, "; ")),
		slog.Any("errors", errorMessages),
	}

	// Build error_detail using GroupAttrs for each individual error
	errorDetailAttrs := make([]slog.Attr, 0, len(errs))

	for i, err := range errs {
		key := fmt.Sprintf("error_%d", i)

		// Collect attributes for this specific error
		var thisErrorAttrs []slog.Attr
		thisErrorAttrs = append(thisErrorAttrs, slog.String("error", err.Error()))

		// Add any extended error details
		if details := collectLogValuerAttrs(err); len(details) > 0 {
			thisErrorAttrs = append(thisErrorAttrs, slog.GroupAttrs("error_detail", details...))
		}

		errorDetailAttrs = append(errorDetailAttrs, slog.GroupAttrs(key, thisErrorAttrs...))
	}

	if len(errorDetailAttrs) > 0 {
		attrs = append(attrs, slog.GroupAttrs("error_detail", errorDetailAttrs...))
	}

	return attrs
}

// slogValueToAny converts a slog.Value to an any type for JSON encoding
func slogValueToAny(v slog.Value) any {
	switch v.Kind() {
	case slog.KindDuration:
		// Duration needs string representation for readability
		return v.Duration().String()
	case slog.KindGroup:
		// Convert group to map
		groupMap := make(map[string]any)
		for _, attr := range v.Group() {
			groupMap[attr.Key] = slogValueToAny(attr.Value)
		}
		return groupMap
	case slog.KindTime:
		return v.Time()
	default:
		// All other types (String, Int64, Bool, Float64, Time, Any, arrays/slices) work with Any()
		av := v.Any()
		if lv, ok := av.(slog.LogValuer); ok {
			return slogValueToAny(lv.LogValue())
		}
		return av
	}
}
