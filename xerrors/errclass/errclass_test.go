package errclass_test

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zircuit-labs/zkr-go-common/xerrors/errclass"
)

var (
	errTest    = fmt.Errorf("this is a test error")
	errTestToo = fmt.Errorf("this is also test error")
)

func TestErrClass(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		testName string
		err      error
		class    errclass.Class
	}{
		{
			testName: "nil error",
			err:      nil,
			class:    errclass.Nil,
		},
		{
			testName: "unknown error",
			err:      errTest,
			class:    errclass.Unknown,
		},
		{
			testName: "panic error",
			err:      errTest,
			class:    errclass.Panic,
		},
		{
			testName: "transient error",
			err:      errTest,
			class:    errclass.Transient,
		},
		{
			testName: "persistent error",
			err:      errTest,
			class:    errclass.Persistent,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			t.Parallel()
			err := errclass.WrapAs(tc.err, tc.class)
			class := errclass.GetClass(err)
			if class != tc.class {
				t.Errorf("unexpected error class: want: %s got %s", tc.class, class)
			}
		})
	}
}

func TestErrClassUnknown(t *testing.T) {
	t.Parallel()

	// errTest doesn't have a class assigned
	class := errclass.GetClass(errTest)
	if class != errclass.Unknown {
		t.Errorf("unexpected error class: want: %s got %s", errclass.Unknown, class)
	}
}

func TestErrClassJoined(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		testName      string
		classA        errclass.Class
		classB        errclass.Class
		expectedClass errclass.Class
	}{
		{
			testName:      "nil nil",
			classA:        errclass.Nil,
			classB:        errclass.Nil,
			expectedClass: errclass.Nil,
		},
		{
			testName:      "nil unknown",
			classA:        errclass.Nil,
			classB:        errclass.Unknown,
			expectedClass: errclass.Unknown,
		},
		{
			testName:      "nil transient",
			classA:        errclass.Nil,
			classB:        errclass.Transient,
			expectedClass: errclass.Transient,
		},
		{
			testName:      "nil persistent",
			classA:        errclass.Nil,
			classB:        errclass.Persistent,
			expectedClass: errclass.Persistent,
		},
		{
			testName:      "nil panic",
			classA:        errclass.Nil,
			classB:        errclass.Panic,
			expectedClass: errclass.Panic,
		},
		{
			testName:      "unknown unknown",
			classA:        errclass.Unknown,
			classB:        errclass.Unknown,
			expectedClass: errclass.Unknown,
		},
		{
			testName:      "unknown transient",
			classA:        errclass.Unknown,
			classB:        errclass.Transient,
			expectedClass: errclass.Transient,
		},
		{
			testName:      "unknown persistent",
			classA:        errclass.Unknown,
			classB:        errclass.Persistent,
			expectedClass: errclass.Persistent,
		},
		{
			testName:      "unknown panic",
			classA:        errclass.Unknown,
			classB:        errclass.Panic,
			expectedClass: errclass.Panic,
		},
		{
			testName:      "transient transient",
			classA:        errclass.Transient,
			classB:        errclass.Transient,
			expectedClass: errclass.Transient,
		},
		{
			testName:      "transient persistent",
			classA:        errclass.Transient,
			classB:        errclass.Persistent,
			expectedClass: errclass.Persistent,
		},
		{
			testName:      "transient panic",
			classA:        errclass.Transient,
			classB:        errclass.Panic,
			expectedClass: errclass.Panic,
		},
		{
			testName:      "persistent persistent",
			classA:        errclass.Persistent,
			classB:        errclass.Persistent,
			expectedClass: errclass.Persistent,
		},
		{
			testName:      "persistent panic",
			classA:        errclass.Persistent,
			classB:        errclass.Panic,
			expectedClass: errclass.Panic,
		},
		{
			testName:      "panic panic",
			classA:        errclass.Panic,
			classB:        errclass.Panic,
			expectedClass: errclass.Panic,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			t.Parallel()
			errA := errclass.WrapAs(errTest, tc.classA)
			if tc.classA == errclass.Nil {
				errA = nil
			}
			errB := errclass.WrapAs(errTestToo, tc.classB)
			if tc.classB == errclass.Nil {
				errB = nil
			}

			errC := errors.Join(errA, errB)
			class := errclass.GetClass(errC)
			if class != tc.expectedClass {
				t.Errorf("unexpected error class: want: %s got %s", tc.expectedClass, class)
			}

			// Joining in the opposite order should yield the same result
			errD := errors.Join(errB, errA)
			class = errclass.GetClass(errD)
			if class != tc.expectedClass {
				t.Errorf("unexpected error class: want: %s got %s", tc.expectedClass, class)
			}
		})
	}
}

