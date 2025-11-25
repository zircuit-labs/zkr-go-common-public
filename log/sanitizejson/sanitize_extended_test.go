package sanitizejson

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeAttrs(t *testing.T) {
	t.Parallel()

	t.Run("EmptyAttrs", func(t *testing.T) {
		t.Parallel()
		attrs := []slog.Attr{}
		result := SanitizeAttrs(attrs)
		assert.Empty(t, result)
	})

	t.Run("SimpleAttrs", func(t *testing.T) {
		t.Parallel()
		attrs := []slog.Attr{
			slog.String("simple.key", "value1"),
			slog.Int("another.key", 42),
			slog.Bool("third.key", true),
		}

		result := SanitizeAttrs(attrs)

		assert.Len(t, result, 3)
		assert.Equal(t, "simple_key", result[0].Key)
		assert.Equal(t, "value1", result[0].Value.String())
		assert.Equal(t, "another_key", result[1].Key)
		assert.Equal(t, int64(42), result[1].Value.Int64())
		assert.Equal(t, "third_key", result[2].Key)
		assert.True(t, result[2].Value.Bool())
	})

	t.Run("NestedGroupAttrs", func(t *testing.T) {
		t.Parallel()
		nestedAttrs := []slog.Attr{
			slog.String("nested.field", "nested_value"),
			slog.Int("count.field", 5),
		}

		attrs := []slog.Attr{
			slog.String("top.level", "top_value"),
			slog.GroupAttrs("group.name", nestedAttrs...),
		}

		result := SanitizeAttrs(attrs)

		assert.Len(t, result, 2)
		assert.Equal(t, "top_level", result[0].Key)
		assert.Equal(t, "top_value", result[0].Value.String())

		// Check group
		assert.Equal(t, "group_name", result[1].Key)
		assert.Equal(t, slog.KindGroup, result[1].Value.Kind())

		groupAttrs := result[1].Value.Group()
		assert.Len(t, groupAttrs, 2)
		assert.Equal(t, "nested_field", groupAttrs[0].Key)
		assert.Equal(t, "nested_value", groupAttrs[0].Value.String())
		assert.Equal(t, "count_field", groupAttrs[1].Key)
		assert.Equal(t, int64(5), groupAttrs[1].Value.Int64())
	})

	t.Run("ComplexErrorTypeKeys", func(t *testing.T) {
		t.Parallel()
		attrs := []slog.Attr{
			slog.String("github.com/org/repo.Type[Param]", "type_value"),
			slog.String("github.com/zircuit-labs/zkr-go-common/xerrors.ExtendedError[github.com/zircuit-labs/zkr-go-common/xerrors/errclass.Class]", "class_value"),
		}

		result := SanitizeAttrs(attrs)

		assert.Len(t, result, 2)
		assert.Equal(t, "github_com/org/repo_Type[Param]", result[0].Key)
		assert.Equal(t, "type_value", result[0].Value.String())
		assert.Equal(t, "github_com/zircuit-labs/zkr-go-common/xerrors_ExtendedError[github_com/zircuit-labs/zkr-go-common/xerrors/errclass_Class]", result[1].Key)
		assert.Equal(t, "class_value", result[1].Value.String())
	})
}

func TestSanitizeAttr(t *testing.T) {
	t.Parallel()

	t.Run("SimpleStringAttr", func(t *testing.T) {
		t.Parallel()
		attr := slog.String("test.key", "test_value")
		result := SanitizeAttr(attr)

		assert.Equal(t, "test_key", result.Key)
		assert.Equal(t, "test_value", result.Value.String())
	})

	t.Run("GroupAttr", func(t *testing.T) {
		t.Parallel()
		nestedAttrs := []slog.Attr{
			slog.String("inner.key1", "value1"),
			slog.String("inner.key2", "value2"),
		}
		attr := slog.GroupAttrs("outer.group", nestedAttrs...)
		result := SanitizeAttr(attr)

		assert.Equal(t, "outer_group", result.Key)
		assert.Equal(t, slog.KindGroup, result.Value.Kind())

		groupAttrs := result.Value.Group()
		assert.Len(t, groupAttrs, 2)
		assert.Equal(t, "inner_key1", groupAttrs[0].Key)
		assert.Equal(t, "value1", groupAttrs[0].Value.String())
		assert.Equal(t, "inner_key2", groupAttrs[1].Key)
		assert.Equal(t, "value2", groupAttrs[1].Value.String())
	})

	t.Run("DeeplyNestedGroups", func(t *testing.T) {
		t.Parallel()
		deepestAttrs := []slog.Attr{
			slog.String("deep.key", "deep_value"),
		}
		middleAttrs := []slog.Attr{
			slog.GroupAttrs("inner.group", deepestAttrs...),
		}
		attr := slog.GroupAttrs("outer.group", middleAttrs...)

		result := SanitizeAttr(attr)

		assert.Equal(t, "outer_group", result.Key)
		outerGroup := result.Value.Group()
		assert.Len(t, outerGroup, 1)

		assert.Equal(t, "inner_group", outerGroup[0].Key)
		innerGroup := outerGroup[0].Value.Group()
		assert.Len(t, innerGroup, 1)

		assert.Equal(t, "deep_key", innerGroup[0].Key)
		assert.Equal(t, "deep_value", innerGroup[0].Value.String())
	})

	t.Run("NonGroupAttr", func(t *testing.T) {
		t.Parallel()
		testCases := []struct {
			name        string
			attr        slog.Attr
			expectedKey string
		}{
			{"Int", slog.Int("test.int", 42), "test_int"},
			{"Bool", slog.Bool("test.bool", true), "test_bool"},
			{"Float64", slog.Float64("test.float", 3.14), "test_float"},
			{"Any", slog.Any("test.any", "any_value"), "test_any"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				result := SanitizeAttr(tc.attr)
				assert.Equal(t, tc.expectedKey, result.Key)
				assert.Equal(t, tc.attr.Value, result.Value)
			})
		}
	})
}

func TestKey_AdditionalCases(t *testing.T) {
	t.Parallel()

	t.Run("KeysWithoutDots", func(t *testing.T) {
		t.Parallel()
		testCases := []string{
			"simple_key",
			"alreadySafe",
			"UPPERCASE",
			"123numeric",
			"with-hyphens",
		}

		for _, key := range testCases {
			result := Key(key)
			assert.Equal(t, key, result, "Key without dots should remain unchanged")
		}
	})

	t.Run("KeysWithMultipleDots", func(t *testing.T) {
		t.Parallel()
		testCases := []struct {
			input    string
			expected string
		}{
			{"a.b.c.d", "a_b_c_d"},
			{"package.name.Type.method", "package_name_Type_method"},
			{"...multiple.dots...", "___multiple_dots___"},
			{"start.middle.end", "start_middle_end"},
		}

		for _, tc := range testCases {
			result := Key(tc.input)
			assert.Equal(t, tc.expected, result)
		}
	})

	t.Run("EmptyKey", func(t *testing.T) {
		t.Parallel()
		result := Key("")
		assert.Equal(t, "", result)
	})

	t.Run("OnlyDots", func(t *testing.T) {
		t.Parallel()
		result := Key("...")
		assert.Equal(t, "___", result)
	})
}
