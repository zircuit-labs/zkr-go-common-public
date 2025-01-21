package strategy

import "github.com/zircuit-labs/zkr-go-common/retry/jitter"

type options struct {
	jitterFunc jitter.Transformation
	base       int // field to define the power base
}

type Option func(options *options)

// WithJitter allows users to specify usage jitter function of their choice.
func WithJitter(jitterFunc jitter.Transformation) Option {
	return func(options *options) {
		options.jitterFunc = jitterFunc
	}
}

// WithoutJitter allows users to forego jitter function usage.
func WithoutJitter() Option {
	return func(options *options) {
		options.jitterFunc = jitter.None()
	}
}

// WithBase allows users to specify the base for the exponential backoff.
func WithBase(base int) Option {
	return func(options *options) {
		options.base = base
	}
}
