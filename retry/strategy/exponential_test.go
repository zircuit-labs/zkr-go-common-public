package strategy_test

import (
	"errors"
	"testing"
	"time"

	"github.com/zircuit-labs/zkr-go-common/retry/strategy"
)

func TestExponential(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		testName             string
		initialDelay         int
		maxDelay             int
		base                 int
		expectedOutputDelays []int
		expectedError        error
	}{
		{
			testName:             "base 2: one second initial, max nine",
			initialDelay:         1,
			maxDelay:             9,
			base:                 2, // Base 2 for 1, 2, 4, 8, 9, 9, 9
			expectedOutputDelays: []int{1, 2, 4, 8, 9, 9, 9},
		},
		{
			testName:             "base 2: two seconds initial, max 21",
			initialDelay:         2,
			maxDelay:             21,
			base:                 2, // Base 2 for 2, 4, 8, 16, 21, 21, 21
			expectedOutputDelays: []int{2, 4, 8, 16, 21, 21, 21},
		},
		{
			testName:             "base 3: one second initial, max 30",
			initialDelay:         1,
			maxDelay:             30,
			base:                 3, // Base 3 for 1, 3, 9, 27, 30, 30, 30
			expectedOutputDelays: []int{1, 3, 9, 27, 30, 30, 30},
		},
		{
			testName:             "base 4: one second initial, max 100",
			initialDelay:         1,
			maxDelay:             100,
			base:                 4, // Base 4 for 1, 4, 16, 64, 100, 100, 100
			expectedOutputDelays: []int{1, 4, 16, 64, 100, 100, 100},
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

			initial := time.Duration(tc.initialDelay) * time.Second
			mx := time.Duration(tc.maxDelay) * time.Second
			factory, err := strategy.NewExponential(initial, mx, strategy.WithBase(tc.base), strategy.WithoutJitter())
			if tc.expectedError != nil {
				if err == nil || !errors.Is(err, tc.expectedError) {
					t.Fatalf("expected error of type: %v, got %v", tc.expectedError, err)
				}
				return
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Ensure the factory produces the same strategy by doing the test multiple times
			for range 3 {
				s := factory()

				// Verify the pattern of the delay values
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
