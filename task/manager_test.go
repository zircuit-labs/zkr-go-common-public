package task_test

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/zircuit-labs/zkr-go-common/log"
	"github.com/zircuit-labs/zkr-go-common/task"
)

var errTest = errors.New("test error")

type TestTask struct {
	errChan chan error
	name    string
	err     error
}

func (t TestTask) Run(ctx context.Context) error {
	defer close(t.errChan)

	select {
	case err := <-t.errChan:
		return err
	case <-ctx.Done():
		return t.err
	}
}

func (t TestTask) Error(err error) {
	t.errChan <- err
}

func NewTestTask(name string, err error) *TestTask {
	return &TestTask{
		errChan: make(chan error),
		name:    name,
		err:     err,
	}
}

func (t TestTask) Name() string {
	return t.name
}

func TestTaskManagerStop(t *testing.T) {
	t.Parallel()

	logger := log.NewTestLogger(t)
	tm := task.NewManager(task.WithLogger(logger))

	task1 := NewTestTask("task1", nil)
	task2 := NewTestTask("task2", nil)

	cleanupCheck := make([]int, 0, 2)
	tm.Cleanup(func() { cleanupCheck = append(cleanupCheck, 1) })
	tm.Cleanup(func() { cleanupCheck = append(cleanupCheck, 2) })

	tm.Run(task1, task2)

	err := tm.Stop()
	assert.NoError(t, err)
	assert.Equal(t, []int{2, 1}, cleanupCheck)
}

func TestTaskManagerStopError(t *testing.T) {
	t.Parallel()

	logger := log.NewTestLogger(t)
	tm := task.NewManager(task.WithLogger(logger))

	task1 := NewTestTask("task1", errTest)
	task2 := NewTestTask("task2", nil)

	cleanupCheck := make([]int, 0, 2)
	tm.Cleanup(func() { cleanupCheck = append(cleanupCheck, 1) })
	tm.Cleanup(func() { cleanupCheck = append(cleanupCheck, 2) })

	tm.Run(task1, task2)

	err := tm.Stop()
	assert.Error(t, err)
	assert.ErrorIs(t, err, errTest)
	assert.Equal(t, []int{2, 1}, cleanupCheck)
}

func TestTaskManagerRunError(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		logger := log.NewTestLogger(t)
		tm := task.NewManager(task.WithLogger(logger))

		task1 := NewTestTask("task1", nil)
		task2 := NewTestTask("task2", nil)

		cleanupCheck := make([]int, 0, 2)
		tm.Cleanup(func() { cleanupCheck = append(cleanupCheck, 1) })
		tm.Cleanup(func() { cleanupCheck = append(cleanupCheck, 2) })

		tm.Run(task1, task2)

		// task 2 encounters an error after it has started running
		go func() {
			time.Sleep(time.Millisecond * 100)
			task2.Error(errTest)
		}()

		err := tm.Wait()
		assert.Error(t, err)
		assert.ErrorIs(t, err, errTest)
		assert.Equal(t, []int{2, 1}, cleanupCheck)
	})
}

func TestTaskManagerRun(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		logger := log.NewTestLogger(t)
		tm := task.NewManager(task.WithLogger(logger))

		task1 := NewTestTask("task1", nil)
		task2 := NewTestTask("task2", nil)

		cleanupCheck := make([]int, 0, 2)
		tm.Cleanup(func() { cleanupCheck = append(cleanupCheck, 1) })
		tm.Cleanup(func() { cleanupCheck = append(cleanupCheck, 2) })

		tm.Run(task1, task2)

		// task 2 stops without error after it has started running
		go func() {
			time.Sleep(time.Millisecond * 100)
			task2.Error(nil)
		}()

		err := tm.Wait()
		assert.NoError(t, err)
		assert.Equal(t, []int{2, 1}, cleanupCheck)
	})
}

func TestTaskManagerRunTerminable(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		logger := log.NewTestLogger(t)
		tm := task.NewManager(task.WithLogger(logger))

		task1 := NewTestTask("task1", nil)
		task2 := NewTestTask("task2", nil)

		cleanupCheck := make([]int, 0, 2)
		tm.Cleanup(func() { cleanupCheck = append(cleanupCheck, 1) })
		tm.Cleanup(func() { cleanupCheck = append(cleanupCheck, 2) })

		tm.Run(task1)
		tm.RunTerminable(task2)

		// task 2 stops without error after it has started running
		go func() {
			time.Sleep(time.Millisecond * 100)
			task2.Error(nil)
		}()

		// task 1 stops with error even later
		go func() {
			time.Sleep(time.Millisecond * 200)
			task1.Error(errTest)
		}()

		// We should expect the error from task1 to stop the manager and
		// then propagate up to the caller.
		// Task2 should stop itself, but not the manager - allowing task1
		// to continue running until it finishes or errors out.

		err := tm.Wait()
		assert.Error(t, err)
		assert.ErrorIs(t, err, errTest)
		assert.Equal(t, []int{2, 1}, cleanupCheck)
	})
}
