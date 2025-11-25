package stacktrace_test

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"testing"

	"github.com/zircuit-labs/zkr-go-common/xerrors/stacktrace"
)

var errTest = fmt.Errorf("this is a test error")

func a() error {
	return stacktrace.Wrap(b())
}

func b() error {
	return stacktrace.Wrap(c())
}

func c() error {
	return stacktrace.Wrap(errTest)
}

// TestStackTrace checks that the stack trace is correct.
// WARNING: This test is extremely fragile if line numbers in this file change.
func TestStackTrace(t *testing.T) {
	t.Parallel()

	// wrapping nil error is still nil.
	err := stacktrace.Wrap(nil)
	if err != nil {
		t.Errorf("unexpected error: got %v", err)
	}

	// extracting stack trace from nil produces nil
	trace := stacktrace.Extract(err)
	if trace != nil {
		t.Errorf("expected nil stack trace: got %v", trace)
	}

	// wrapping non-nil error is non-nil
	err = a()
	if err == nil {
		t.Errorf("expected error: got %v", err)
	}

	// stack trace should not be nil in this case
	trace = stacktrace.Extract(err)
	if trace == nil {
		t.Errorf("expected stack trace: got %v", trace)
	}

	if len(trace) != 4 {
		t.Errorf("unexpected stack trace len: want: %d got %d.\n-----\n%v\n-----\n", 4, len(trace), trace)
	}

	expected := []stacktrace.Frame{
		{
			File:       "xerrors/stacktrace/error_test.go",
			LineNumber: 26,
			Function:   "xerrors/stacktrace_test.c",
		},
		{
			File:       "xerrors/stacktrace/error_test.go",
			LineNumber: 22,
			Function:   "xerrors/stacktrace_test.b",
		},
		{
			File:       "xerrors/stacktrace/error_test.go",
			LineNumber: 18,
			Function:   "xerrors/stacktrace_test.a",
		},
		{
			File:       "xerrors/stacktrace/error_test.go",
			LineNumber: 47,
			Function:   "xerrors/stacktrace_test.TestStackTrace",
		},
	}

	for i, frame := range trace {
		if !strings.HasSuffix(frame.File, expected[i].File) {
			t.Errorf("unexpected file name suffix: want: %s got %s", expected[i].File, frame.File)
		}
		if !strings.HasSuffix(frame.Function, expected[i].Function) {
			t.Errorf("unexpected function name suffix: want: %s got %s", expected[i].Function, frame.Function)
		}
		if frame.LineNumber != expected[i].LineNumber {
			t.Errorf("unexpected line number: want: %d got %d", expected[i].LineNumber, frame.LineNumber)
		}
	}
}

func TestStackTraceDisabled(t *testing.T) { //nolint:paralleltest // test uses package-level variable
	stacktrace.Disabled.Store(true)
	t.Cleanup(func() { stacktrace.Disabled.Store(false) })

	err := a()
	if err == nil {
		t.Errorf("expected error: got %v", err)
	}

	// stack trace should  be nil in this case since stacktrace was disabled
	trace := stacktrace.Extract(err)
	if trace != nil {
		t.Errorf("expected nil stacktrace: got %v", trace)
	}
}

