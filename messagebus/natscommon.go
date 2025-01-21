package messagebus

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/zircuit-labs/zkr-go-common/config"
	"github.com/zircuit-labs/zkr-go-common/log"
	"github.com/zircuit-labs/zkr-go-common/retry"
	"github.com/zircuit-labs/zkr-go-common/xerrors/stacktrace"
)

const (
	natsConfigPath = "nats"
)

var (
	ErrNoSubject        = fmt.Errorf("must provide a subject")
	ErrNATSNotConnected = fmt.Errorf("nats: status is not connected")
	ErrNoJetstream      = fmt.Errorf("nats: jetstream not supported")
)

type natsCommonConfig struct {
	Address         string
	CredentialsPath string `koanf:"credentialspath"` // Use this for .creds files
	UserJWT         string `koanf:"userjwt"`         // Or use UserJWT and NKeySeed for passing values directly.
	NKeySeed        string `koanf:"nkeyseed"`
}

// NewNatsConnection creates a new NATS connection.
func NewNatsConnection(cfg *config.Configuration, opts ...Option) (*nats.Conn, error) {
	options := parseOptions(opts)

	// Set default value
	natsConfig := natsCommonConfig{
		Address: nats.DefaultURL,
	}

	// Update value from config
	if err := cfg.Unmarshal(options.natsConnectionConfigPath, &natsConfig); err != nil {
		return nil, stacktrace.Wrap(err)
	}

	// prepare connection options
	connectionOptions := make([]nats.Option, 0)

	// add user credentials
	if natsConfig.CredentialsPath != "" {
		connectionOptions = append(connectionOptions, nats.UserCredentials(natsConfig.CredentialsPath))
	} else if natsConfig.UserJWT != "" && natsConfig.NKeySeed != "" {
		connectionOptions = append(connectionOptions, nats.UserJWTAndSeed(natsConfig.UserJWT, natsConfig.NKeySeed))
	}

	// Connect to NATS
	nc, err := nats.Connect(natsConfig.Address, connectionOptions...)
	if err != nil {
		return nil, stacktrace.Wrap(err)
	}

	return nc, nil
}

// NewJetStreamConnection creates a new NATS connection and a JetStream context.
func NewJetStreamConnection(cfg *config.Configuration, opts ...Option) (*nats.Conn, jetstream.JetStream, error) {
	// Set up NATS connection.
	nc, err := NewNatsConnection(cfg, opts...)
	if err != nil {
		return nil, nil, err
	}

	// Setup JetStream connection.
	js, err := jetstream.New(nc)
	if err != nil {
		return nil, nil, stacktrace.Wrap(err)
	}

	return nc, js, nil
}

type (
	MarshalFn   func(v any) ([]byte, error)
	UnmarshalFn func(data []byte, v any) error
)

type Retrier interface {
	Try(ctx context.Context, f func() error) error
}

type options struct {
	logger                   *slog.Logger
	marshaler                MarshalFn
	unmarshaler              UnmarshalFn
	retrier                  Retrier
	inProgressInterval       time.Duration
	consumerConfig           *jetstream.ConsumerConfig
	nc                       *nats.Conn
	js                       jetstream.JetStream
	natsConnectionConfigPath string
	consumerSubjectTransform map[string]string
	durableQueue             string
}

func parseOptions(opts []Option) options {
	// Set up default options
	options := options{
		logger:                   log.NewNilLogger(),
		marshaler:                json.Marshal,
		unmarshaler:              json.Unmarshal,
		retrier:                  retry.NewRetrier(retry.WithMaxAttempts(10)),
		inProgressInterval:       defaultInProgressInterval,
		consumerConfig:           nil,
		nc:                       nil,
		js:                       nil,
		natsConnectionConfigPath: natsConfigPath,
	}

	// Apply provided options
	for _, opt := range opts {
		opt(&options)
	}

	return options
}

// Option is an option func for NewManager.
type Option func(options *options)

// WithLogger sets the logger to be used.
func WithLogger(logger *slog.Logger) Option {
	return func(options *options) {
		options.logger = logger
	}
}

// WithDataSerialization sets an alternative method to serialize data.
func WithDataSerialization(marshaler MarshalFn, unmarshaler UnmarshalFn) Option {
	return func(options *options) {
		options.marshaler = marshaler
		options.unmarshaler = unmarshaler
	}
}

// WithRetrier allows users to specify a retry mechanism to use.
func WithRetrier(retrier Retrier) Option {
	return func(options *options) {
		options.retrier = retrier
	}
}

// WithInProgressInterval sets the interval to be used for sending InProgress updates.
func WithInProgressInterval(d time.Duration) Option {
	return func(options *options) {
		options.inProgressInterval = d
	}
}

// WithConsumerConfig allows for overriding the default consumer config with a custom one.
func WithConsumerConfig(consumerConfig *jetstream.ConsumerConfig) Option {
	return func(options *options) {
		options.consumerConfig = consumerConfig
	}
}

// WithNATSConnection allows for providing a ready-made nats connection.
func WithNATSConnection(nc *nats.Conn) Option {
	return func(options *options) {
		options.nc = nc
		js, err := jetstream.New(nc)
		if err == nil {
			options.js = js
		}
	}
}

// WithNATSConnectionConfigPath allows to set the cfgPath to the nats connection config.
func WithNATSConnectionConfigPath(configPath string) Option {
	return func(options *options) {
		options.natsConnectionConfigPath = configPath
	}
}

// WithConsumerSubjectTransform allows for transforming the subject before creating a consumer.
func WithConsumerSubjectTransform(transform map[string]string) Option {
	return func(options *options) {
		options.consumerSubjectTransform = transform
	}
}

// WithDurableQueue allows for setting the durable queue name outside of the consumer config.
func WithDurableQueue(queue string) Option {
	return func(options *options) {
		options.durableQueue = queue
	}
}
