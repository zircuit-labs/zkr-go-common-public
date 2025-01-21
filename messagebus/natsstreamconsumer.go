package messagebus

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/zircuit-labs/zkr-go-common/calm/errgroup"
	"github.com/zircuit-labs/zkr-go-common/config"
	"github.com/zircuit-labs/zkr-go-common/log"
	"github.com/zircuit-labs/zkr-go-common/task/polling"
	"github.com/zircuit-labs/zkr-go-common/xerrors/errclass"
	"github.com/zircuit-labs/zkr-go-common/xerrors/stacktrace"
)

const (
	// The default AckWait is 30 seconds, meaning any message that
	// hasn't been given an Ack or an InProgress will be resent.
	// Use 15 seconds as the default time to send InProgress updates.
	defaultInProgressInterval = 15 * time.Second

	// This is the maximum time we will ask NATS to wait before redelivering a message
	maxNakDelay = time.Minute
	// This is the minimum time we will ask NATS to wait before redelivering a message
	baseNakDelay = time.Millisecond * 100
)

// required config for a stream consumer
type natsStreamConsumerConfig struct {
	Stream       string
	DurableQueue string `koanf:"durablequeue"`
	Description  string
	Subject      string
}

// ConsumerHandler handles the incoming messages
// using generic type T allows us to abstract the JSON unmarshal
type ConsumerHandler[T any] interface {
	HandleMessage(ctx context.Context, data T, subject string, metadata jetstream.MsgMetadata) error
}

// NatsStreamConsumer is a Task does the dirty work of talking to NATS Jetstream
// allowing users to focus on handling the messages with the ConsumerHandler
type NatsStreamConsumer[T any] struct {
	nc            *nats.Conn
	shouldCloseNC bool
	js            jetstream.JetStream
	consumer      jetstream.Consumer
	handler       ConsumerHandler[T]
	opts          options
}

// NewNatsStreamConsumer creates a new NatsStreamConsumer
func NewNatsStreamConsumer[T any](cfg *config.Configuration, cfgPath string, handler ConsumerHandler[T], opts ...Option) (*NatsStreamConsumer[T], error) {
	// Prepare all the options and config
	options := parseOptions(opts)
	streamConfig := natsStreamConsumerConfig{}
	if err := cfg.Unmarshal(cfgPath, &streamConfig); err != nil {
		return nil, stacktrace.Wrap(err)
	}

	// Set up consumer config
	var consumerConfig jetstream.ConsumerConfig
	if options.consumerConfig != nil {
		// If consumer config option was provided, use that
		consumerConfig = *options.consumerConfig
	} else {
		// Otherwise use defaults and provided values
		consumerConfig = jetstream.ConsumerConfig{
			Durable:       streamConfig.DurableQueue,
			Description:   streamConfig.Description,
			FilterSubject: streamConfig.Subject,
		}

		// Use the durable queue name if provided
		if options.durableQueue != "" {
			consumerConfig.Durable = options.durableQueue
		}

		// If a subject can change (ie there is a transform), then the consumer durable name should be unique to the subject.
		// Otherwise a previous durable consumer could have skipped a message that the new consumer wants, but will never get.
		// For this reason, also set the inactive threshold to 15 minutes so that old consumers are cleaned up.
		if len(options.consumerSubjectTransform) > 0 {
			consumerConfig.FilterSubject = transformSubject(consumerConfig.FilterSubject, options.consumerSubjectTransform)
			consumerConfig.InactiveThreshold = time.Minute * 15
			if consumerConfig.Durable != "" {
				// Names must not contain certain characters, therefore we cannot directly reference the subject.
				// See https://docs.nats.io/running-a-nats-service/nats_admin/jetstream_admin/naming
				consumerConfig.Durable = consumerConfig.Durable + "-" + subjectHash(consumerConfig.FilterSubject)
			}
		}

	}

	natsStreamConsumer := NatsStreamConsumer[T]{
		handler: handler,
		opts:    options,
	}

	if options.nc != nil {
		if options.js == nil {
			return nil, stacktrace.Wrap(ErrNoJetstream)
		}
		// Use provided NATS connection
		natsStreamConsumer.nc = options.nc
		natsStreamConsumer.js = options.js
	} else {
		// Set up NATS connection from config
		nc, js, err := NewJetStreamConnection(cfg, opts...)
		if err != nil {
			return nil, stacktrace.Wrap(err)
		}
		natsStreamConsumer.shouldCloseNC = true
		natsStreamConsumer.nc = nc
		natsStreamConsumer.js = js
	}

	// Create the consumer
	consumer, err := natsStreamConsumer.js.CreateOrUpdateConsumer(context.Background(), streamConfig.Stream, consumerConfig)
	if err != nil {
		return nil, stacktrace.Wrap(err)
	}
	natsStreamConsumer.consumer = consumer

	return &natsStreamConsumer, nil
}

