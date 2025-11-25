package sighup

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"slices"
	"syscall"

	"github.com/zircuit-labs/zkr-go-common/log"
	"github.com/zircuit-labs/zkr-go-common/xerrors/errcontext"
	"github.com/zircuit-labs/zkr-go-common/xerrors/stacktrace"
)

// Action defines the interface for an action to be run.
type Action interface {
	// Run is the polling action.
	// It is expected to be context aware, especially if the action
	// could take any significant time to complete.
	Run(context.Context) error

	// Cleanup is executed when the polling task terminates.
	Cleanup()

	// Name returns the name of the action for logging purposes.
	Name() string
}

// DefaultSignals are the os signals that will cause this task to do something.
var DefaultSignals = []os.Signal{syscall.SIGHUP}

// Task is a Task that waits for a signal from the OS to execute a specific action.
type Task struct {
	sigCh            chan os.Signal
	actions          []Action
	terminateOnError bool
	logger           *slog.Logger
}

type options struct {
	signals          []os.Signal
	actions          []Action
	terminateOnError bool
	logger           *slog.Logger
}

// Option is an option func for NewTask.
type Option func(options *options)

// WithLogger sets the logger to be used.
func WithLogger(logger *slog.Logger) Option {
	return func(options *options) {
		options.logger = logger
	}
}

func WithTerminateOnError(terminate bool) Option {
	return func(options *options) {
		options.terminateOnError = terminate
	}
}

// WithSignals overrides the default signals being listened for.
func WithSignals(signals ...os.Signal) Option {
	return func(options *options) {
		options.signals = signals
	}
}

// WithActions adds actions to the task on creation.
func WithAction(action Action) Option {
	return func(options *options) {
		options.actions = append(options.actions, action)
	}
}

// NewTask creates a new sighup Task
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
		sigCh:            make(chan os.Signal, 1),
		terminateOnError: options.terminateOnError,
		actions:          options.actions,
		logger:           options.logger,
	}
	signal.Notify(task.sigCh, options.signals...)
	return task
}

// Name returns the name of this task.
func (t *Task) Name() string {
	return "sighup task"
}

// AddAction appends an action to the task's action list.
func (t *Task) AddAction(act Action) {
	t.actions = append(t.actions, act)
}

// Run executes the task.
func (t *Task) Run(ctx context.Context) error {
	var errOuter error
loop:
	for {
		select {
		case sig := <-t.sigCh:
			t.logger.Debug("signal received, executing actions", slog.String("signal", sig.String()))
			for _, act := range t.actions {
				t.logger.Debug("executing action", slog.String("action", act.Name()))
				if err := act.Run(ctx); err != nil {
					t.logger.Error("error executing action", log.ErrAttr(err), slog.String("action", act.Name()))
					if t.terminateOnError {
						t.logger.Debug("terminating task due to error in action", slog.String("action", act.Name()))
						errOuter = errcontext.Add(stacktrace.Wrap(err), slog.String("action", act.Name()))
						break loop
					}
				}
			}
		case <-ctx.Done():
			break loop
		}
	}

	// Stop listening for signals and clean up
	signal.Stop(t.sigCh)
	close(t.sigCh)

	// Cleanup actions in reverse order of registration
	for _, act := range slices.Backward(t.actions) {
		act.Cleanup()
	}

	t.logger.Debug("task complete")
	return errOuter
}