// TestGetClassNestedJoined tests GetClass with nested joined errors.
func TestGetClassNestedJoined(t *testing.T) {
	t.Parallel()

	// Create base errors
	errA := errors.New("error A")
	errB := errors.New("error B")
	errC := errors.New("error C")
	errD := errors.New("error D")

	// Wrap with different classes
	errATransient := errclass.WrapAs(errA, errclass.Transient)
	errBPanic := errclass.WrapAs(errB, errclass.Panic)
	errCPersistent := errclass.WrapAs(errC, errclass.Persistent)
	errDUnknown := errclass.WrapAs(errD, errclass.Unknown)

	// Create nested joined structure: Join(Join(A,B), Join(C,D))
	errAB := errors.Join(errATransient, errBPanic)
	errCD := errors.Join(errCPersistent, errDUnknown)
	errABCD := errors.Join(errAB, errCD)

	// Without override, should find the maximum class (Panic)
	class := errclass.GetClass(errABCD)
	if class != errclass.Panic {
		t.Errorf("nested joined error without override: want %s, got %s", errclass.Panic, class)
	}

	// Now test hierarchical override: wrap the joined AB with Transient
	errABWrapped := errclass.WrapAs(errAB, errclass.Transient)
	errABCDWithOverride := errors.Join(errABWrapped, errCD)

	// Should find Persistent (max of Transient from AB override and Persistent from C)
	class = errclass.GetClass(errABCDWithOverride)
	if class != errclass.Persistent {
		t.Errorf("nested joined with override: want %s, got %s", errclass.Persistent, class)
	}

	// Test wrapping the entire nested structure
	errAllWrapped := errclass.WrapAs(errABCD, errclass.Unknown)
	class = errclass.GetClass(errAllWrapped)
	if class != errclass.Unknown {
		t.Errorf("wrapped nested joined: want %s, got %s", errclass.Unknown, class)
	}
}

// TestGetClassHierarchicalOverride tests that explicit class wrapping on joined errors
// takes precedence over the classes of contained errors.
func TestGetClassHierarchicalOverride(t *testing.T) {
	t.Parallel()

	// Create errors with high severity classes
	errPanic1 := errclass.WrapAs(errors.New("panic 1"), errclass.Panic)
	errPanic2 := errclass.WrapAs(errors.New("panic 2"), errclass.Panic)

	// Join them
	joinedPanics := errors.Join(errPanic1, errPanic2)

	// Without override, should return Panic
	if class := errclass.GetClass(joinedPanics); class != errclass.Panic {
		t.Errorf("joined panics without override: want %s, got %s", errclass.Panic, class)
	}

	// Explicitly override the joined error to Transient (lower severity)
	overriddenJoined := errclass.WrapAs(joinedPanics, errclass.Transient)

	// The explicit override should win
	if class := errclass.GetClass(overriddenJoined); class != errclass.Transient {
		t.Errorf("overridden joined panics: want %s, got %s", errclass.Transient, class)
	}

	// Now join this overridden error with another error
	errPersistent := errclass.WrapAs(errors.New("persistent"), errclass.Persistent)
	finalJoined := errors.Join(overriddenJoined, errPersistent)

	// Should find Persistent (max of Transient override and Persistent)
	if class := errclass.GetClass(finalJoined); class != errclass.Persistent {
		t.Errorf("joined with overridden: want %s, got %s", errclass.Persistent, class)
	}
}

