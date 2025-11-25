package polling_test

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zircuit-labs/zkr-go-common/task/polling"
)

var errTest = errors.New("example error")

type testAction struct {
	Err           error
	CallCount     int32
	CleanupCalled bool
}

func (a *testAction) Run(_ context.Context) error {
	a.CallCount++
	return a.Err
}

func (a *testAction) Cleanup() {
	a.CleanupCalled = true
}

func TestPollingTask(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		testName          string
		runAtStart        bool
		actionErr         error
		numTicks          int
		expectedCallCount int32
	}{
		{
			testName:          "five ticks",
			runAtStart:        false,
			actionErr:         nil,
			numTicks:          5,
			expectedCallCount: 5,
		},
		{
			testName:          "three ticks",
			runAtStart:        false,
			actionErr:         nil,
			numTicks:          3,
			expectedCallCount: 3,
		},
		{
			testName:          "five ticks plus start up",
			runAtStart:        true,
			actionErr:         nil,
			numTicks:          5,
			expectedCallCount: 6,
		},
		{
			testName:          "three ticks plus start up",
			runAtStart:        true,
			actionErr:         nil,
			numTicks:          3,
			expectedCallCount: 4,
		},
		{
			testName:          "five ticks with ignored error",
			runAtStart:        false,
			actionErr:         errTest,
			numTicks:          5,
			expectedCallCount: 5,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			t.Parallel()

			synctest.Test(t, func(t *testing.T) {
				action := testAction{Err: tc.actionErr}
				// Use a real ticker with a 100ms interval
				pollInterval := 100 * time.Millisecond
				options := []polling.Option{
					polling.WithInterval(pollInterval),
				}
				if tc.runAtStart {
					options = append(options, polling.WithRunAtStart())
				}

				task := polling.NewTask(tc.testName, &action, options...)
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()

				// start the task (which blocks) and capture any resulting error in a channel
				errCh := make(chan error)
				go func() {
					err := task.Run(ctx)
					errCh <- err
				}()

				// Sleep to let the task run for the expected number of ticks
				// Each tick is 100ms, so we sleep for numTicks * 100ms + a small buffer
				sleepDuration := time.Duration(tc.numTicks)*pollInterval + 50*time.Millisecond
				time.Sleep(sleepDuration)

				// verify that the cleanup action has not yet been called
				assert.False(t, action.CleanupCalled)

				// cancel the context, the task should now stop
				cancel()

				// Wait for the task to complete
				select {
				case err := <-errCh:
					require.NoError(t, err)
				case <-time.After(100 * time.Millisecond):
					t.Fatal("task failed to stop when context was cancelled")
				}

				// check the action was run the expected number of times
				assert.Equal(t, tc.expectedCallCount, action.CallCount)

				// verify that the cleanup action was called
				assert.True(t, action.CleanupCalled)
			})
		})
	}
}

func TestPollingTaskTerminateOnError(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		testName          string
		runAtStart        bool
		expectedCallCount int32
	}{
		{
			testName:          "error on startup",
			runAtStart:        true,
			expectedCallCount: 1,
		},
		{
			testName:          "error on first poll",
			runAtStart:        false,
			expectedCallCount: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			t.Parallel()

			synctest.Test(t, func(t *testing.T) {
				action := testAction{Err: errTest}
				pollInterval := 100 * time.Millisecond
				options := []polling.Option{
					polling.WithInterval(pollInterval),
					polling.WithTerminateOnError(),
				}
				if tc.runAtStart {
					options = append(options, polling.WithRunAtStart())
				}

				task := polling.NewTask(tc.testName, &action, options...)
				ctx := t.Context()

				// start the task (which blocks) and capture any resulting error in a channel
				errCh := make(chan error)
				go func() {
					err := task.Run(ctx)
					errCh <- err
				}()

				// If not running at start, we need to wait for the first tick
				if !tc.runAtStart {
					time.Sleep(pollInterval + 10*time.Millisecond)
				}

				// The task should exit when the polling action is first run
				select {
				case err := <-errCh:
					assert.ErrorIs(t, err, errTest)
				case <-time.After(200 * time.Millisecond):
					t.Fatal("task should have exited with error")
				}

				// check the action was run the expected number of times
				assert.Equal(t, tc.expectedCallCount, action.CallCount)

				// verify that the cleanup action was called
				assert.True(t, action.CleanupCalled)
			})
		})
	}
}
