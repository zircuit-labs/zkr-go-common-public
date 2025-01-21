// Package polling provides a Task that periodically executes a function.
package polling

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/zircuit-labs/zkr-go-common/log"
)

const (
	defaultPollInterval = time.Minute
)

// Ticker mimics time.Ticker but can be replaced for testing purposes
type Ticker interface {
	Stop()
	Chan() <-chan time.Time
	Tick() // for use only in testing
}

type tickerWrapper struct {
	t *time.Ticker
}

func (t *tickerWrapper) Tick() {
	// not supported for a real ticker
}

func (t *tickerWrapper) Stop() {
	t.t.Stop()
}

func (t *tickerWrapper) Chan() <-chan time.Time {
	return t.t.C
}

func defaultTicker(d time.Duration) *tickerWrapper {
	return &tickerWrapper{t: time.NewTicker(d)}
}

// Action defines the interface for an action to be periodically run.
type Action interface {
	// Run is the polling action.
	// It is expected to be context aware, especially if the action
	// could take any significant time to complete.
	Run(context.Context) error

	// Cleanup is executed when the polling task terminates.
	Cleanup()
}

// Task periodically runs the PollingAction.
type Task struct {
	name   string
	action Action
	opts   options
}

type options struct {
	testTicker       Ticker
	pollingInterval  time.Duration
	runAtStart       bool
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

// WithInterval sets the polling action interval.
func WithInterval(d time.Duration) Option {
	return func(options *options) {
		options.pollingInterval = d
	}
}

// WithRunAtStart ensures the polling action is executed immediately
// when the task is run (rather than waiting for the polling interval).
func WithRunAtStart() Option {
	return func(options *options) {
		options.runAtStart = true
	}
}

// WithTerminateOnError causes the task to exit with an error if the
// polling action returns an error (by default it just logs a warning).
func WithTerminateOnError() Option {
	return func(options *options) {
		options.terminateOnError = true
	}
}

// WithTestTicker replaces the underlying ticker mechanism.
// This should only be used for testing purposes.
func WithTestTicker(t Ticker) Option {
	return func(options *options) {
		options.testTicker = t
	}
}

// NewTask creates a new PollingTask.
func NewTask(name string, action Action, opts ...Option) *Task {
	// Set up default options
	options := options{
		testTicker:       nil,
		pollingInterval:  defaultPollInterval,
		runAtStart:       false,
		terminateOnError: false,
		logger:           log.NewNilLogger(),
	}

	// Apply provided options
	for _, opt := range opts {
		opt(&options)
	}

	task := &Task{
		name:   name,
		action: action,
		opts:   options,
	}
	return task
}

// Name returns the name of this task.
func (t *Task) Name() string {
	return t.name
}

// Run executes the task.
func (t *Task) Run(ctx context.Context) error {
	defer t.action.Cleanup()

	var ticker Ticker
	if t.opts.testTicker != nil {
		ticker = t.opts.testTicker
	} else {
		ticker = defaultTicker(t.opts.pollingInterval)
	}
	defer ticker.Stop()

	if t.opts.runAtStart {
		if err := t.executeAction(ctx); err != nil {
			return err
		}
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.Chan():
			if err := t.executeAction(ctx); err != nil {
				return err
			}
		}
	}
}

func (t *Task) executeAction(ctx context.Context) error {
	if err := t.action.Run(ctx); err != nil {
		if errors.Is(err, context.Canceled) {
			return nil
		} else if t.opts.terminateOnError {
			return err
		}
		// Don't return the error so that the task will not terminate,
		// however still log this as an error for appropriate visibility.
		t.opts.logger.Error("polling action failed", log.ErrAttr(err), slog.String("task", t.Name()))
	}
	return nil
}
