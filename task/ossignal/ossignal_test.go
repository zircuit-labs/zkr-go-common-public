package ossignal_test

import (
	"context"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zircuit-labs/zkr-go-common/task/ossignal"
)

const (
	waitTime = time.Millisecond * 50
)

func TestSignal(t *testing.T) {
	t.Parallel()

	// use a signal that won't cause issues with testing
	task := ossignal.NewTask(ossignal.WithSignals(syscall.SIGCONT))
	assert.Equal(t, "os signal task", task.Name())

	// start the task (which blocks) and capture any resulting error in a channel
	errCh := make(chan error)
	go func() {
		ctx := context.Background()
		err := task.Run(ctx)
		errCh <- err
	}()

	timer := time.NewTimer(waitTime)
	defer timer.Stop()

	// waiting around for a while, the task should not exit
	select {
	case err := <-errCh:
		require.NoError(t, err)
	case <-timer.C:
		break
	}

	// send the expected signal, the task should now stop
	err := syscall.Kill(syscall.Getpid(), syscall.SIGCONT)
	require.NoError(t, err)

	// verify that the task stops (wait a max amount of time for this)
	timer.Reset(waitTime)
	select {
	case err := <-errCh:
		require.NoError(t, err)
	case <-timer.C:
		t.Fatal("os signal task failed to exit after being signalled")
	}
}

func TestContext(t *testing.T) {
	t.Parallel()

	// use a different signal from the other test
	task := ossignal.NewTask(ossignal.WithSignals(syscall.SIGIO))
	assert.Equal(t, "os signal task", task.Name())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// start the task (which blocks) and capture any resulting error in a channel
	errCh := make(chan error)
	defer close(errCh)
	go func() {
		err := task.Run(ctx)
		errCh <- err
	}()

	timer := time.NewTimer(waitTime)
	defer timer.Stop()

	// waiting around for a while, the task should not exit
	select {
	case err := <-errCh:
		require.NoError(t, err)
	case <-timer.C:
		break
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
}
