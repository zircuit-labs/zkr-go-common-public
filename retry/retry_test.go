package retry_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zircuit-labs/zkr-go-common/retry"
	"github.com/zircuit-labs/zkr-go-common/retry/strategy"
	"github.com/zircuit-labs/zkr-go-common/xerrors"
	"github.com/zircuit-labs/zkr-go-common/xerrors/errclass"
)

var (
	errTest       = fmt.Errorf("this is a test error")
	errPersistent = errclass.WrapAs(errTest, errclass.Persistent)
	errTransient  = errclass.WrapAs(errTest, errclass.Transient)
)

type foo struct {
	count       int
	errs        []error
	shouldPanic bool
}

func (f *foo) bar() error {
	if f.shouldPanic {
		panic("this is a test panic")
	}

	defer func() {
		f.count++
	}()

	if f.count < len(f.errs) {
		return f.errs[f.count]
	}
	return nil
}

func TestRetrySemantics(t *testing.T) {
	t.Parallel()

	noWait, err := strategy.NewConstant(0)
	require.NoError(t, err)

	testCases := []struct {
		testName          string
		cancel            bool
		unknownAs         errclass.Class
		maxAttempts       int
		errs              []error
		shouldPanic       bool
		expectedCause     retry.FailureCause
		expectedAttemptNo int
	}{
		{
			testName:          "immediate success",
			cancel:            false,
			unknownAs:         errclass.Transient,
			maxAttempts:       3,
			errs:              nil,
			shouldPanic:       false,
			expectedCause:     retry.Success,
			expectedAttemptNo: 0,
		},
		{
			testName:          "immediate panic",
			cancel:            false,
			unknownAs:         errclass.Transient,
			maxAttempts:       3,
			errs:              nil,
			shouldPanic:       true,
			expectedCause:     retry.PersistentErrorEncountered,
			expectedAttemptNo: 1,
		},
		{
			testName:          "transient error x2, max 3",
			cancel:            false,
			unknownAs:         errclass.Transient,
			maxAttempts:       3,
			errs:              []error{errTransient, errTransient},
			shouldPanic:       false,
			expectedCause:     retry.Success,
			expectedAttemptNo: 0,
		},
		{
			testName:          "transient error x4, max 3",
			cancel:            false,
			unknownAs:         errclass.Transient,
			maxAttempts:       3,
			errs:              []error{errTransient, errTransient, errTransient, errTransient},
			shouldPanic:       false,
			expectedCause:     retry.MaxAttemptsReached,
			expectedAttemptNo: 4,
		},
		{
			testName:          "transient error x4, max 2",
			cancel:            false,
			unknownAs:         errclass.Transient,
			maxAttempts:       2,
			errs:              []error{errTransient, errTransient, errTransient, errTransient},
			shouldPanic:       false,
			expectedCause:     retry.MaxAttemptsReached,
			expectedAttemptNo: 3,
		},
		{
			testName:          "persistent error x4, max 3",
			cancel:            false,
			unknownAs:         errclass.Transient,
			maxAttempts:       3,
			errs:              []error{errPersistent, errPersistent, errPersistent, errPersistent},
			shouldPanic:       false,
			expectedCause:     retry.PersistentErrorEncountered,
			expectedAttemptNo: 1,
		},
		{
			testName:          "unknown error as transient x4, max 3",
			cancel:            false,
			unknownAs:         errclass.Transient,
			maxAttempts:       3,
			errs:              []error{errTest, errTest, errTest, errTest},
			shouldPanic:       false,
			expectedCause:     retry.MaxAttemptsReached,
			expectedAttemptNo: 4,
		},
		{
			testName:          "unknown error as persistent x4, max 3",
			cancel:            false,
			unknownAs:         errclass.Persistent,
			maxAttempts:       3,
			errs:              []error{errTest, errTest, errTest, errTest},
			shouldPanic:       false,
			expectedCause:     retry.PersistentErrorEncountered,
			expectedAttemptNo: 1,
		},
		{
			testName:          "transient then persistent error",
			cancel:            false,
			unknownAs:         errclass.Transient,
			maxAttempts:       3,
			errs:              []error{errTransient, errPersistent},
			shouldPanic:       false,
			expectedCause:     retry.PersistentErrorEncountered,
			expectedAttemptNo: 2,
		},
		{
			testName:          "context cancelled",
			cancel:            true,
			unknownAs:         errclass.Transient,
			maxAttempts:       3,
			errs:              []error{errTransient, errTransient},
			shouldPanic:       false,
			expectedCause:     retry.ContextDone,
			expectedAttemptNo: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			t.Parallel()

			// set up the retrier
			retrier, err := retry.NewRetrier(
				retry.WithStrategy(noWait),
				retry.WithMaxAttempts(tc.maxAttempts),
				retry.WithUnknownErrorsAs(tc.unknownAs),
			)
			require.NoError(t, err)

			// set up the test function
			f := &foo{
				errs:        tc.errs,
				shouldPanic: tc.shouldPanic,
			}

			// set up context
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			// cancel now if testing for cancellation
			if tc.cancel {
				cancel()
			}

			// execute the retry
			err = retrier.Try(ctx, f.bar)

			// if eventual success, then no error should be returned
			if tc.expectedCause == retry.Success {
				assert.NoError(t, err)
				return
			}

			// verify error type
			switch {
			case tc.shouldPanic:
				require.Equal(t, errclass.Panic.String(), errclass.GetClass(err).String())
			case tc.cancel:
				require.ErrorIs(t, err, context.Canceled)
			default:
				require.ErrorIs(t, err, errTest)
			}

			// verify stats
			stats, ok := xerrors.Extract[retry.Stats](err)
			require.True(t, ok)
			assert.Equal(t, tc.expectedCause, stats.Cause)
			assert.Equal(t, tc.expectedAttemptNo, stats.AttemptNumber)
		})
	}
}
