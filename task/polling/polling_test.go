package polling_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zircuit-labs/zkr-go-common/task/polling"
)

const (
	waitTime = time.Millisecond * 50
)

var errTest = errors.New("example error")

type fakeTicker struct {
	Ch chan time.Time
}

func (t *fakeTicker) Stop() {
	close(t.Ch)
}

func (t *fakeTicker) Chan() <-chan time.Time {
	return t.Ch
}

func (t *fakeTicker) Tick() {
	t.Ch <- time.Now()
}

func newFakeTicker() polling.Ticker {
	return &fakeTicker{Ch: make(chan time.Time)}
}

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

			ticker := newFakeTicker()
			action := testAction{Err: tc.actionErr}
			options := []polling.Option{
				polling.WithTestTicker(ticker),
			}
			if tc.runAtStart {
				options = append(options, polling.WithRunAtStart())
			}

			task := polling.NewTask(tc.testName, &action, options...)
			ctx, cancel := context.WithCancel(context.Background())

			// start the task (which blocks) and capture any resulting error in a channel
			errCh := make(chan error)
			defer close(errCh)
			go func() {
				err := task.Run(ctx)
				errCh <- err
			}()

			// cause the fake ticker to "fire" the required number of times
			for i := 0; i < tc.numTicks; i++ {
				ticker.Tick()
			}

			// use a real clock for a small delay to ensure
			// context switch allowing task to actually run
			timer := time.NewTimer(waitTime)
			defer timer.Stop()

			// waiting around for a while, the task should not exit
			// since in this test the task doesn't terminate on error
			select {
			case err := <-errCh:
				cancel()
				require.NoError(t, err)
			case <-timer.C:
				break
			}

			// verify that the cleanup action has not yet been called
			assert.False(t, action.CleanupCalled)

			// cancel the context, the task should now stop
			cancel()

			// verify that the task stops (wait a max amount of time for this)
			timer.Reset(waitTime)
			select {
			case err := <-errCh:
				require.NoError(t, err)
			case <-timer.C:
				t.Fatal("task failed to stop when context was cancelled")
			}

			// check the action was run the expected number of times
			assert.Equal(t, tc.expectedCallCount, action.CallCount)

			// verify that the cleanup action was called
			assert.True(t, action.CleanupCalled)
		})
	}
}

func TestPollingTaskTerminateOnError(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		testName          string
		runAtStart        bool
		numTicks          int
		expectedCallCount int32
	}{
		{
			testName:          "error on startup",
			runAtStart:        true,
			numTicks:          2,
			expectedCallCount: 1,
		},
		{
			testName:          "error on first poll",
			runAtStart:        false,
			numTicks:          2,
			expectedCallCount: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			t.Parallel()

			ticker := newFakeTicker()
			action := testAction{Err: errTest}
			options := []polling.Option{
				polling.WithTestTicker(ticker),
				polling.WithTerminateOnError(),
			}
			if tc.runAtStart {
				options = append(options, polling.WithRunAtStart())
			}

			task := polling.NewTask(tc.testName, &action, options...)
			ctx := context.Background()

			// start the task (which blocks) and capture any resulting error in a channel
			errCh := make(chan error)
			defer close(errCh)
			go func() {
				err := task.Run(ctx)
				errCh <- err
			}()

			// use a real clock for a small delay to ensure
			// context switch allowing task to actually run
			timer := time.NewTimer(waitTime)
			defer timer.Stop()

			// advance the fake ticker only if we don't run at start
			if !tc.runAtStart {
				ticker.Tick()
			}

			// waiting around for a while, the task should exit when
			// the polling action is first run
			select {
			case err := <-errCh:
				assert.ErrorIs(t, err, errTest)
			case <-timer.C:
				t.Log("task should have exited with error")
				t.Fail()
			}

			// check the action was run the expected number of times
			assert.Equal(t, tc.expectedCallCount, action.CallCount)

			// verify that the cleanup action was called
			assert.True(t, action.CleanupCalled)
		})
	}
}
