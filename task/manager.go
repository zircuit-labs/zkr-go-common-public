package task

import (
	"context"
	"log/slog"
	"slices"

	"github.com/zircuit-labs/zkr-go-common/calm/errgroup"
	"github.com/zircuit-labs/zkr-go-common/log"
)

// Manager manages a group of tasks that
// should all stop when any one of them stops.
type Manager struct {
	ctx     context.Context
	cancel  context.CancelFunc
	group   *errgroup.Group
	logger  *slog.Logger
	cleanup []func()
}

type options struct {
	logger *slog.Logger
}

// Option is an option func for NewManager.
type Option func(options *options)

// WithLogger sets the logger to be used.
func WithLogger(logger *slog.Logger) Option {
	return func(options *options) {
		options.logger = logger
	}
}

// NewManager creates a Manager.
func NewManager(opts ...Option) *Manager {
	// Set up default options
	options := options{
		logger: log.NewNilLogger(),
	}

	// Apply provided options
	for _, opt := range opts {
		opt(&options)
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{
		ctx:    ctx,
		cancel: cancel,
		group:  errgroup.New(),
		logger: options.logger,
	}
}

// Run immediately starts all of the given tasks.
func (tm *Manager) Run(tasks ...Task) {
	for _, task := range tasks {
		t := task // local for closure

		// Note: calm/errgroup will recover
		// a panic as an error, so we don't need to
		tm.group.Go(tm.runTask(t, true))
	}
}

// Run immediately starts all of the given tasks. These tasks are expected to
// to terminate without error, while others continue running.
func (tm *Manager) RunTerminable(tasks ...Task) {
	for _, task := range tasks {
		t := task // local for closure

		// Note: calm/errgroup will recover
		// a panic as an error, so we don't need to
		tm.group.Go(tm.runTask(t, false))
	}
}

// Cleanup registers a function that runs after all tasks are stopped.
// Similar to defer, cleanup functions are executed in the reverse order
// in which they were registered.
func (tm *Manager) Cleanup(f func()) {
	tm.cleanup = append(tm.cleanup, f)
}

// Wait blocks until all tasks are complete, then executes all registered
// cleanup functions.
// Wait returns the first encountered error.
func (tm *Manager) Wait() error {
	err := tm.group.Wait()
	for _, f := range slices.Backward(tm.cleanup) {
		f()
	}
	return err
}

// Stop cancels the context immediately and waits for all running tasks to complete.
func (tm *Manager) Stop() error {
	tm.cancel()
	return tm.Wait()
}

func (tm *Manager) runTask(t Task, terminateAll bool) func() error {
	return func() error {
		tm.logger.Info("task starting", slog.String("task", t.Name()))
		if err := t.Run(tm.ctx); err != nil {
			tm.logger.Error("task failed", slog.String("task", t.Name()), log.ErrAttr(err))
			tm.cancel()
			return err
		}

		if terminateAll {
			// when the task completes, regardless of why, cancel the context
			// so that other tasks know they should also stop
			defer tm.cancel()
		}

		tm.logger.Info("task stopped", slog.String("task", t.Name()))
		return nil
	}
}

// Context returns the context used for managing all tasks.
func (tm *Manager) Context() context.Context {
	return tm.ctx
}