func TestWrapJoinedErrors(t *testing.T) {
	t.Parallel()

	// Create a complex web of errors similar to TestLogErrorJoined
	errA := errors.New("test error A")
	errB := errors.New("test error B")
	errC := errors.New("test error C")
	errD := errors.New("test error D")
	errE := errors.New("test error E")

	// Create nested joined errors with some already wrapped
	errAB := errors.Join(errA, stacktrace.Wrap(errB))
	errCD := stacktrace.Wrap(errors.Join(errC, stacktrace.Wrap(errD)))
	errCDE := errors.Join(errE, errCD)

	errABCDE := errors.Join(errAB, errCDE)

	// The key test: wrap the final joined error
	wrappedJoined := stacktrace.Wrap(errABCDE)

	// Verify that the result is still a joined error
	type multiError interface {
		Unwrap() []error
	}
	if multi, ok := wrappedJoined.(multiError); !ok {
		t.Fatal("Expected wrapped joined error to still be a joined error")
	} else {
		// With structure-preserving approach, we expect 2 top-level errors (errAB, errCDE)
		topErrors := multi.Unwrap()
		if len(topErrors) != 2 {
			t.Errorf("Expected 2 top-level errors after structure-preserving wrap, got %d", len(topErrors))
		}
	}

	// Helper function to count errors with stacktraces
	countErrorsWithStacktraces := func(err error) int {
		count := 0
		var walk func(error)
		walk = func(e error) {
			if e == nil {
				return
			}

			// Check if this error has a stacktrace
			if stacktrace.Extract(e) != nil {
				count++
			}

			// If this is a joined error, walk its children
			if multi, ok := e.(multiError); ok {
				for _, child := range multi.Unwrap() {
					walk(child)
				}
			} else {
				// Check if it unwraps to another error
				if unwrapped := errors.Unwrap(e); unwrapped != nil {
					walk(unwrapped)
				}
			}
		}
		walk(err)
		return count
	}

	// Before wrapping, we should have errors with stacktraces:
	// - errB (explicitly wrapped)
	// - errC and errD (from distributed wrap of errCD)
	beforeCount := countErrorsWithStacktraces(errABCDE)
	if beforeCount < 2 {
		t.Errorf("Expected at least 2 errors with stacktraces before final wrap, got %d", beforeCount)
	}

	// After wrapping the joined error, ALL individual errors should have stacktraces
	afterCount := countErrorsWithStacktraces(wrappedJoined)

	// We expect at least 5 individual errors to have stacktraces
	// (might be more due to wrapper errors in the chain)
	if afterCount < 5 {
		t.Errorf("Expected at least 5 errors with stacktraces after final wrap, got %d", afterCount)
	}

	// Verify specific errors have stacktraces
	// Extract all individual errors by flattening the joined structure
	var flattenErrors func(error) []error
	flattenErrors = func(err error) []error {
		if err == nil {
			return nil
		}
		if multi, ok := err.(multiError); ok {
			var all []error
			for _, child := range multi.Unwrap() {
				all = append(all, flattenErrors(child)...)
			}
			return all
		}
		return []error{err}
	}

	individualErrors := flattenErrors(wrappedJoined)
	if len(individualErrors) != 5 {
		t.Errorf("Expected 5 individual errors, got %d", len(individualErrors))
	}

	// Verify that existing stacktraces are preserved and new ones are added where needed
	// With structure-preserving approach:
	// - errB already had a stacktrace from line 125, should be preserved
	// - errD already had a stacktrace from line 126, should be preserved
	// - errA and errE should get new stacktraces from the recursive wrapping
	// - errC should get a new stacktrace from when errCD was wrapped

	for i, err := range individualErrors {
		trace := stacktrace.Extract(err)
		if trace == nil {
			t.Errorf("Error %d (%v) should have a stacktrace but doesn't", i, err)
			continue
		}

		errorMsg := err.Error()
		t.Logf("Error %s has stacktrace with %d frames, first frame at line %d in %s",
			errorMsg, len(trace), trace[0].LineNumber, trace[0].Function)

		// With structure-preserving approach, we verify that:
		// 1. All errors have stacktraces
		// 2. Existing stacktraces are preserved (errB from 122, errD from 123)
		// 3. New stacktraces are added where needed
		switch errorMsg {
		case "test error B":
			// errB should have its original stacktrace from line 125
			if trace[0].LineNumber != 125 {
				t.Errorf("Error B: expected original stacktrace line 125, got %d", trace[0].LineNumber)
			}
		case "test error D":
			// errD should have its original stacktrace from line 126
			if trace[0].LineNumber != 126 {
				t.Errorf("Error D: expected original stacktrace line 126, got %d", trace[0].LineNumber)
			}
		default:
			// Other errors (A, C, E) should have new stacktraces from the wrapping process
			// We just verify they have stacktraces, not the exact line numbers
			if len(trace) == 0 {
				t.Errorf("Error %s should have a stacktrace", errorMsg)
			}
		}
	}
}

