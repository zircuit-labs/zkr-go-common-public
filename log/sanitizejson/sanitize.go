package sanitizejson

import (
	"log/slog"
	"strings"
)

// Key replaces dots with underscores to prevent nested object interpretation
func Key(key string) string {
	// Replace dots with underscores
	// These characters are commonly interpreted as path separators
	key = strings.ReplaceAll(key, ".", "_")
	return key
}

// SanitizeAttrs recursively sanitizes keys in slog.Attr structures
func SanitizeAttrs(attrs []slog.Attr) []slog.Attr {
	sanitized := make([]slog.Attr, len(attrs))
	for i, attr := range attrs {
		sanitized[i] = SanitizeAttr(attr)
	}
	return sanitized
}

// SanitizeAttr sanitizes a single slog.Attr, including nested groups
func SanitizeAttr(attr slog.Attr) slog.Attr {
	safeKey := Key(attr.Key)

	if attr.Value.Kind() == slog.KindGroup {
		// Recursively sanitize group attributes
		groupAttrs := attr.Value.Group()
		sanitizedGroupAttrs := SanitizeAttrs(groupAttrs)
		return slog.GroupAttrs(safeKey, sanitizedGroupAttrs...)
	}

	return slog.Attr{Key: safeKey, Value: attr.Value}
}
