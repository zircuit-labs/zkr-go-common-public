// Package ossignal provides a Task that listens for signals from the operating system.
package ossignal

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/zircuit-labs/zkr-go-common/log"
)

// DefaultSignals are the os signals that will cause this task to exit.
var DefaultSignals = []os.Signal{
	syscall.SIGINT,
	syscall.SIGTERM,
	syscall.SIGQUIT,
}

// Task is a Task that waits for a termination signal from the OS.
type Task struct {
	sigCh  chan os.Signal
	logger *slog.Logger
}

type options struct {
	signals []os.Signal
	logger  *slog.Logger
}

// Option is an option func for NewTask.
type Option func(options *options)

// WithLogger sets the logger to be used.
func WithLogger(logger *slog.Logger) Option {
	return func(options *options) {
		options.logger = logger
	}
}

// WithSignals overrides the default signals being listened for.
func WithSignals(signals ...os.Signal) Option {
	return func(options *options) {
		options.signals = signals
	}
}

// NewTask creates a new OSSignalTask.
func NewTask(opts ...Option) *Task {
	// Set up default options
	options := options{
		signals: DefaultSignals,
		logger:  log.NewNilLogger(),
	}

	// Apply provided options
	for _, opt := range opts {
		opt(&options)
	}

	task := &Task{
		sigCh:  make(chan os.Signal, 1),
		logger: options.logger,
	}
	signal.Notify(task.sigCh, options.signals...)
	return task
}

// Name returns the name of this task.
func (t *Task) Name() string {
	return "os signal task"
}

// Run executes the task.
func (t *Task) Run(ctx context.Context) error {
	select {
	case sig := <-t.sigCh:
		_ = sig
		// Log this as an error, even though it is expected in many cases
		// The reason being that it could help to detect issues much sooner in cases where
		// the OS has signaled a service to stop in the unexpected case.
		// While this may result in false-positive alerts, that is preferred over missing
		// the potential early warning signs that something else is seriously wrong.
		t.logger.Error("os signal received", slog.String("signal", sig.String()))
	case <-ctx.Done():
	}

	signal.Stop(t.sigCh)
	close(t.sigCh)
	return nil
}
