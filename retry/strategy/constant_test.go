package strategy_test

import (
	"errors"
	"testing"
	"time"

	"github.com/zircuit-labs/zkr-go-common/retry/strategy"
)

func TestConstant(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		testName             string
		initialDelay         int
		expectedOutputDelays []int
		expectedError        error
	}{
		{
			testName:             "one second",
			initialDelay:         1,
			expectedOutputDelays: []int{1, 1, 1, 1, 1},
		},
		{
			testName:             "four seconds",
			initialDelay:         4,
			expectedOutputDelays: []int{4, 4, 4, 4, 4},
		},
		{
			testName:             "zero seconds",
			initialDelay:         0,
			expectedOutputDelays: []int{0, 0, 0, 0, 0},
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
			factory, err := strategy.NewConstant(initial, strategy.WithoutJitter())
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
