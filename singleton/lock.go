package singleton

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/zircuit-labs/zkr-go-common/log"
	"github.com/zircuit-labs/zkr-go-common/xerrors/errcontext"
	"github.com/zircuit-labs/zkr-go-common/xerrors/stacktrace"
)

const (
	BucketName                  = "singleton_locks"
	BucketTTL                   = time.Minute * 15 // lock validity must not exceed this time
	UnlockTimeout               = time.Millisecond * 100
	defaultLockValidityInterval = (time.Minute * 5) + (time.Second * 10)
	defaultLockRefreshInterval  = time.Minute // refresh must be less than validity
)

var (
	ErrInvalidOption = errors.New("invalid option provided")
	ErrLockLost      = errors.New("lock was unexpectedly lost")
)

type LockFactory[T any] struct {
	kv         jetstream.KeyValue
	instanceID string
	opts       options
}

type options struct {
	lockValidityInterval time.Duration
	lockRefreshInterval  time.Duration
	logger               *slog.Logger
}

type Option func(options *options)

func WithLockValidityInterval(interval time.Duration) Option {
	return func(options *options) {
		options.lockValidityInterval = interval
	}
}

func WithLockRefreshInterval(interval time.Duration) Option {
	return func(options *options) {
		options.lockRefreshInterval = interval
	}
}

func WithLogger(logger *slog.Logger) Option {
	return func(options *options) {
		options.logger = logger
	}
}

// NewLockFactory creates a new lock factory.
func NewLockFactory[T any](nc *nats.Conn, instanceID string, opts ...Option) (*LockFactory[T], error) {
	options := options{
		lockValidityInterval: defaultLockValidityInterval,
		lockRefreshInterval:  defaultLockRefreshInterval,
		logger:               log.NewNilLogger(),
	}
	for _, opt := range opts {
		opt(&options)
	}
	if options.lockValidityInterval < options.lockRefreshInterval {
		return nil, stacktrace.Wrap(ErrInvalidOption)
	}
	if BucketTTL < options.lockValidityInterval {
		return nil, stacktrace.Wrap(ErrInvalidOption)
	}

	options.logger = options.logger.With(
		slog.String("instance", instanceID),
	)

	js, err := jetstream.New(nc)
	if err != nil {
		return nil, stacktrace.Wrap(err)
	}

	kv, err := js.CreateOrUpdateKeyValue(context.Background(), jetstream.KeyValueConfig{
		Bucket:      BucketName,
		TTL:         BucketTTL,
		Compression: true,
		MaxBytes:    500 << 20, // 500MB
	})
	if err != nil {
		return nil, stacktrace.Wrap(err)
	}

	return &LockFactory[T]{
		kv:         kv,
		instanceID: instanceID,
		opts:       options,
	}, nil
}

// TryCreateLock attempts to create a new lock, but does not block if the lock is already held.
// If the lock is already held, the current lock content is returned instead.
func (f *LockFactory[T]) TryCreateLock(ctx context.Context, key string, content T) (*Lock[T], *T, error) {
	lock := &Lock[T]{
		kv:         f.kv,
		key:        key,
		content:    content,
		instanceID: f.instanceID,
		opts:       f.opts,
	}
	lock.LockCtx, lock.cancel = context.WithCancelCause(context.Background())
	lock.opts.logger = lock.opts.logger.With(slog.String("key", key))

	// Try to acquire the lock, or return the current lock content.
	for {
		// Marshal the lock content every time we try to acquire
		// the lock so the expiry time is updated.
		v, err := lock.Marshal(content)
		if err != nil {
			return nil, nil, stacktrace.Wrap(err)
		}

		// Attempt to acquire the lock.
		rev, err := f.kv.Create(ctx, key, v)
		switch {
		case errors.Is(err, jetstream.ErrKeyExists):
			// The lock is held by someone else.
		case err != nil:
			// Unexpected error.
			return nil, nil, stacktrace.Wrap(err)
		default:
			// Lock acquired.
			lock.opts.logger.Info("lock acquired", slog.Uint64("rev", rev))
			lock.rev = rev
			lock.locked = true
			lock.wg.Go(lock.continuallyRefresh)
			return lock, nil, nil
		}

		// Otherwise, get the current lock content.
		kve, err := f.kv.Get(ctx, key)
		switch {
		case errors.Is(err, jetstream.ErrKeyNotFound):
			// The lock was released. Try again.
			continue
		case err != nil:
			// Unexpected error.
			return nil, nil, stacktrace.Wrap(err)
		default:
			// Parse the current value.
			var value lockValue[T]
			if err := json.Unmarshal(kve.Value(), &value); err != nil {
				// The value is garbage: delete it, ignoring any errors, and try again.
				lock.opts.logger.Warn("detected garbage lock contents - deleting key", log.ErrAttr(err))
				_ = f.kv.Delete(ctx, key, jetstream.LastRevision(kve.Revision()))
				continue
			}
			return nil, &value.Content, nil
		}
	}
}

