# Retry

This package provides highly customizable retry functionality.

## Usage

### Basic

Assuming you have a `func foo() error` to be run, it's as simple as:

```go
r := NewRetrier()
err := r.Try(ctx, foo)
```

### Advanced

Retry `foo` and then `bar` using the same settings:

- make no more than 10 attempts
- only retry errors which have been explicitly classed as transient
- use exponential backoff starting at 100ms, but no more than 30s; and
- using "Equal" jitter: actual delay is [n/2, n)

```go
r := NewRetrier(
    WithStrategy(
        strategy.NewExponential(
            time.Millisecond*100,
            time.Second*30,
            strategy.WithJitter(
                jitter.Equal(),
            ),
        )
    ),
    WithMaxAttempts(10),
    WithUnknownErrorsAs(errclass.Persistent),
)

err = r.Try(ctx, foo)
err = r.Try(ctx, bar)
```

Note that the passed context is respected and can be used to cut the retry attempts short from outside. It is recommended that the passed function also be made context aware. For example:

```go
func baz(ctx context.Context) error {
    // do something and respect ctx
}

func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    err := r.Try(func() error {
        return baz(ctx)
    })
}
```

## Options

### Strategy

Use the option `WithStrategy` to set a custom back off delay strategy. See `retry/strategy` for several pre-written options.

By default, the back off delay starts at 1 second and doubles each time to a maximum of 1 minute, but also uses full jitter so the exact delay is randomized.

#### Jitter

When providing a strategy, you may also set a custom jitter strategy using the options `WithJitter` (or `WithoutJitter` if preferred). There are a selection of pre-written jitter options in `retry/jitter`

### MaxAttempts

Use the option `WithMaxAttempts` to put a limit on the number of attempts that should be made to execute the function. A value less of less than 1 will be considered as infinite (default).

### Errors with Unknown Class

This package is best used in conjunction with `xerrors/errclass` in order to specify if an error is transient or persistent. However, if an error was not classified in this way it will be designated as an `Unknown` class.

The option `WithUnknownErrorsAs` allows users to specify how these should be treated. By default, they are considered `Transient` and will be retried.

## Panics

The provided function is executed wrapped in `calm.Unpanic` which will recover from a panic and return an error instead. In such a case, no further attempts will be made, and the error along with information about the panic will be returned from `Try`

## Additional Failure Information

In the failure scenario, `Try` will return the last encountered error wrapped alongside `RetryStats` using `xerrors`. Use `xerrors.Extract` to access these from a returned error:

```go
stats, ok := xerrors.Extract[RetryStats](err)
```