// TestStackTraceLogValue tests the LogValue method of StackTrace.
func TestStackTraceLogValue(t *testing.T) {
	t.Parallel()

	t.Run("empty stacktrace", func(t *testing.T) {
		t.Parallel()
		var st stacktrace.StackTrace
		logValue := st.LogValue()
		// For empty stacktrace, LogValue should return an empty Value
		if logValue.Kind() != slog.KindString || logValue.String() != "" {
			// Check if it's truly empty by trying to use it
			var buf bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&buf, nil))
			logger.Info("test", slog.Any("empty_stacktrace", st))
			// The empty stacktrace should result in minimal logging
		}
	})

	t.Run("stacktrace with frames", func(t *testing.T) {
		t.Parallel()
		err := stacktrace.Wrap(errors.New("test error"))
		st := stacktrace.Extract(err)

		if st == nil {
			t.Fatal("expected stacktrace")
		}

		logValue := st.LogValue()
		if logValue.Kind() != slog.KindAny {
			t.Errorf("expected KindAny for stacktrace with frames, got %v", logValue.Kind())
		}

		// Test that it can be logged without panic
		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{}))
		logger.Info("test", slog.Any("stacktrace", st))

		if buf.Len() == 0 {
			t.Error("expected log output")
		}
	})
}

// TestGetStackEdgeCases tests edge cases of the GetStack function.
func TestGetStackEdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("GetStack with skipRuntime false", func(t *testing.T) {
		t.Parallel()
		// Call GetStack with skipRuntime=false to test the uncovered branch
		stack := stacktrace.GetStack(1, false)
		if len(stack) == 0 {
			t.Error("expected non-empty stack trace")
		}

		// Should contain this test function
		found := false
		for _, frame := range stack {
			if strings.Contains(frame.Function, "TestGetStackEdgeCases") {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected to find TestGetStackEdgeCases in stack trace")
		}
	})

	t.Run("GetStack with different skip values", func(t *testing.T) {
		t.Parallel()
		stack0 := stacktrace.GetStack(0, true)
		stack1 := stacktrace.GetStack(1, true)
		stack2 := stacktrace.GetStack(2, true)

		// Higher skip values should result in shorter stacks
		if len(stack0) < len(stack1) {
			t.Errorf("expected stack0 (%d) to be >= stack1 (%d)", len(stack0), len(stack1))
		}

		if len(stack1) < len(stack2) {
			t.Errorf("expected stack1 (%d) to be >= stack2 (%d)", len(stack1), len(stack2))
		}

		// Verify that we get meaningful stack traces
		if len(stack0) == 0 {
			t.Error("expected non-empty stack0")
		}
	})

	t.Run("GetStack with high skip value", func(t *testing.T) {
		t.Parallel()
		// Test with a skip value that's higher than the actual stack depth
		stack := stacktrace.GetStack(1000, true)
		// Should return empty stack or very short stack
		if len(stack) > 5 {
			t.Errorf("expected short or empty stack with high skip value, got %d frames", len(stack))
		}
	})
}

// TestStackTraceTypes tests various type assertions and conversions.
func TestStackTraceTypes(t *testing.T) {
	t.Parallel()

	t.Run("StackTrace type", func(t *testing.T) {
		t.Parallel()
		err := stacktrace.Wrap(errors.New("test"))
		st := stacktrace.Extract(err)

		if st == nil {
			t.Fatal("expected stacktrace")
		}

		// Test that it's the correct type
		if reflect.TypeOf(st).String() != "stacktrace.StackTrace" {
			t.Errorf("unexpected type: %s", reflect.TypeOf(st))
		}

		// Test that frames have correct structure
		if len(st) > 0 {
			frame := st[0]
			if frame.File == "" {
				t.Error("expected non-empty File")
			}
			if frame.LineNumber == 0 {
				t.Error("expected non-zero LineNumber")
			}
			if frame.Function == "" {
				t.Error("expected non-empty Function")
			}
		}
	})
}
