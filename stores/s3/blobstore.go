package s3

import (
	"bytes"
	"context"
	"errors"
	"log/slog"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/zircuit-labs/zkr-go-common/config"
	"github.com/zircuit-labs/zkr-go-common/xerrors/errcontext"
	"github.com/zircuit-labs/zkr-go-common/xerrors/stacktrace"
)

var (
	ErrNoRegion = errors.New("no region supplied")
	ErrNoBucket = errors.New("no bucket supplied")
	ErrNotFound = errors.New("entity not found")
)

type BlobStore struct {
	bucket string
	s3     *s3.S3
}

type BlobStoreConfig struct {
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

func NewBlobStoreFromConfig(config BlobStoreConfig) (*BlobStore, error) {
	if config.Region == "" {
		return nil, stacktrace.Wrap(ErrNoRegion)
	}
	if config.Bucket == "" {
		return nil, stacktrace.Wrap(ErrNoBucket)
	}

	awsConfig := &aws.Config{
		Endpoint:         aws.String(config.Endpoint),
		Region:           aws.String(config.Region),
		S3ForcePathStyle: aws.Bool(config.S3ForcePathStyle),
		DisableSSL:       aws.Bool(config.DisableSSL),
	}
	if config.AccessKeyID != "" && config.SecretAccessKey != "" {
		awsConfig.Credentials = credentials.NewStaticCredentials(
			config.AccessKeyID, config.SecretAccessKey, "",
		)
	}

	session, err := session.NewSession(awsConfig)
	if err != nil {
		return nil, stacktrace.Wrap(err)
	}

	s3Session := s3.New(session)
	return &BlobStore{
		bucket: config.Bucket,
		s3:     s3Session,
	}, nil
}

func NewBlobStore(cfg *config.Configuration, cfgPath string) (*BlobStore, error) {
	config := BlobStoreConfig{}
	if err := cfg.Unmarshal(cfgPath, &config); err != nil {
		return nil, stacktrace.Wrap(err)
	}

	return NewBlobStoreFromConfig(config)
}

func (b *BlobStore) Upload(ctx context.Context, key string, data []byte) error {
	_, err := b.s3.PutObjectWithContext(ctx, &s3.PutObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(key),
		Body:   aws.ReadSeekCloser(bytes.NewReader(data)),
	})
	if err != nil {
		return stacktrace.Wrap(err)
	}

	return nil
}

func (b *BlobStore) Get(ctx context.Context, key string) (res []byte, err error) {
	defer func() {
		err = errcontext.Add(err, slog.String("key", key))
	}()

	data, err := b.s3.GetObjectWithContext(ctx, &s3.GetObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		//nolint:errorlint // This is the AWS SDK error handling pattern
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeNoSuchKey:
				return nil, ErrNotFound
			default:
				return nil, stacktrace.Wrap(err)
			}
		}
		return nil, stacktrace.Wrap(err)
	}
	defer data.Body.Close()

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(data.Body)
	if err != nil {
		return nil, stacktrace.Wrap(err)
	}

	return buf.Bytes(), nil
}

func (b *BlobStore) Exists(ctx context.Context, key string) (err error) {
	defer func() {
		err = errcontext.Add(err, slog.String("key", key))
	}()

	_, err = b.s3.HeadObjectWithContext(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		//nolint:errorlint // This is the AWS SDK error handling pattern
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeNoSuchKey:
				return ErrNotFound
			default:
				return stacktrace.Wrap(err)
			}
		}
		return stacktrace.Wrap(err)
	}

	return nil
}