// HealthCheck returns an error if the NATS connection is not "connected".
func (n *NatsStreamConsumer[T]) HealthCheck(ctx context.Context) error {
	if n.nc.Status() != nats.CONNECTED {
		return stacktrace.Wrap(ErrNATSNotConnected)
	}

	return nil
}

// Name returns the name of this task
func (n *NatsStreamConsumer[T]) Name() string {
	return fmt.Sprintf("nats-stream-consumer (%s)", n.consumer.CachedInfo().Config.Durable)
}

// Run consumes messages from NATS and passes them to the handler
func (n *NatsStreamConsumer[T]) Run(ctx context.Context) error {
	// Only close the nats connection if it was one we made.
	// Otherwise the responsibility for this lies with its creator.
	if n.shouldCloseNC {
		defer n.nc.Close()
	}

	consumerErrChan := make(chan error)
	defer close(consumerErrChan)

	// Handle messages
	cc, err := n.consumer.Consume(
		// handle consumer messages
		func(msg jetstream.Msg) {
			n.handleMessage(ctx, msg)
		},
		// handle consumer errors
		jetstream.ConsumeErrHandler(func(cc jetstream.ConsumeContext, err error) {
			// stop immediately to avoid causing further errors
			// however ErrNoHeartbeat is safe to ignore so long as we still have a valid
			// connection to nats server
			if errors.Is(err, nats.ErrNoHeartbeat) || errors.Is(err, jetstream.ErrNoHeartbeat) {
				if n.nc.Status() != nats.CONNECTED {
					cc.Stop()
					consumerErrChan <- stacktrace.Wrap(ErrNATSNotConnected)
				}
			} else {
				cc.Stop()
				consumerErrChan <- stacktrace.Wrap(err)
			}
		}),
	)
	if err != nil {
		return stacktrace.Wrap(err)
	}
	defer cc.Stop()

	// Run until stopped or consumer error
	select {
	case <-ctx.Done():
		return nil
	case err := <-consumerErrChan:
		return stacktrace.Wrap(err)
	}
}

