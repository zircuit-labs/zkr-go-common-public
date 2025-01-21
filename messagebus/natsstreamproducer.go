package messagebus

import (
	"context"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/zircuit-labs/zkr-go-common/config"
	"github.com/zircuit-labs/zkr-go-common/xerrors/stacktrace"
)

// required config for a streaming producer
type natsStreamProducerConfig struct {
	// Subject identifies where to produce messages to
	Subject string
}

// NatsStreamProducer produces messages using NATS JetStream
type NatsStreamProducer[T any] struct {
	config           natsStreamProducerConfig
	nc               *nats.Conn
	shouldCloseNC    bool
	js               jetstream.JetStream
	opts             options
	subjectTransform func(data T, defaultSubject string) string
}

func nilTransform[T any](_ T, defaultSubject string) string {
	return defaultSubject
}

// NewNatsStreamProducer creates a new NatsStreamProducer
func NewNatsStreamProducer[T any](cfg *config.Configuration, cfgPath string, opts ...Option) (*NatsStreamProducer[T], error) {
	options := parseOptions(opts)

	// Parse and validate stream config.
	streamConfig := natsStreamProducerConfig{}
	if err := cfg.Unmarshal(cfgPath, &streamConfig); err != nil {
		return nil, stacktrace.Wrap(err)
	}
	if streamConfig.Subject == "" {
		return nil, stacktrace.Wrap(ErrNoSubject)
	}

	producer := NatsStreamProducer[T]{
		config:           streamConfig,
		opts:             options,
		subjectTransform: nilTransform[T],
	}

	if options.nc != nil {
		if options.js == nil {
			return nil, stacktrace.Wrap(ErrNoJetstream)
		}
		// Use provided NATS connection
		producer.nc = options.nc
		producer.js = options.js
	} else {
		// Set up NATS connection from config
		nc, js, err := NewJetStreamConnection(cfg, opts...)
		if err != nil {
			return nil, stacktrace.Wrap(err)
		}
		producer.shouldCloseNC = true
		producer.nc = nc
		producer.js = js
	}

	return &producer, nil
}

// SetSubjectTransform allows for users to set dynamic subjects on which to produce based on the input data.
func (n *NatsStreamProducer[T]) SetSubjectTransform(f func(data T, defaultSubject string) string) {
	n.subjectTransform = f
}

// Produce sends the data to the stream
func (n *NatsStreamProducer[T]) Produce(ctx context.Context, data T) error {
	b, err := n.opts.marshaler(&data)
	if err != nil {
		return stacktrace.Wrap(err)
	}

	err = n.opts.retrier.Try(ctx, func() error {
		sub := n.subjectTransform(data, n.config.Subject)
		_, err = n.js.Publish(ctx, sub, b)
		if err != nil {
			return stacktrace.Wrap(err)
		}
		return nil
	})

	return err
}

// Close terminates the connections
func (n *NatsStreamProducer[T]) Close() {
	// Only close the nats connection if it was one we made.
	// Otherwise the responsibility for this lies with its creator.
	if n.shouldCloseNC {
		n.nc.Close()
	}
}
