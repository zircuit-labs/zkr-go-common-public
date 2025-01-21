package messagebus

import (
	"context"
	"errors"
	"log/slog"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/zircuit-labs/zkr-go-common/config"
	"github.com/zircuit-labs/zkr-go-common/xerrors/stacktrace"
)

var ErrNoMessages = errors.New("no messages found")

// GetLastMessage consumes only the last published message for a give subject.
// Returns ErrNoMessages if no message exists on that subject.
func GetLastMessage[T any](cfg *config.Configuration, cfgPath string, opts ...Option) (T, *jetstream.MsgMetadata, error) {
	var data T

	options := parseOptions(opts)
	streamConfig := natsStreamConsumerConfig{}
	if err := cfg.Unmarshal(cfgPath, &streamConfig); err != nil {
		return data, nil, stacktrace.Wrap(err)
	}

	consumerConfig := jetstream.ConsumerConfig{
		Description:   streamConfig.Description,
		FilterSubject: streamConfig.Subject,
		AckPolicy:     jetstream.AckNonePolicy,     // Don't require an ACK
		DeliverPolicy: jetstream.DeliverLastPolicy, // Deliver the last message first
	}

	var nc *nats.Conn
	var js jetstream.JetStream

	if options.nc != nil {
		if options.js == nil {
			return data, nil, stacktrace.Wrap(ErrNoJetstream)
		}
		// Use provided NATS connection
		nc = options.nc
		js = options.js
	} else {
		// Set up NATS connection from config
		_nc, _js, err := NewJetStreamConnection(cfg, opts...)
		if err != nil {
			return data, nil, stacktrace.Wrap(err)
		}
		nc = _nc
		js = _js
		// Only drain the nats connection if it was one we made.
		// Otherwise the responsibility for this lies with its creator.
		defer func() { _ = nc.Drain() }()
	}

	// Create the consumer
	consumer, err := js.CreateOrUpdateConsumer(context.Background(), streamConfig.Stream, consumerConfig)
	if err != nil {
		return data, nil, stacktrace.Wrap(err)
	}

	// Fetch the single message that we care about (the last one)
	// NOTE: this is a non-blocking operation
	msgBatch, err := consumer.FetchNoWait(1)
	if err != nil {
		return data, nil, stacktrace.Wrap(err)
	}

	// Read the channel to get the message.
	// The channel will be closed once the message (or lack thereof) is pushed into it.
	msg, ok := <-msgBatch.Messages()
	if !ok {
		// The channel is already closed, therefore there are no messages
		// Although this may be a legitimately expected result, log the consumer config to help debug if it isn't
		options.logger.Info("no messages found for consumer", slog.Any("consumerConfig", consumerConfig))
		return data, nil, stacktrace.Wrap(ErrNoMessages)
	}

	// extract the message metadata
	metadata, err := msg.Metadata()
	if err != nil {
		return data, nil, stacktrace.Wrap(err)
	}

	// unmarshal the message data
	if err := options.unmarshaler(msg.Data(), &data); err != nil {
		return data, nil, stacktrace.Wrap(err)
	}

	return data, metadata, nil
}
