package calm_test

import (
	"strings"
	"testing"

	"github.com/zircuit-labs/zkr-go-common/calm"
	"github.com/zircuit-labs/zkr-go-common/xerrors/errclass"
	"github.com/zircuit-labs/zkr-go-common/xerrors/stacktrace"
)

func a() error {
	return b()
}

func b() error {
	return c()
}

func c() error {
	panic("this is a test panic")
}

// TestStackTrace checks that a panic is caught correctly.
// WARNING: This test is extremely fragile if line numbers change.
func TestUnpanic(t *testing.T) {
	t.Parallel()

	err := calm.Unpanic(a)
	if err == nil {
		t.Errorf("expected error: got %v", err)
	}

	class := errclass.GetClass(err)
	if class != errclass.Panic {
		t.Errorf("unexpected error class: want: %s got %s", errclass.Panic, class)
	}

	trace := stacktrace.Extract(err)
	if trace == nil {
		t.Errorf("expected stack trace: got %v", trace)
	}

	if len(trace) != 5 {
		t.Errorf("unexpected stack trace len: want: %d got %d.\n-----\n%v\n-----\n", 4, len(trace), trace)
	}

	expected := []stacktrace.Frame{
		{
			File:       "calm/calm_test.go",
			LineNumber: 21,
			Function:   "calm_test.c",
		},
		{
			File:       "calm/calm_test.go",
			LineNumber: 17,
			Function:   "calm_test.b",
		},
		{
			File:       "calm/calm_test.go",
			LineNumber: 13,
			Function:   "calm_test.a",
		},
		{
			File:       "calm/calm.go",
			LineNumber: 30,
			Function:   "calm.Unpanic",
		},
		{
			File:       "calm/calm_test.go",
			LineNumber: 29,
			Function:   "calm_test.TestUnpanic",
		},
	}

	for i, frame := range trace {
		if !strings.HasSuffix(frame.File, expected[i].File) {
			t.Errorf("unexpected file name suffix: want: %s got %s", expected[i].File, frame.File)
		}
		if !strings.HasSuffix(frame.File, expected[i].File) {
			t.Errorf("unexpected function name suffix: want: %s got %s", expected[i].Function, frame.Function)
		}
		if frame.LineNumber != expected[i].LineNumber {
			t.Errorf("unexpected line number: want: %d got %d", expected[i].LineNumber, frame.LineNumber)
		}
	}
}
