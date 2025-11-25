package errcontext_test

import (
	"errors"
	"fmt"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/zircuit-labs/zkr-go-common/xerrors"
	"github.com/zircuit-labs/zkr-go-common/xerrors/errclass"
	"github.com/zircuit-labs/zkr-go-common/xerrors/errcontext"
)

var errTest = fmt.Errorf("this is a test error")

// TestErrorAs validates that the contextualized error can be cast properly.
func TestErrorAs(t *testing.T) {
	t.Parallel()

	err := errcontext.Add(errTest, slog.String("test", "test"))
	assert.ErrorIs(t, err, errTest)
	extendedError := xerrors.ExtendedError[errcontext.Context]{}
	assert.ErrorAs(t, err, &extendedError)
	assert.Equal(t, []slog.Attr{slog.String("test", "test")}, errcontext.Get(err).Flatten())
}

// TestAddContext validates that context can be added and retrieved.
func TestAddContext(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		testName string
		err      error
		contexts [][]slog.Attr
	}{
		{
			testName: "nil error",
			err:      nil,
			contexts: nil,
		},
		{
			testName: "no context",
			err:      errTest,
			contexts: nil,
		},
		{
			testName: "single context",
			err:      errTest,
			contexts: [][]slog.Attr{
				{slog.String("one", "one")},
			},
		},
		{
			testName: "double-sized single context",
			err:      errTest,
			contexts: [][]slog.Attr{
				{slog.String("one", "one"), slog.String("two", "two")},
			},
		},
		{
			testName: "two single contexts",
			err:      errTest,
			contexts: [][]slog.Attr{
				{slog.String("one", "one")},
				{slog.String("two", "two")},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			t.Parallel()
			err := tc.err
			var expected []slog.Attr
			for _, context := range tc.contexts {
				expected = append(expected, context...)
				err = errcontext.Add(err, context...)
			}

			actual := errcontext.Get(err).Flatten()
			assert.ElementsMatch(t, expected, actual)
		})
	}
}

// TestAddContextOverOthers validates that context can be added multiple times.
func TestAddContextOverOthers(t *testing.T) {
	t.Parallel()

	// add some context
	err := errcontext.Add(errTest, slog.String("one", "one"))

	// wrap the error in a different way (add a class)
	err = errclass.WrapAs(err, errclass.Transient)

	// add some more context
	err = errcontext.Add(err, slog.String("two", "two"))

	// ensure the class remains
	assert.Equal(t, errclass.Transient, errclass.GetClass(err))

	// ensure all added context is present
	assert.ElementsMatch(t, []slog.Attr{slog.String("one", "one"), slog.String("two", "two")}, errcontext.Get(err).Flatten())

	// add context with duplicate key - this overwrites (last-entry-wins)
	err = errcontext.Add(err, slog.String("two", "three"))

	// ensure the duplicate key was overwritten with the new value
	assert.ElementsMatch(t, []slog.Attr{
		slog.String("one", "one"),
		slog.String("two", "three"), // overwritten value (last-entry-wins)
	}, errcontext.Get(err).Flatten())
}

// TestAddContextToJoinedErrors demonstrates the structure-preserving behavior with joined errors.
// errcontext.Add now applies context to direct children while preserving the joined structure,
// similar to how stacktrace.Wrap works.
func TestAddContextToJoinedErrors(t *testing.T) {
	t.Parallel()

	// Create individual errors
	errA := errors.New("test error A")
	errB := errors.New("test error B")
	errC := errors.New("test error C")
	errD := errors.New("test error D")
	errE := errors.New("test error E")

	// Create complex joined error structure
	errAB := errors.Join(errA, errcontext.Add(errB, slog.String("context_B", "from_B")))
	errCD := errcontext.Add(errors.Join(errC, errD), slog.String("context_CD", "from_CD"))
	errCDE := errors.Join(errE, errCD)

	errABCDE := errors.Join(errAB, errCDE)

	// The test: add context to the final joined error
	finalErr := errcontext.Add(errABCDE, slog.String("final_context", "from_final"))

	// Verify the structure is still a joined error
	type multiError interface {
		Unwrap() []error
	}
	if multi, ok := finalErr.(multiError); ok {
		t.Logf("Final error is still a joined error with %d top-level errors", len(multi.Unwrap()))
		assert.Equal(t, 2, len(multi.Unwrap()), "Should have 2 top-level errors after structure-preserving context distribution")
	} else {
		t.Fatal("Final error should still be a joined error after context distribution")
	}

	individualErrors := xerrors.Flatten(finalErr)
	t.Logf("Found %d individual errors after flattening", len(individualErrors))

	// With the new behavior, we expect 5 individual errors
	assert.Equal(t, 5, len(individualErrors), "Should have 5 individual errors after context distribution")

	// Verify each individual error contains the expected contextual data
	for i, err := range individualErrors {
		ctx := errcontext.Get(err)
		assert.NotNil(t, ctx, "Error %d should have context", i)

		contextAttrs := ctx.Flatten()

		// Check error-specific context based on error message
		errorMsg := err.Error()
		switch errorMsg {
		case "test error A", "test error E":
			// should only have final_context
			assert.Equal(t, []slog.Attr{slog.String("final_context", "from_final")}, contextAttrs)
		case "test error B":
			// errB should have context_B and final_context
			expected := []slog.Attr{
				slog.String("context_B", "from_B"),
				slog.String("final_context", "from_final"),
			}
			assert.ElementsMatch(t, expected, contextAttrs)
		case "test error C", "test error D":
			// errC and errD should have context_CD and final_context
			expected := []slog.Attr{
				slog.String("context_CD", "from_CD"),
				slog.String("final_context", "from_final"),
			}
			assert.ElementsMatch(t, expected, contextAttrs)
		default:
			t.Errorf("Unexpected error message: %s", errorMsg)
		}
	}
}

// TestLogValue validates that Context.LogValue() works correctly.
func TestLogValue(t *testing.T) {
	t.Parallel()

	// Test empty context
	emptyContext := errcontext.Context{}
	assert.Zero(t, emptyContext.LogValue())

	// Test context with values
	context := errcontext.Context{
		"key1": slog.StringValue("value1"),
		"key2": slog.IntValue(42),
	}

	logValue := context.LogValue()
	assert.Equal(t, slog.KindGroup, logValue.Kind())

	// Check that the group contains the expected attributes
	attrs := logValue.Group()
	assert.Len(t, attrs, 2)

	// Convert to map for order-independent comparison
	attrMap := make(map[string]slog.Value)
	for _, attr := range attrs {
		attrMap[attr.Key] = attr.Value
	}

	expectedMap := map[string]slog.Value{
		"key1": slog.StringValue("value1"),
		"key2": slog.IntValue(42),
	}
	assert.Equal(t, expectedMap, attrMap)
}

// TestAddNilError validates that Add with nil error returns nil.
func TestAddNilError(t *testing.T) {
	t.Parallel()

	result := errcontext.Add(nil, slog.String("key", "value"))
	assert.Nil(t, result)
}
