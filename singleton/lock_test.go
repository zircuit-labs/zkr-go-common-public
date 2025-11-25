package singleton_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zircuit-labs/zkr-go-common/calm/errgroup"
	zkrlog "github.com/zircuit-labs/zkr-go-common/log"
	"github.com/zircuit-labs/zkr-go-common/messagebus/testutils"
	"github.com/zircuit-labs/zkr-go-common/singleton"
)

const (
	lockRefreshInterval  = time.Millisecond * 10
	lockValidityInterval = time.Millisecond * 100
)

func createLockFactory[T any](t *testing.T, nc *nats.Conn, logger *slog.Logger) *singleton.LockFactory[T] {
	t.Helper()

	lockFactory, err := singleton.NewLockFactory[T](
		nc,
		xid.New().String(),
		singleton.WithLogger(logger),
		singleton.WithLockRefreshInterval(lockRefreshInterval),
		singleton.WithLockValidityInterval(lockValidityInterval),
	)
	require.NoError(t, err)
	return lockFactory
}

func TestLockLost(t *testing.T) { //nolint:paralleltest // parallel exposes a data race in the nats server code itself, but does not affect the validity of this test/code.
	natsServer := testutils.NewEmbeddedServer(t)
	t.Cleanup(natsServer.Close)
	nc, js := natsServer.Conn(t)
	t.Cleanup(nc.Close)

	// create the lock factory
	logger := zkrlog.NewTestLogger(t)
	lockFactory := createLockFactory[any](t, nc, logger)

	// acquire the lock
	ctx := t.Context()
	lock, err := lockFactory.CreateLock(ctx, t.Name(), nil)
	require.NoError(t, err)
	require.True(t, lock.Locked())

	// run the lock in the background
	eg := errgroup.New()
	eg.Go(func() error {
		return lock.Run(ctx)
	})

	// Outside of the lock context, delete the lock value causing the lock to be lost
	kv, err := js.KeyValue(ctx, singleton.BucketName)
	require.NoError(t, err)
	err = kv.Delete(ctx, t.Name())
	require.NoError(t, err)

	// lock.Run() should return ErrLockLost
	// (the refresh will fail due to revision change)
	err = eg.Wait()
	assert.ErrorIs(t, err, singleton.ErrLockLost)
	assert.False(t, lock.Locked())
}

func TestLockLostConnection(t *testing.T) { //nolint:paralleltest // parallel exposes a data race in the nats server code itself, but does not affect the validity of this test/code.
	natsServer := testutils.NewEmbeddedServer(t)
	t.Cleanup(natsServer.Close)
	nc, _ := natsServer.Conn(t)
	t.Cleanup(nc.Close)

	// create the lock factory
	logger := zkrlog.NewTestLogger(t)
	lockFactory := createLockFactory[any](t, nc, logger)

	// acquire the lock
	ctx := t.Context()
	lock, err := lockFactory.CreateLock(ctx, t.Name(), nil)
	require.NoError(t, err)
	require.True(t, lock.Locked())

	// run the lock in the background
	eg := errgroup.New()
	eg.Go(func() error {
		return lock.Run(ctx)
	})

	// Close the nats connection used by the lock
	nc.Close()

	// lock.Run() should return ErrLockLost
	// (the refresh will fail due to closed connection)
	err = eg.Wait()
	assert.ErrorIs(t, err, singleton.ErrLockLost)
	assert.False(t, lock.Locked())
}

func TestRun(t *testing.T) { //nolint:paralleltest // parallel exposes a data race in the nats server code itself, but does not affect the validity of this test/code.
	natsServer := testutils.NewEmbeddedServer(t)
	t.Cleanup(natsServer.Close)
	nc, _ := natsServer.Conn(t)
	t.Cleanup(nc.Close)

	// create the lock factory
	logger := zkrlog.NewTestLogger(t)
	lockFactory := createLockFactory[any](t, nc, logger)

	// acquire the lock
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	lock, err := lockFactory.CreateLock(ctx, t.Name(), nil)
	require.NoError(t, err)
	require.True(t, lock.Locked())

	// run the lock in the background
	eg := errgroup.New()
	eg.Go(func() error {
		return lock.Run(ctx)
	})

	// wait long enough for the lock to be refreshed multiple times
	time.Sleep(lockRefreshInterval * 5)

	// cancel the context to stop the lock task and unlock the lock
	cancel()

	// lock.Run() should return nil
	err = eg.Wait()
	assert.NoError(t, err)
	assert.False(t, lock.Locked())
}

