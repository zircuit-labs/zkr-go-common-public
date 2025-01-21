# Calm

This package provides standardized panic recovery mechanisms.

Ideally code should never panic. If it does, that generally means that something is wrong and there's no way for the code to continue. However, exiting the process immediately is also less than ideal, especially if there is an opportunity to gracefully shutdown instead.

## Unpanic

The function `calm.Unpanic(f func() error) error` runs the provided function `f()` and simply returns its return value directly - unless `f` causes a panic. In that case, `Unpanic` will recover from the panic, collect a stack trace from the panic, and then return an error containing the data from the panic and the stack trace.

### WARNING

`Unpanic` is not a panacea. It is not possible to recover from a panic that happens inside a goroutine, and worse still such a panic will terminate the entire runtime. To avoid this, use `Unpanic` to wrap the entirety of any code that is to be run in a goroutine - but be mindful that you will not have such a luxury with imported libraries.

## Errgroup

`golang.org/x/sync/errgroup` is a wonderful tool for synchronizing multiple goroutines. `calm/errgroup` is just a wrapper which ensures the goroutines are wrapped with `Unpanic`.
