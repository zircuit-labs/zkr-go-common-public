# Package `singleton`

This package provides a distributed locking backed by NATS KV store, which uses fencing to ensure correctness. The lock has limited time validity, and will extend that validity itself while locked.

**NOTE:** Should the lock fail to extend the validity, the lock can be lost and claimed by another process. Therefore, checks should be made to ensure the lock is still held (see Run func).

## Usage

1. Create a lock factory to create and acquire locks.
2. Obtain a lock before doing any critical work.
    - DO NOT close the provided nats connection until everything is complete.
2. Execute critical work, with ability to stop if the lock is lost.
3. In parallel to work, check for lock loss using `Lock.Run(ctx)`
    - If this returns any error, immediately stop work.
4. Unlock the lock when work is done (or simply cancel the context passed to `Run`).

Alternatively, `TryCreateLock` can be used to create and acquire a lock, or in the event that the lock with the same key already exists and is locked, obtain the data held by that lock. This may be useful for passing information about the current lock holder. The data can be of any type, so long as it can be (un)marshalled to/from JSON (This is be decided by the factory type at compile time).