// TestGetClassComplexNesting tests a complex real-world scenario.
func TestGetClassComplexNesting(t *testing.T) {
	t.Parallel()

	// Simulate a real-world scenario:
	// - Database errors are Transient (can retry)
	// - Validation errors are Persistent (won't succeed on retry)
	// - System errors are Panic (critical failures)

	dbErr1 := errclass.WrapAs(errors.New("connection timeout"), errclass.Transient)
	dbErr2 := errclass.WrapAs(errors.New("deadlock"), errclass.Transient)
	dbErrors := errors.Join(dbErr1, dbErr2)

	validationErr := errclass.WrapAs(errors.New("invalid input"), errclass.Persistent)

	// First operation: DB errors + validation
	operation1Errs := errors.Join(dbErrors, validationErr)

	// Operation1 should be Persistent (can't retry invalid input)
	if class := errclass.GetClass(operation1Errs); class != errclass.Persistent {
		t.Errorf("operation1: want %s, got %s", errclass.Persistent, class)
	}

	// Admin override: treat all operation1 errors as Transient
	operation1Override := errclass.WrapAs(operation1Errs, errclass.Transient)

	// System panic occurs
	systemErr := errclass.WrapAs(errors.New("out of memory"), errclass.Panic)

	// Final error combines overridden operation1 with system error
	finalErr := errors.Join(operation1Override, systemErr)

	// Should be Panic (highest severity)
	if class := errclass.GetClass(finalErr); class != errclass.Panic {
		t.Errorf("final error: want %s, got %s", errclass.Panic, class)
	}
}

// TestClassString tests the String() method for all Class values.
func TestClassString(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		class    errclass.Class
		expected string
	}{
		{errclass.Nil, "nil"},
		{errclass.Unknown, "unknown"},
		{errclass.Transient, "transient"},
		{errclass.Persistent, "persistent"},
		{errclass.Panic, "panic"},
		{errclass.Class(999), "unknown"}, // Test default case for undefined value
	}

	for _, tc := range testCases {
		t.Run(tc.expected, func(t *testing.T) {
			t.Parallel()
			result := tc.class.String()
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestClassLogValue tests the LogValue() method.
func TestClassLogValue(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		class    errclass.Class
		expected string
	}{
		{"nil", errclass.Nil, "nil"},
		{"unknown", errclass.Unknown, "unknown"},
		{"transient", errclass.Transient, "transient"},
		{"persistent", errclass.Persistent, "persistent"},
		{"panic", errclass.Panic, "panic"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Create a logger that writes to a buffer
			var buf bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
				ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
					// Remove time for consistent testing
					if a.Key == slog.TimeKey {
						return slog.Attr{}
					}
					return a
				},
			}))

			// Log the class
			logger.Info("test", slog.Any("errclass", tc.class))

			// Check that the expected class string appears in the log
			logOutput := buf.String()
			assert.Contains(t, logOutput, tc.expected)
			assert.Contains(t, logOutput, "errclass.class="+tc.expected)
		})
	}
}

// TestWrapAsNilError tests WrapAs with nil error.
func TestWrapAsNilError(t *testing.T) {
	t.Parallel()

	result := errclass.WrapAs(nil, errclass.Panic)
	assert.Nil(t, result)
}

