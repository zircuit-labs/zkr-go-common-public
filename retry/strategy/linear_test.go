package strategy_test

import (
	"errors"
	"testing"
	"time"

	"github.com/zircuit-labs/zkr-go-common/retry/strategy"
)

func TestLinear(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		testName             string
		initialDelay         int
		maxDelay             int
		expectedOutputDelays []int
		expectedError        error
	}{
		{
			testName:             "one second, max four",
			initialDelay:         1,
			maxDelay:             4,
			expectedOutputDelays: []int{1, 2, 3, 4, 4, 4, 4},
		},
		{
			testName:             "two seconds, max eleven",
			initialDelay:         2,
			maxDelay:             11,
			expectedOutputDelays: []int{2, 4, 6, 8, 10, 11, 11, 11, 11},
		},
		{
			testName:      "zero seconds",
			initialDelay:  0,
			expectedError: strategy.ErrInvalidInitialDelay,
		},
		{
			testName:      "negative seconds",
			initialDelay:  -100,
			expectedError: strategy.ErrInvalidInitialDelay,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			t.Parallel()

			// NewConstant creates a factory
			initial := time.Duration(tc.initialDelay) * time.Second
			mx := time.Duration(tc.maxDelay) * time.Second
			factory, err := strategy.NewLinear(initial, mx, strategy.WithoutJitter())
			if tc.expectedError != nil {
				if err == nil || !errors.Is(err, tc.expectedError) {
					t.Fatalf("expected error of type: %v, got %v", tc.expectedError, err)
				}
				return
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// ensure the factory produces the same strategy by doing the test multiple times
			for range 3 {
				s := factory()

				// verify the pattern of the delay values
				for _, expected := range tc.expectedOutputDelays {
					actual := int(s.NextDelay().Seconds())
					if actual != expected {
						t.Errorf("unexpected output: want: %v got %v", expected, actual)
					}
				}
			}
		})
	}
}