// CreateLock creates a new lock and blocks until the lock has been acquired.
func (f *LockFactory[T]) CreateLock(ctx context.Context, key string, content T) (*Lock[T], error) {
	lock := &Lock[T]{
		kv:         f.kv,
		key:        key,
		content:    content,
		instanceID: f.instanceID,
		opts:       f.opts,
	}
	lock.LockCtx, lock.cancel = context.WithCancelCause(context.Background())
	lock.opts.logger = lock.opts.logger.With(slog.String("key", key))

	for {
		// Marshal the lock content every time we try to acquire
		// the lock so the expiry time is updated.
		v, err := lock.Marshal(content)
		if err != nil {
			return nil, stacktrace.Wrap(err)
		}

		// Attempt to acquire the lock.
		rev, err := f.kv.Create(ctx, key, v)
		switch {
		case errors.Is(err, jetstream.ErrKeyExists):
			// The lock is held by someone else.
		case err != nil:
			// Unexpected error.
			return nil, stacktrace.Wrap(err)
		default:
			// Lock acquired.
			lock.opts.logger.Info("lock acquired", slog.Uint64("rev", rev))
			lock.rev = rev
			lock.locked = true
			lock.wg.Go(lock.continuallyRefresh)
			return lock, nil
		}

		// Otherwise, get the current lockholder details.
		kve, err := f.kv.Get(ctx, key)
		switch {
		case errors.Is(err, jetstream.ErrKeyNotFound):
			// The lock was released. Try again.
			continue
		case err != nil:
			// Unexpected error.
			return nil, stacktrace.Wrap(err)
		}

		// Parse the current value.
		var value lockValue[T]
		if err := json.Unmarshal(kve.Value(), &value); err != nil {
			// The value is garbage: delete it, ignoring any errors, and try again.
			f.opts.logger.Warn("detected garbage lock contents - deleting key", log.ErrAttr(err), slog.Uint64("rev", kve.Revision()))
			_ = f.kv.Delete(ctx, key, jetstream.LastRevision(kve.Revision()))
			continue
		}

		// If lock has expired: delete it, ignoring any errors, and try again.
		if value.ExpiresAt.Compare(time.Now()) == -1 {
			f.opts.logger.Info("detected expired lock - deleting key", slog.Uint64("rev", kve.Revision()))
			_ = f.kv.Delete(ctx, key, jetstream.LastRevision(kve.Revision()))
			continue
		}

		// The current lock is valid, and won't expire until this time.
		waitTime := time.Until(value.ExpiresAt)

		// Alternatively, the lock holder might release before then.
		watcher, err := f.kv.Watch(ctx, key, jetstream.MetaOnly(), jetstream.UpdatesOnly())
		if err != nil {
			return nil, stacktrace.Wrap(err)
		}

		// Wait until something of interest happens (ie until the lock may be available again).
		if err := wait(ctx, waitTime, watcher.Updates()); err != nil {
			return nil, stacktrace.Wrap(err)
		}
		if err := watcher.Stop(); err != nil {
			return nil, stacktrace.Wrap(err)
		}

	}
}

// Wait until either the context is done, the timer fires, or a change of the key-value is detected.
func wait(ctx context.Context, d time.Duration, changes <-chan jetstream.KeyValueEntry) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-changes:
		return nil
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

type lockValue[T any] struct {
	InstanceID string    `json:"instance_id"`
	ExpiresAt  time.Time `json:"expires_at"`
	Content    T         `json:"content,omitempty"`
}

// Lock is a distributed one-time use lock.
// The lock is backed by NATS KV store, which uses fencing to ensure correctness.
// The lock has limited time validity, and will extend that validity itself while locked.
// Should the lock fail to extend the validity, the lock can be lost and claimed by another process.
// Therefore, checks should be made to ensure the lock is still held (see Run func).
// Further reading: https://martin.kleppmann.com/2016/02/08/how-to-do-distributed-locking.html
// Further watching: https://youtu.be/XLJ5_5MsgGQ?t=1665&feature=shared
type Lock[T any] struct {
	mu         sync.Mutex
	kv         jetstream.KeyValue
	key        string
	content    T
	instanceID string
	opts       options
	rev        uint64
	locked     bool
	wg         sync.WaitGroup
	LockCtx    context.Context
	cancel     context.CancelCauseFunc
}

