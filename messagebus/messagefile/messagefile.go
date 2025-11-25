package messagefile

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/zircuit-labs/zkr-go-common/log"
	"github.com/zircuit-labs/zkr-go-common/xerrors/stacktrace"
)

var (
	ErrInvalidMessageLocation = errors.New("invalid message location")
	ErrS3Disabled             = errors.New("s3 functionality is disabled")
	ErrNotFound               = errors.New("entity not found")
)

type MessageLocation struct {
	S3ObjectName string `json:"s3_name,omitempty"`   // S3 object bucket name
	S3BucketName string `json:"s3_bucket,omitempty"` // S3 bucket name
	FilePath     string `json:"path,omitempty"`      // File path for locally stored messages
}

type (
	UnmarshalFn func(data []byte, v any) error
)

type options struct {
	logger      *slog.Logger
	unmarshaler UnmarshalFn
	s3Config    *S3Config
}

func parseOptions(opts []Option) options {
	// Set up default options
	options := options{
		logger:      log.NewNilLogger(),
		unmarshaler: json.Unmarshal,
	}

	// Apply provided options
	for _, opt := range opts {
		opt(&options)
	}

	return options
}

type Option func(options *options)

// WithLogger sets the logger to be used.
func WithLogger(logger *slog.Logger) Option {
	return func(options *options) {
		options.logger = logger
	}
}

// WithUnmarshal allows for alternative data serialization formats.
func WithUnmarshal(unmarshaler UnmarshalFn) Option {
	return func(options *options) {
		options.unmarshaler = unmarshaler
	}
}

// With S3Config sets the S3 configuration.
func WithS3Config(cfg S3Config) Option {
	return func(options *options) {
		options.s3Config = &cfg
	}
}

type S3Config struct {
	Endpoint        string `koanf:"endpoint"`
	AccessKeyID     string `koanf:"accesskeyid"`
	SecretAccessKey string `koanf:"secretaccesskey"`
	Bucket          string `koanf:"bucket"`
	Region          string `koanf:"region"`

	// Set to true for minio, false for AWS
	S3ForcePathStyle bool `koanf:"s3forcepathstyle"`
	// Set to true for minio, false for AWS
	DisableSSL bool `koanf:"disablessl"`
}

type MessageFetcher[T any] struct {
	s3   *s3.Client
	opts options
}

func NewMessageFetcher[T any](ctx context.Context, opts ...Option) (*MessageFetcher[T], error) {
	options := parseOptions(opts)

	fetcher := MessageFetcher[T]{
		opts: options,
	}

	// If S3 is disabled, nothing more to do.
	if options.s3Config == nil {
		return &fetcher, nil
	}

	// Create the S3 client
	var awsConfig aws.Config
	var err error
	if options.s3Config.AccessKeyID != "" && options.s3Config.SecretAccessKey != "" {
		awsConfig, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(options.s3Config.Region),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				options.s3Config.AccessKeyID,
				options.s3Config.SecretAccessKey,
				"",
			)),
		)
	} else {
		awsConfig, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(options.s3Config.Region),
		)
	}
	if err != nil {
		return nil, stacktrace.Wrap(err)
	}

	// Create S3 client with custom options
	clientOptions := []func(*s3.Options){
		func(o *s3.Options) {
			if options.s3Config.Endpoint != "" {
				o.BaseEndpoint = aws.String(options.s3Config.Endpoint)
			}
			o.UsePathStyle = options.s3Config.S3ForcePathStyle
			if options.s3Config.DisableSSL {
				o.EndpointOptions.DisableHTTPS = true
			}
		},
	}

	fetcher.s3 = s3.NewFromConfig(awsConfig, clientOptions...)
	return &fetcher, nil
}

func (f MessageFetcher[T]) Fetch(ctx context.Context, ml *MessageLocation) (T, error) {
	switch {
	case ml.FilePath != "":
		return f.fetchLocal(ml)
	case ml.S3BucketName != "" && ml.S3ObjectName != "":
		return f.fetchS3(ctx, ml)
	default:
		var t T
		return t, stacktrace.Wrap(ErrInvalidMessageLocation)
	}
}

func (f MessageFetcher[T]) fetchS3(ctx context.Context, ml *MessageLocation) (T, error) {
	var t T

	if f.s3 == nil {
		return t, stacktrace.Wrap(ErrS3Disabled)
	}

	// Read the file from S3
	data, err := f.s3.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(ml.S3BucketName),
		Key:    aws.String(ml.S3ObjectName),
	})
	if err != nil {
		var noSuchKey *types.NoSuchKey
		if errors.As(err, &noSuchKey) {
			return t, stacktrace.Wrap(ErrNotFound)
		}
		return t, stacktrace.Wrap(err)
	}
	defer data.Body.Close()

	// Read the data from the body
	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(data.Body)
	if err != nil {
		return t, stacktrace.Wrap(err)
	}

	// Unmarshal the data into the type T
	err = json.Unmarshal(buf.Bytes(), &t)
	if err != nil {
		return t, stacktrace.Wrap(err)
	}

	return t, nil
}

func (f MessageFetcher[T]) fetchLocal(ml *MessageLocation) (T, error) {
	var t T

	// Read the file from the local path
	data, err := os.ReadFile(ml.FilePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return t, stacktrace.Wrap(ErrNotFound)
		}
		return t, stacktrace.Wrap(err)
	}

	// Unmarshal the data into the type T
	err = json.Unmarshal(data, &t)
	if err != nil {
		return t, stacktrace.Wrap(err)
	}

	return t, nil
}
