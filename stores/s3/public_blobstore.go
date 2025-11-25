package s3

import (
	"bytes"
	"context"
	"errors"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/zircuit-labs/zkr-go-common/config"
	"github.com/zircuit-labs/zkr-go-common/xerrors/errcontext"
	"github.com/zircuit-labs/zkr-go-common/xerrors/stacktrace"
)

// PublicBlobStore is a read-only S3 client for accessing public buckets without credentials
type PublicBlobStore struct {
	bucket string
	s3     S3Client
}

type PublicBlobStoreConfig struct {
	Endpoint string `koanf:"endpoint"`
	Bucket   string `koanf:"bucket"`
	Region   string `koanf:"region"`

	// Set to true for minio, false for AWS
	S3ForcePathStyle bool `koanf:"s3forcepathstyle"`
	// Set to true for minio, false for AWS
	DisableSSL bool `koanf:"disablessl"`
}

// NewPublicBlobStoreFromConfig creates a new PublicBlobStore from the provided config
// This client uses anonymous credentials and is intended for read-only access to public buckets
func NewPublicBlobStoreFromConfig(ctx context.Context, config PublicBlobStoreConfig) (*PublicBlobStore, error) {
	if config.Region == "" {
		return nil, stacktrace.Wrap(ErrNoRegion)
	}
	if config.Bucket == "" {
		return nil, stacktrace.Wrap(ErrNoBucket)
	}

	// Create AWS config with anonymous credentials
	awsConfig, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(config.Region),
		awsconfig.WithCredentialsProvider(aws.AnonymousCredentials{}),
	)
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
	return &PublicBlobStore{
		bucket: config.Bucket,
		s3:     s3Client,
	}, nil
}

// NewPublicBlobStore creates a new PublicBlobStore from the application configuration
func NewPublicBlobStore(ctx context.Context, cfg *config.Configuration, cfgPath string) (*PublicBlobStore, error) {
	config := PublicBlobStoreConfig{}
	if err := cfg.Unmarshal(cfgPath, &config); err != nil {
		return nil, stacktrace.Wrap(err)
	}

	return NewPublicBlobStoreFromConfig(ctx, config)
}

// Get retrieves an object from the public S3 bucket
func (p *PublicBlobStore) Get(ctx context.Context, key string) (res []byte, err error) {
	defer func() {
		err = errcontext.Add(err, slog.String("key", key))
	}()

	data, err := p.s3.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(p.bucket),
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

// Exists checks if an object exists in the public S3 bucket
func (p *PublicBlobStore) Exists(ctx context.Context, key string) (err error) {
	defer func() {
		err = errcontext.Add(err, slog.String("key", key))
	}()

	_, err = p.s3.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var noSuchKey *types.NoSuchKey
		var notFound *types.NotFound
		if errors.As(err, &noSuchKey) || errors.As(err, &notFound) {
			return stacktrace.Wrap(ErrNotFound)
		}
		return stacktrace.Wrap(err)
	}

	return nil
}

// ListOptions contains options for listing objects in an S3 bucket.
//
// Fields:
//   - Prefix: Filter objects by key prefix (e.g., "snapshots/2024/"). Empty string means no filter.
//   - MaxKeys: Maximum number of keys to return per page (S3 default is 1000 if not set).
//   - MaxPages: Maximum number of pages to retrieve. Set to 0 for unlimited pages (will retrieve all matching objects).
//
// Example:
//
//	opts := s3.ListOptions{
//	    Prefix:   "data/2024/",
//	    MaxKeys:  100,
//	    MaxPages: 5,
//	}
//	keys, err := client.List(ctx, opts)
type ListOptions struct {
	Prefix   string
	MaxKeys  int32
	MaxPages int
}

// List lists objects in the public S3 bucket with optional prefix filtering and pagination control.
// This method is recommended over GetAllList() for buckets with many objects as it provides
// fine-grained control over pagination to prevent timeouts.
//
// The opts parameter controls filtering and pagination:
//   - Use Prefix to filter objects by key prefix (server-side filtering)
//   - Use MaxKeys to limit results per API call (helps with performance)
//   - Use MaxPages to limit total number of API calls (prevents timeout on large buckets)
//
// Returns a slice of object keys matching the criteria.
func (p *PublicBlobStore) List(ctx context.Context, opts ListOptions) ([]string, error) {
	var keys []string
	var continuationToken *string
	pagesRetrieved := 0

	for {
		select {
		case <-ctx.Done():
			return nil, stacktrace.Wrap(ctx.Err())
		default:
		}

		input := &s3.ListObjectsV2Input{
			Bucket:            aws.String(p.bucket),
			ContinuationToken: continuationToken,
		}

		if opts.Prefix != "" {
			input.Prefix = aws.String(opts.Prefix)
		}

		if opts.MaxKeys > 0 {
			input.MaxKeys = aws.Int32(opts.MaxKeys)
		}

		output, err := p.s3.ListObjectsV2(ctx, input)
		if err != nil {
			return nil, stacktrace.Wrap(err)
		}

		for _, obj := range output.Contents {
			if obj.Key != nil {
				keys = append(keys, *obj.Key)
			}
		}

		pagesRetrieved++

		// Check if we should stop pagination
		if output.IsTruncated == nil || !*output.IsTruncated {
			break
		}

		if opts.MaxPages > 0 && pagesRetrieved >= opts.MaxPages {
			break
		}

		continuationToken = output.NextContinuationToken
	}

	return keys, nil
}