// TestGetClassEdgeCases tests edge cases for GetClass.
func TestGetClassEdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("nil error", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, errclass.Nil, errclass.GetClass(nil))
	})

	t.Run("plain error without class", func(t *testing.T) {
		t.Parallel()
		err := errors.New("plain error")
		assert.Equal(t, errclass.Unknown, errclass.GetClass(err))
	})

	t.Run("wrapped error with class in chain", func(t *testing.T) {
		t.Parallel()
		baseErr := errors.New("base")
		classedErr := errclass.WrapAs(baseErr, errclass.Transient)
		wrappedErr := fmt.Errorf("wrapped: %w", classedErr)
		assert.Equal(t, errclass.Transient, errclass.GetClass(wrappedErr))
	})

	t.Run("empty joined error", func(t *testing.T) {
		t.Parallel()
		// errors.Join with no arguments returns nil
		joined := errors.Join()
		assert.Equal(t, errclass.Nil, errclass.GetClass(joined))
	})

	t.Run("joined error with single nil", func(t *testing.T) {
		t.Parallel()
		// errors.Join with single nil returns nil
		joined := errors.Join(nil)
		assert.Equal(t, errclass.Nil, errclass.GetClass(joined))
	})

	t.Run("joined error with nil and non-nil", func(t *testing.T) {
		t.Parallel()
		err := errclass.WrapAs(errors.New("test"), errclass.Panic)
		joined := errors.Join(nil, err, nil)
		assert.Equal(t, errclass.Panic, errclass.GetClass(joined))
	})

	t.Run("joined error with all Nil-classed children", func(t *testing.T) {
		t.Parallel()
		// Create errors explicitly wrapped with Nil class
		err1 := errclass.WrapAs(errors.New("error1"), errclass.Nil)
		err2 := errclass.WrapAs(errors.New("error2"), errclass.Nil)
		joined := errors.Join(err1, err2)
		// This should trigger the maxClass == Nil case and return Unknown
		assert.Equal(t, errclass.Unknown, errclass.GetClass(joined))
	})

	t.Run("deeply wrapped joined error", func(t *testing.T) {
		t.Parallel()
		err1 := errclass.WrapAs(errors.New("err1"), errclass.Transient)
		err2 := errclass.WrapAs(errors.New("err2"), errclass.Persistent)
		joined := errors.Join(err1, err2)
		wrapped := fmt.Errorf("context: %w", joined)
		// When a joined error is wrapped with fmt.Errorf, it's no longer a joined error
		// Extract will find the first class it encounters (Transient in this case)
		// This is expected behavior - if you want to preserve the max class semantics,
		// use errclass.WrapAs instead of fmt.Errorf
		assert.Equal(t, errclass.Transient, errclass.GetClass(wrapped))
	})

	t.Run("joined error wrapped with class", func(t *testing.T) {
		t.Parallel()
		err1 := errclass.WrapAs(errors.New("err1"), errclass.Transient)
		err2 := errclass.WrapAs(errors.New("err2"), errclass.Persistent)
		joined := errors.Join(err1, err2)
		// Wrapping with WrapAs respects hierarchical override
		wrapped := errclass.WrapAs(joined, errclass.Unknown)
		assert.Equal(t, errclass.Unknown, errclass.GetClass(wrapped))
	})

	t.Run("multiple levels of wrapping", func(t *testing.T) {
		t.Parallel()
		err := errors.New("base")
		err = errclass.WrapAs(err, errclass.Transient)
		err = fmt.Errorf("level1: %w", err)
		err = fmt.Errorf("level2: %w", err)
		err = fmt.Errorf("level3: %w", err)
		assert.Equal(t, errclass.Transient, errclass.GetClass(err))
	})
}

// TestGetClassWithCustomErrorTypes tests GetClass with custom error types.
func TestGetClassWithCustomErrorTypes(t *testing.T) {
	t.Parallel()

	t.Run("custom error without class", func(t *testing.T) {
		t.Parallel()
		err := &customError{msg: "custom"}
		assert.Equal(t, errclass.Unknown, errclass.GetClass(err))
	})

	t.Run("wrapped custom error with class", func(t *testing.T) {
		t.Parallel()
		err := &customError{msg: "custom"}
		wrappedErr := errclass.WrapAs(err, errclass.Persistent)
		assert.Equal(t, errclass.Persistent, errclass.GetClass(wrappedErr))
	})
}

// customError is a custom error type for testing.
type customError struct {
	msg string
}

func (e *customError) Error() string {
	return e.msg
}