// Refresh the lock expiry on a regular interval.
func (l *Lock[T]) continuallyRefresh() {
	ticker := time.NewTicker(l.opts.lockRefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := l.refresh(); err != nil {
				return
			}
		case <-l.LockCtx.Done():
			l.opts.logger.Debug("lock refresh cancelled")
			return
		}
	}
}

// refresh the lock expiry
func (l *Lock[T]) refresh() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// It is possible that unlock has happened while this func was blocked
	// on the mutex. Do nothing if not locked.
	if !l.locked {
		return nil
	}

	v, err := l.Marshal(l.content)
	if err != nil {
		return stacktrace.Wrap(err)
	}
	rev, err := l.kv.Update(l.LockCtx, l.key, v, l.rev)
	switch {
	case err == nil:
		l.opts.logger.Debug("lock refreshed", slog.Uint64("rev", rev))
		l.rev = rev
		return nil
	case errors.Is(err, l.LockCtx.Err()):
		// Context was cancelled during operation.
		// Ignore the error at this point (it will immediately be handled by the caller).
		return nil
	default:
		l.opts.logger.Error("lock refresh failed", log.ErrAttr(err), slog.Uint64("rev", l.rev))
		errLostLock := errcontext.Add(ErrLockLost, slog.Uint64("rev", l.rev), slog.String("key", l.key))
		l.cancel(errors.Join(stacktrace.Wrap(errLostLock), err))
		l.rev = 0
		l.locked = false
		return stacktrace.Wrap(err)
	}
}

// Locked returns true if the lock is currently held.
func (l *Lock[T]) Locked() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.locked
}

// Unlock releases the lock.
func (l *Lock[T]) Unlock() error {
	// Ensure the refresh goroutine has stopped after this func has returned.
	// Put this defer outside the mutex to avoid deadlock.
	defer l.wg.Wait()

	l.mu.Lock()
	defer l.mu.Unlock()

	// It is possible the lock has been lost while this func was blocked
	// on the mutex. Do nothing if not locked. Also prevents double unlocking.
	if !l.locked {
		return nil
	}

	l.cancel(nil) // cancel the context without error
	oldRev := l.rev
	l.rev = 0
	l.locked = false

	timeoutCtx, cancel := context.WithTimeout(context.Background(), UnlockTimeout)
	defer cancel()

	err := l.kv.Delete(timeoutCtx, l.key, jetstream.LastRevision(oldRev))
	if err != nil {
		l.opts.logger.Warn("failed to release lock", log.ErrAttr(err), slog.Uint64("rev", oldRev))
		return stacktrace.Wrap(err)
	}
	l.opts.logger.Info("lock released", slog.Uint64("rev", oldRev))
	return nil
}

// Marshal returns a byte slice of the value to be held by the lock.
func (l *Lock[T]) Marshal(content T) ([]byte, error) {
	value := lockValue[T]{
		InstanceID: l.instanceID,
		ExpiresAt:  time.Now().Add(l.opts.lockValidityInterval).UTC(),
		Content:    content,
	}
	return json.Marshal(value)
}

// Run blocks until the lock is lost, unlocked, or the context is done.
// Run will then unlock the lock if possible.
func (l *Lock[T]) Run(ctx context.Context) error {
	// Return nil if context is already done.
	if ctx.Err() != nil {
		return nil //nolint:nilerr // intentional
	}

	// Wait for the lock to be lost, unlocked, or the context to be done.
	select {
	case <-ctx.Done():
	case <-l.LockCtx.Done():
		err := context.Cause(l.LockCtx)
		switch {
		case errors.Is(err, context.Canceled), err == nil:
			// The lock was unlocked.
		default:
			// The lock was lost.
			return stacktrace.Wrap(err)
		}
	}

	// Unlock the lock.
	if err := l.Unlock(); err != nil {
		return stacktrace.Wrap(err)
	}
	return nil
}

// Name returns the name of this lock.
func (l *Lock[T]) Name() string {
	return fmt.Sprintf("singleton-lock-%s", l.key)
}