func TestTryCreateLock(t *testing.T) { //nolint:paralleltest // parallel exposes a data race in the nats server code itself, but does not affect the validity of this test/code.
	natsServer := testutils.NewEmbeddedServer(t)
	t.Cleanup(natsServer.Close)
	nc, _ := natsServer.Conn(t)
	t.Cleanup(nc.Close)

	// create the lock factory
	logger := zkrlog.NewTestLogger(t)
	lockFactory := createLockFactory[string](t, nc, logger)

	// acquire the lock
	ctx := t.Context()
	lockA, err := lockFactory.CreateLock(ctx, t.Name(), "lockA content")
	require.NoError(t, err)
	require.True(t, lockA.Locked())

	// try to acquire the lock again with a different lock content (same key)
	lockB, content, err := lockFactory.TryCreateLock(ctx, t.Name(), "lockB content")
	require.NoError(t, err)
	assert.Nil(t, lockB)

	// Since A already acquired the lock, B should not be able to acquire it
	// Instead, B should get the content of the lock (A's content)
	require.NotNil(t, content)
	assert.Equal(t, "lockA content", *content)

	// unlock A
	err = lockA.Unlock()
	require.NoError(t, err)
	require.False(t, lockA.Locked())

	// try to acquire the lock again with the same lock content (same key)
	lockC, content, err := lockFactory.TryCreateLock(ctx, t.Name(), "lockC content")
	require.NoError(t, err)
	require.True(t, lockC.Locked())
	assert.Nil(t, content)
	require.NotNil(t, lockC)

	// unlock C
	err = lockC.Unlock()
	require.NoError(t, err)
	require.False(t, lockC.Locked())
}

type ab struct {
	idx   int
	value int
}

func pushValues(size, value int, ch chan ab) {
	for i := range size {
		ch <- ab{
			idx:   i,
			value: value,
		}
	}
}

func collectValues(size int, ch chan ab, out chan []int) {
	res := make([]int, size)
	for s := range ch {
		res[s.idx] = s.value
	}
	out <- res
	close(out)
}

func valuesIdentical(values []int) bool {
	if len(values) < 2 {
		return true
	}
	for i := 1; i < len(values); i++ {
		if values[0] != values[i] {
			return false
		}
	}
	return true
}

var (
	size          = 50
	instanceCount = 10
)

// TestCanary proves that this setup without locks results in non-deterministic behavior
// It is possible that this could fail, but it should be extremely rare.
// Each of the instanceCount go routines will write to a channel which sets a value
// in an array equal to the instance number. Without locks, we expect the array
// at the end to be filled with different numbers (ie not all the same).
func TestCanary(t *testing.T) { //nolint:paralleltest // parallel exposes a data race in the nats server code itself, but does not affect the validity of this test/code.
	// Don't regularly run this test:
	// it only exists to prove non-deterministic behavior can happen without careful locking.
	t.Skip("This test is intentionally disabled as it is only a proof-of-need")

	ch := make(chan ab)
	out := make(chan []int)

	eg := errgroup.New()
	for i := range instanceCount {
		eg.Go(func() error {
			pushValues(size, i, ch)
			return nil
		})
	}

	go collectValues(size, ch, out)
	require.NoError(t, eg.Wait())
	close(ch)
	res := <-out
	assert.False(t, valuesIdentical(res))
}

// TestLock takes the same setup as TestCanary but adds the singleton lock
// to each goroutine. The end result is still non-deterministic, but
// we expect each value in the array to no be identical.
func TestLock(t *testing.T) { //nolint:paralleltest // parallel exposes a data race in the nats server code itself, but does not affect the validity of this test/code.
	ch := make(chan ab)
	out := make(chan []int)

	natsServer := testutils.NewEmbeddedServer(t)
	t.Cleanup(natsServer.Close)
	nc, _ := natsServer.Conn(t)
	t.Cleanup(nc.Close)

	// create all the locks first
	logger := zkrlog.NewTestLogger(t)
	lockFactories := make([]*singleton.LockFactory[string], instanceCount)
	locks := make([]*singleton.Lock[string], instanceCount)
	for i := range instanceCount {
		lockFactories[i] = createLockFactory[string](t, nc, logger)
	}

	eg := errgroup.New()
	for i := range instanceCount {
		// Each instance should get the lock, and only then do the work
		// at the same time as the work, check for lock lost via Run().
		// No lock should ever be lost, and the Unlock() call should
		// terminate run without error.
		// Do all this inside a goroutine so each has a chance of
		// getting the lock first.
		eg.Go(func() error {
			lock, err := lockFactories[i].CreateLock(t.Context(), t.Name(), "test")
			if err != nil {
				return err
			}
			locks[i] = lock
			eg.Go(func() error {
				pushValues(size, i, ch)
				return locks[i].Unlock()
			})
			eg.Go(func() error {
				return locks[i].Run(t.Context())
			})
			return nil
		})
	}

	go collectValues(size, ch, out)
	err := eg.Wait()
	assert.NoError(t, err)
	close(ch)
	res := <-out
	assert.True(t, valuesIdentical(res))
}
