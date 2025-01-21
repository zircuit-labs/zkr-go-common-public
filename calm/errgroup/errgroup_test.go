package errgroup_test

import (
	"fmt"
	"testing"

	"github.com/zircuit-labs/zkr-go-common/calm/errgroup"
	"github.com/zircuit-labs/zkr-go-common/xerrors/errclass"
)

var errTest = fmt.Errorf("this is a test error")

func a() error {
	return nil
}

func b() error {
	return errTest
}

func c() error {
	panic("this is a test panic")
}

type errFunc func() error

func TestErrGroup(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		testName      string
		funcs         []errFunc
		expectedClass errclass.Class
	}{
		{
			testName:      "funcs return nil",
			funcs:         []errFunc{a, a, a},
			expectedClass: errclass.Nil,
		},
		{
			testName:      "one func has error",
			funcs:         []errFunc{a, a, b},
			expectedClass: errclass.Unknown,
		},
		{
			testName:      "one func has panic",
			funcs:         []errFunc{a, a, c},
			expectedClass: errclass.Panic,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			t.Parallel()

			g := errgroup.New()
			for _, f := range tc.funcs {
				g.Go(f)
			}

			err := g.Wait()
			class := errclass.GetClass(err)
			if class != tc.expectedClass {
				t.Errorf("unexpected error class: want: %s got %s", tc.expectedClass, class)
			}
		})
	}
}
