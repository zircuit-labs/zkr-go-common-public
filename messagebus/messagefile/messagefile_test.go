package messagefile_test

import (
	"embed"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/zircuit-labs/zkr-go-common/log"
	"github.com/zircuit-labs/zkr-go-common/messagebus/messagefile"
)

//go:embed testdata/*
var fs embed.FS

type SampleMessage struct {
	Content    string    `json:"content"`
	Timestamp  time.Time `json:"timestamp"`
	IntValue   int       `json:"int_value"`
	ArrayValue []int     `json:"array_value"`
}

func TestFileFetcher(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	logger := log.NewTestLogger(t)

	fetcher, err := messagefile.NewMessageFetcher[SampleMessage](ctx, messagefile.WithLogger(logger))
	require.NoError(t, err)

	// failed deserialization
	_, err = fetcher.Fetch(ctx, &messagefile.MessageLocation{
		FilePath: "testdata/bad.txt",
	})
	assert.Error(t, err)

	// file does not exist
	_, err = fetcher.Fetch(ctx, &messagefile.MessageLocation{
		FilePath: "testdata/does_not_exist",
	})
	assert.ErrorIs(t, err, messagefile.ErrNotFound)

	// s3 disabled
	_, err = fetcher.Fetch(ctx, &messagefile.MessageLocation{
		S3ObjectName: "object",
		S3BucketName: "bucket",
	})
	assert.ErrorIs(t, err, messagefile.ErrS3Disabled)

	// invalid message location input
	_, err = fetcher.Fetch(ctx, &messagefile.MessageLocation{})
	assert.ErrorIs(t, err, messagefile.ErrInvalidMessageLocation)

	// happy path
	expected := SampleMessage{
		Content:    "Hello World!",
		Timestamp:  time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC),
		IntValue:   42,
		ArrayValue: []int{1, 2, 3},
	}

	actual, err := fetcher.Fetch(ctx, &messagefile.MessageLocation{
		FilePath: "testdata/good.json",
	})
	require.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func TestS3Fetcher_Docker(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	logger := log.NewTestLogger(t)

	// Set up S3 via LocalStack
	awsEndpoint := "http://localhost:4566"
	awsRegion := "us-east-1"
	awsBucket := "test-bucket"

	req := testcontainers.ContainerRequest{
		Image:        "localstack/localstack",
		ExposedPorts: []string{"4566:4566/tcp"},
		WaitingFor:   wait.ForLog("Ready"),
	}

	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		if err := c.Terminate(t.Context()); err != nil {
			t.Log(err)
		}
	})

	awsConfig, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(awsRegion),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("test", "test", "")),
	)
	require.NoError(t, err)

	s3Client := s3.NewFromConfig(awsConfig, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(awsEndpoint)
		o.UsePathStyle = true
		o.EndpointOptions.DisableHTTPS = true
	})

	// Create a bucket
	_, err = s3Client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(awsBucket),
	})
	require.NoError(t, err)

	// Validate bucket creation
	bucketList, err := s3Client.ListBuckets(ctx, &s3.ListBucketsInput{})
	require.NoError(t, err)
	require.Equal(t, len(bucketList.Buckets), 1)
	require.Equal(t, *bucketList.Buckets[0].Name, awsBucket)

	// Upload files
	file, err := fs.Open("testdata/good.json")
	require.NoError(t, err)
	_, err = s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(awsBucket),
		Key:    aws.String("good.json"),
		Body:   file,
	})
	require.NoError(t, err)

	file, err = fs.Open("testdata/bad.txt")
	require.NoError(t, err)
	_, err = s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(awsBucket),
		Key:    aws.String("bad.txt"),
		Body:   file,
	})
	require.NoError(t, err)

	// Set up fetcher
	fetcher, err := messagefile.NewMessageFetcher[SampleMessage](ctx,
		messagefile.WithLogger(logger),
		messagefile.WithS3Config(messagefile.S3Config{
			Endpoint:         awsEndpoint,
			AccessKeyID:      "test",
			SecretAccessKey:  "test",
			Bucket:           awsBucket,
			Region:           awsRegion,
			S3ForcePathStyle: true,
			DisableSSL:       true,
		}),
	)
	require.NoError(t, err)

	// failed deserialization
	_, err = fetcher.Fetch(ctx, &messagefile.MessageLocation{
		S3BucketName: awsBucket,
		S3ObjectName: "bad.txt",
	})
	assert.Error(t, err)

	// file does not exist
	_, err = fetcher.Fetch(ctx, &messagefile.MessageLocation{
		S3BucketName: awsBucket,
		S3ObjectName: "does_not_exist",
	})
	assert.ErrorIs(t, err, messagefile.ErrNotFound)

	// invalid message location input
	_, err = fetcher.Fetch(ctx, &messagefile.MessageLocation{})
	assert.ErrorIs(t, err, messagefile.ErrInvalidMessageLocation)

	// happy path
	expected := SampleMessage{
		Content:    "Hello World!",
		Timestamp:  time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC),
		IntValue:   42,
		ArrayValue: []int{1, 2, 3},
	}

	actual, err := fetcher.Fetch(ctx, &messagefile.MessageLocation{
		S3BucketName: awsBucket,
		S3ObjectName: "good.json",
	})
	require.NoError(t, err)
	assert.Equal(t, expected, actual)
}
