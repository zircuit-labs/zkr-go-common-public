package strategy_test

import (
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
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			t.Parallel()

			// NewConstant creates a factory
			initial := time.Duration(tc.initialDelay) * time.Second
			mx := time.Duration(tc.maxDelay) * time.Second
			factory := strategy.NewLinear(initial, mx, strategy.WithoutJitter())

			// ensure the factory produces the same strategy by doing the test multiple times
			for i := 0; i < 3; i++ {
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
