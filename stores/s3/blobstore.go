package s3

import (
	"bytes"
	"context"
	"errors"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/zircuit-labs/zkr-go-common/config"
	"github.com/zircuit-labs/zkr-go-common/xerrors/errcontext"
	"github.com/zircuit-labs/zkr-go-common/xerrors/stacktrace"
)

var (
	ErrNoRegion = errors.New("no region supplied")
	ErrNoBucket = errors.New("no bucket supplied")
	ErrNotFound = errors.New("entity not found")
)

type S3Client interface {
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
	ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
	DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
}

type BlobStore struct {
	bucket string
	s3     S3Client
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

func NewBlobStoreFromConfig(ctx context.Context, config BlobStoreConfig) (*BlobStore, error) {
	if config.Region == "" {
		return nil, stacktrace.Wrap(ErrNoRegion)
	}
	if config.Bucket == "" {
		return nil, stacktrace.Wrap(ErrNoBucket)
	}

	// Create the S3 client
	var awsConfig aws.Config
	var err error
	if config.AccessKeyID != "" && config.SecretAccessKey != "" {
		awsConfig, err = awsconfig.LoadDefaultConfig(ctx,
			awsconfig.WithRegion(config.Region),
			awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				config.AccessKeyID,
				config.SecretAccessKey,
				"",
			)),
		)
	} else {
		awsConfig, err = awsconfig.LoadDefaultConfig(ctx,
			awsconfig.WithRegion(config.Region),
		)
	}
	if err != nil {
		return nil, stacktrace.Wrap(err)
	}

	// Create S3 client with custom options
	clientOptions := []func(*s3.Options){
		func(o *s3.Options) {
			if config.Endpoint != "" {
				o.BaseEndpoint = aws.String(config.Endpoint)
			}
			o.UsePathStyle = config.S3ForcePathStyle
			if config.DisableSSL {
				o.EndpointOptions.DisableHTTPS = true
			}
		},
	}

	s3Client := s3.NewFromConfig(awsConfig, clientOptions...)
	return &BlobStore{
		bucket: config.Bucket,
		s3:     s3Client,
	}, nil
}

func NewBlobStore(ctx context.Context, cfg *config.Configuration, cfgPath string) (*BlobStore, error) {
	config := BlobStoreConfig{}
	if err := cfg.Unmarshal(cfgPath, &config); err != nil {
		return nil, stacktrace.Wrap(err)
	}

	return NewBlobStoreFromConfig(ctx, config)
}

func (b *BlobStore) SetBucket(bucket string) {
	b.bucket = bucket
}

func (b *BlobStore) GetBucket() string {
	return b.bucket
}

func (b *BlobStore) Upload(ctx context.Context, key string, data []byte) error {
	_, err := b.s3.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(data),
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

	data, err := b.s3.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var noSuchKey *types.NoSuchKey
		if errors.As(err, &noSuchKey) {
			return nil, stacktrace.Wrap(ErrNotFound)
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

	_, err = b.s3.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var (
			noSuchKey *types.NoSuchKey
			notFound  *types.NotFound
		)
		if errors.As(err, &noSuchKey) || errors.As(err, &notFound) {
			return stacktrace.Wrap(ErrNotFound)
		}
		return stacktrace.Wrap(err)
	}

	return nil
}

func (b *BlobStore) GetAllList(ctx context.Context) ([]string, error) {
	var keys []string
	var continuationToken *string

	for {
		// handle context cancellation
		select {
		case <-ctx.Done():
			return nil, stacktrace.Wrap(ctx.Err())
		default:
		}

		output, err := b.s3.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket:            aws.String(b.bucket),
			ContinuationToken: continuationToken,
		})
		if err != nil {
			return nil, stacktrace.Wrap(err)
		}

		for _, obj := range output.Contents {
			if obj.Key != nil {
				keys = append(keys, *obj.Key)
			}
		}

		if output.IsTruncated == nil || !*output.IsTruncated {
			break
		}
		continuationToken = output.NextContinuationToken
	}

	return keys, nil
}

func (b *BlobStore) Delete(ctx context.Context, key string) (err error) {
	defer func() {
		err = errcontext.Add(err, slog.String("key", key))
	}()

	_, err = b.s3.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return stacktrace.Wrap(err)
	}
	return nil
}