func (n *NatsStreamConsumer[T]) handleMessage(ctx context.Context, msg jetstream.Msg) {
	meta, err := msg.Metadata()
	if err != nil || meta == nil {
		// This should never happen, but if it does we should log an error and retry the message later
		n.opts.logger.Error("failed to fetch message metadata", log.ErrAttr(err), slog.String("task", n.Name()), slog.String("subject", msg.Subject()))
		_ = msg.NakWithDelay(baseNakDelay)
		return
	}
	logger := n.opts.logger.With(
		slog.String("task", n.Name()),
		slog.String("subject", msg.Subject()),
		slog.Uint64("sequence_number", meta.Sequence.Stream),
		slog.Uint64("delivery_attempt", meta.NumDelivered),
	)

	var data T
	err = n.opts.unmarshaler(msg.Data(), &data)
	if err != nil {
		// If we can't unmarshal the data, it's useless to us.
		// Log a warning, and consider it otherwise handled.
		logger.Error("failed to unmarshal data - skipping", log.ErrAttr(err),
			slog.String("comment", "This should never happen, and a human needs to investigate how and why it did."))
		return
	}

	// The default `AckWait` for NATS consumers is 30 seconds.
	// If the message is not acked within that time frame, it will be resent.
	// Since we expect messages may take much longer to process than that,
	// this block will send an InProgress message, which resets the AckWait countdown,
	// at regular intervals while the message is being worked on.
	progressAcker := newInProgressAcker(msg, n.opts.inProgressInterval)
	innerCtx, cancel := context.WithCancel(ctx)
	g := errgroup.New()

	// Call the handler to deal with the message.
	// Cancel the innerCtx when done in order to stop the progressAcker
	g.Go(func() error {
		defer cancel()
		metadata, err := msg.Metadata()
		if err != nil {
			return stacktrace.Wrap(err)
		} else if metadata == nil {
			return stacktrace.Wrap(errors.New("metadata is nil"))
		}
		return n.handler.HandleMessage(innerCtx, data, msg.Subject(), *metadata)
	})
	// Meanwhile, run the progressAcker (always returns nil)
	g.Go(func() error {
		return progressAcker.Run(innerCtx)
	})

	err = g.Wait()
	var ackErr error
	switch errclass.GetClass(err) {
	case errclass.Nil:
		ackErr = msg.Ack()
	case errclass.Persistent, errclass.Panic:
		logger.Error("failed to handle message - skipping", log.ErrAttr(err),
			slog.String("comment", "This indicates that a message is lost, and a human needs to investigate."))
		ackErr = msg.Ack()
	default: // errclass.Transient or error class was not explicitly set
		delay := calculateNakDelay(meta)
		ackErr = msg.NakWithDelay(delay)
		if meta.NumDelivered < 10 {
			logger.Warn("failed to handle message - will retry", log.ErrAttr(err), slog.Duration("delay", delay))
		} else {
			logger.Error("failed to handle message - will retry", log.ErrAttr(err), slog.Duration("delay", delay),
				slog.String("comment", "This message has been retried at least 10 times. A human needs to investigate"))
		}
	}

	if ackErr != nil {
		logger.Warn("failed to ack/nak message", log.ErrAttr(err))
	}
}

func newInProgressAcker(msg jetstream.Msg, d time.Duration) *polling.Task {
	action := inProgressAction{Msg: msg}
	// NOTE: never include WithTerminateOnError option since we don't want
	// a failure to send the InProgress message to result in a message handling error.
	options := []polling.Option{
		polling.WithRunAtStart(),
		polling.WithInterval(d),
	}
	return polling.NewTask("msg in progress acker", &action, options...)
}

type inProgressAction struct {
	Msg jetstream.Msg
}

func (a *inProgressAction) Run(_ context.Context) error {
	return a.Msg.InProgress()
}

func (a *inProgressAction) Cleanup() {}

// When we intentionally Nak a message (because there was an error in handling it),
// If we don't provide a delay value then NATS will retry it again instantly.
// Most likely we don't want to spam ourselves, but we don't want to wait forever either.
// This helper will use the message metadata to calculate a bounded doubling backoff strategy
func calculateNakDelay(meta *jetstream.MsgMetadata) time.Duration {
	// don't bother with calculation after the 10th attempt
	if meta.NumDelivered <= 10 {
		calculatedDelay := time.Duration(2^meta.NumDelivered) * baseNakDelay
		if calculatedDelay < maxNakDelay {
			return calculatedDelay
		}
	}

	return maxNakDelay
}

func transformSubject(subject string, transform map[string]string) string {
	for k, v := range transform {
		subject = strings.ReplaceAll(subject, k, v)
	}
	return subject
}

func subjectHash(subject string) string {
	hash := fnv.New64a()
	hash.Write([]byte(subject))
	return strconv.FormatUint(hash.Sum64(), 16)
}
