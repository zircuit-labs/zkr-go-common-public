package errcontext_test

import (
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
	assert.Equal(t, errcontext.Context{slog.String("test", "test")}, errcontext.Get(err))
}

// TestAddContext validates that context can be added and retrieved.
func TestAddContext(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		testName string
		err      error
		contexts []errcontext.Context
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
			contexts: []errcontext.Context{{
				slog.String("one", "one"),
			}},
		},
		{
			testName: "double-sized single context",
			err:      errTest,
			contexts: []errcontext.Context{{
				slog.String("one", "one"),
				slog.String("two", "two"),
			}},
		},
		{
			testName: "two single contexts",
			err:      errTest,
			contexts: []errcontext.Context{
				{slog.String("one", "one")},
				{slog.String("two", "two")},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			t.Parallel()
			err := tc.err
			var expected errcontext.Context
			for _, context := range tc.contexts {
				expected = append(expected, context...)
				err = errcontext.Add(err, context...)
			}

			actual := errcontext.Get(err)
			assert.Equal(t, expected, actual)
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
	assert.Equal(t, errcontext.Context{slog.String("one", "one"), slog.String("two", "two")}, errcontext.Get(err))
}
