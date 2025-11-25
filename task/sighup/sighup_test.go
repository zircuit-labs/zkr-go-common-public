package sighup_test

import (
	"context"
	"errors"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zircuit-labs/zkr-go-common/log"
	"github.com/zircuit-labs/zkr-go-common/task/sighup"
)

const (
	waitTime = time.Millisecond * 50
)

var (
	cleanCounter atomic.Uint32
	errTest      = errors.New("example error")
)

type testAction struct {
	Err           error
	ErrAfter      int
	CallCount     int
	CleanupCalled uint32
	name          string
}

func (a *testAction) Run(_ context.Context) error {
	a.CallCount++
	if a.ErrAfter > 0 && a.CallCount >= a.ErrAfter {
		return a.Err
	}
	return nil
}

func (a *testAction) Cleanup() {
	c := cleanCounter.Add(1)
	a.CleanupCalled = c
}

func (a *testAction) Name() string {
	return a.name
}

func TestSighupTask(t *testing.T) { //nolint:paralleltest // cannot test in parallel as it uses syscall.Kill
	testCases := []struct {
		testName          string
		actions           []*testAction
		numSigs           int
		expectedCallCount int
		expectedError     error
		terminateOnError  bool
	}{
		{
			testName:          "five signals",
			actions:           []*testAction{{name: "a0"}},
			numSigs:           5,
			expectedCallCount: 5,
			expectedError:     nil,
			terminateOnError:  false,
		},
		{
			testName:          "five signals, times two",
			actions:           []*testAction{{name: "a1"}, {name: "a2"}},
			numSigs:           5,
			expectedCallCount: 5,
			expectedError:     nil,
			terminateOnError:  false,
		},
		{
			testName:          "five signals, with error on one action",
			actions:           []*testAction{{name: "a1"}, {name: "a2", Err: errTest, ErrAfter: 3}},
			numSigs:           5,
			expectedCallCount: 5,
			expectedError:     nil,
			terminateOnError:  false,
		},
		{
			testName:          "three signals, with error on one action and terminate on error",
			actions:           []*testAction{{name: "a1"}, {name: "a2", Err: errTest, ErrAfter: 3}},
			numSigs:           3,
			expectedCallCount: 3,
			expectedError:     errTest,
			terminateOnError:  true,
		},
	}

	for _, tc := range testCases { //nolint:paralleltest // cannot test in parallel as it uses syscall.Kill
		t.Run(tc.testName, func(t *testing.T) {
			// Reset cleanup counter for test isolation
			cleanCounter.Store(0)

			// Note: Cannot use synctest.Test here because signal.Notify uses OS signals
			// which are not compatible with synctest's deterministic testing model
			opts := []sighup.Option{
				sighup.WithLogger(log.NewTestLogger(t)),
			}
			if tc.terminateOnError {
				opts = append(opts, sighup.WithTerminateOnError(tc.terminateOnError))
			}
			for _, action := range tc.actions {
				opts = append(opts, sighup.WithAction(action))
			}
			task := sighup.NewTask(opts...)

			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			// start the task (which blocks) and capture any resulting error in a channel
			errCh := make(chan error)
			defer close(errCh)
			go func() {
				err := task.Run(ctx)
				errCh <- err
			}()

			// send the expected signal several times
			for range tc.numSigs {
				err := syscall.Kill(syscall.Getpid(), syscall.SIGHUP)
				require.NoError(t, err)
				time.Sleep(waitTime) // give the task some time to process the signal
			}

			// use a real clock for a small delay to ensure
			// context switch allowing task to actually run
			timer := time.NewTimer(waitTime)
			defer timer.Stop()

			// waiting around for a while, the task should not exit
			// since in this test the task doesn't terminate on error
			select {
			case err := <-errCh:
				if tc.expectedError != nil {
					assert.ErrorIs(t, err, tc.expectedError, "expected error did not match")
					// verify that the cleanup actions were called in the expected order
					for i, action := range tc.actions {
						assert.Greater(t, action.CleanupCalled, uint32(0), "cleanup was not called for action '%s'", action.Name())
						if i > 0 {
							assert.Less(t, action.CleanupCalled, tc.actions[i-1].CleanupCalled, "cleanup called for action '%s' should be less than previous action %s", action.Name(), tc.actions[i-1].Name())
						}
					}
					return // end of the test
				}
				require.NoError(t, err)
			case <-timer.C:
			}

			// verify that the cleanup action has not yet been called for any action
			for _, action := range tc.actions {
				assert.Equal(t, uint32(0), action.CleanupCalled)
			}

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

			// check the actions were run the expected number of times
			for _, action := range tc.actions {
				assert.Equal(t, tc.expectedCallCount, action.CallCount, "action '%s' was not called the expected number of times", action.Name())
			}

			// verify that the cleanup actions were called in the expected order
			for i, action := range tc.actions {
				assert.Greater(t, action.CleanupCalled, uint32(0), "cleanup was not called for action '%s'", action.Name())
				if i > 0 {
					assert.Less(t, action.CleanupCalled, tc.actions[i-1].CleanupCalled, "cleanup called for action '%s' should be less than previous action %s", action.Name(), tc.actions[i-1].Name())
				}
			}
		})
	}
}
