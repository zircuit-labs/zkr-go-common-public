package strategy_test

import (
	"testing"
	"time"

	"github.com/zircuit-labs/zkr-go-common/retry/strategy"
)

func TestConstant(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		testName             string
		inputDelay           int
		expectedOutputDelays []int
	}{
		{
			testName:             "one second",
			inputDelay:           1,
			expectedOutputDelays: []int{1, 1, 1, 1},
		},
		{
			testName:             "four seconds",
			inputDelay:           4,
			expectedOutputDelays: []int{4, 4, 4, 4, 4},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			t.Parallel()

			// NewConstant creates a factory
			initial := time.Duration(tc.inputDelay) * time.Second
			factory := strategy.NewConstant(initial, strategy.WithoutJitter())

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
