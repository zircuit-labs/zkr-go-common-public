package s3

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/stretchr/testify/assert"
	gomock "go.uber.org/mock/gomock"
)

func testPublicSetup(t *testing.T) (PublicBlobStore, PublicBlobStoreConfig, *MockS3Client) {
	t.Helper()
	config := PublicBlobStoreConfig{
		Endpoint:         "https://sls.s3.us-east-0.amazonaws.com",
		Bucket:           "public-snapshots",
		Region:           "us-east-0",
		S3ForcePathStyle: false,
		DisableSSL:       false,
	}

	ctrl := gomock.NewController(t)
	mockS3 := NewMockS3Client(ctrl)

	pbs := PublicBlobStore{
		bucket: config.Bucket,
		s3:     mockS3,
	}
	return pbs, config, mockS3
}

func TestNewPublicBlobStoreFromConfig(t *testing.T) {
	t.Parallel()
	_, config, _ := testPublicSetup(t)
	ctx := t.Context()
	publicBlobStore, err := NewPublicBlobStoreFromConfig(ctx, config)
	assert.NoError(t, err)
	assert.Equal(t, publicBlobStore.bucket, config.Bucket)
	assert.NotNil(t, publicBlobStore.s3)
}

func TestNewPublicBlobStoreFromConfigErrors(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	_, err := NewPublicBlobStoreFromConfig(ctx, PublicBlobStoreConfig{})
	assert.ErrorIs(t, err, ErrNoRegion)

	_, err = NewPublicBlobStoreFromConfig(ctx, PublicBlobStoreConfig{Region: "example"})
	assert.ErrorIs(t, err, ErrNoBucket)
}

func TestPublicBlobStoreGet(t *testing.T) {
	t.Parallel()
	pbs, config, mockS3 := testPublicSetup(t)
	ctx := t.Context()

	key1 := "snapshot-001.json"
	key2 := "missing.json"
	expectedData := []byte("snapshot data")

	mockS3.EXPECT().GetObject(ctx, gomock.AssignableToTypeOf(&s3.GetObjectInput{
		Bucket: aws.String(config.Bucket),
		Key:    aws.String(key1),
	})).Return(&s3.GetObjectOutput{
		Body: io.NopCloser(bytes.NewReader(expectedData)),
	}, nil)

	mockS3.EXPECT().GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(config.Bucket),
		Key:    aws.String(key2),
	}).Return(nil, &types.NoSuchKey{})

	data, err := pbs.Get(ctx, key1)
	assert.NoError(t, err)
	assert.ElementsMatch(t, expectedData, data)

	result, err := pbs.Get(ctx, key2)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrNotFound)
	assert.Nil(t, result)
}

func TestPublicBlobStoreExists(t *testing.T) {
	t.Parallel()
	pbs, config, mockS3 := testPublicSetup(t)
	ctx := t.Context()
	key1 := "snapshot-001.json"
	key2 := "missing.json"

	mockS3.EXPECT().HeadObject(ctx, gomock.AssignableToTypeOf(&s3.HeadObjectInput{
		Bucket: aws.String(config.Bucket),
		Key:    aws.String(key1),
	})).Return(&s3.HeadObjectOutput{}, nil)

	mockS3.EXPECT().HeadObject(ctx, gomock.AssignableToTypeOf(&s3.HeadObjectInput{
		Bucket: aws.String(config.Bucket),
		Key:    aws.String(key2),
	})).Return(nil, &types.NoSuchKey{})

	err := pbs.Exists(ctx, key1)
	assert.NoError(t, err)

	err = pbs.Exists(ctx, key2)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestPublicBlobStoreList(t *testing.T) {
	t.Parallel()
	pbs, config, mockS3 := testPublicSetup(t)
	ctx := t.Context()

	page1 := &s3.ListObjectsV2Output{
		Contents: []types.Object{
			{Key: aws.String("data/file1.json")},
			{Key: aws.String("data/file2.json")},
		},
		IsTruncated:           aws.Bool(true),
		NextContinuationToken: aws.String("token-123"),
	}

	page2 := &s3.ListObjectsV2Output{
		Contents: []types.Object{
			{Key: aws.String("data/file3.json")},
		},
		IsTruncated: aws.Bool(false),
	}

	gomock.InOrder(
		mockS3.EXPECT().ListObjectsV2(ctx, gomock.AssignableToTypeOf(&s3.ListObjectsV2Input{
			Bucket: aws.String(config.Bucket),
		})).Return(page1, nil),

		mockS3.EXPECT().ListObjectsV2(ctx, gomock.AssignableToTypeOf(&s3.ListObjectsV2Input{
			Bucket:            aws.String(config.Bucket),
			ContinuationToken: aws.String("token-123"),
		})).Return(page2, nil),
	)

	opts := ListOptions{}
	keys, err := pbs.List(ctx, opts)
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{"data/file1.json", "data/file2.json", "data/file3.json"}, keys)
}

func TestPublicBlobStoreListWithPrefix(t *testing.T) {
	t.Parallel()
	pbs, config, mockS3 := testPublicSetup(t)
	ctx := t.Context()

	mockS3.EXPECT().ListObjectsV2(ctx, gomock.AssignableToTypeOf(&s3.ListObjectsV2Input{
		Bucket: aws.String(config.Bucket),
		Prefix: aws.String("snapshots/2024/"),
	})).DoAndReturn(func(_ context.Context, input *s3.ListObjectsV2Input, _ ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
		assert.Equal(t, "snapshots/2024/", *input.Prefix)
		return &s3.ListObjectsV2Output{
			Contents: []types.Object{
				{Key: aws.String("snapshots/2024/file1.json")},
				{Key: aws.String("snapshots/2024/file2.json")},
			},
			IsTruncated: aws.Bool(false),
		}, nil
	})

	opts := ListOptions{
		Prefix: "snapshots/2024/",
	}
	keys, err := pbs.List(ctx, opts)
	assert.NoError(t, err)
	assert.Len(t, keys, 2)
	assert.ElementsMatch(t, []string{"snapshots/2024/file1.json", "snapshots/2024/file2.json"}, keys)
}

func TestPublicBlobStoreListWithMaxKeys(t *testing.T) {
	t.Parallel()
	pbs, config, mockS3 := testPublicSetup(t)
	ctx := t.Context()

	mockS3.EXPECT().ListObjectsV2(ctx, gomock.AssignableToTypeOf(&s3.ListObjectsV2Input{
		Bucket:  aws.String(config.Bucket),
		MaxKeys: aws.Int32(50),
	})).DoAndReturn(func(_ context.Context, input *s3.ListObjectsV2Input, _ ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
		assert.Equal(t, int32(50), *input.MaxKeys)
		return &s3.ListObjectsV2Output{
			Contents: []types.Object{
				{Key: aws.String("file1.json")},
			},
			IsTruncated: aws.Bool(false),
		}, nil
	})

	opts := ListOptions{
		MaxKeys: 50,
	}
	keys, err := pbs.List(ctx, opts)
	assert.NoError(t, err)
	assert.Len(t, keys, 1)
}

func TestPublicBlobStoreListWithMaxPages(t *testing.T) {
	t.Parallel()
	pbs, config, mockS3 := testPublicSetup(t)
	ctx := t.Context()

	page1 := &s3.ListObjectsV2Output{
		Contents: []types.Object{
			{Key: aws.String("file1.json")},
			{Key: aws.String("file2.json")},
		},
		IsTruncated:           aws.Bool(true),
		NextContinuationToken: aws.String("token-1"),
	}

	page2 := &s3.ListObjectsV2Output{
		Contents: []types.Object{
			{Key: aws.String("file3.json")},
		},
		IsTruncated:           aws.Bool(true),
		NextContinuationToken: aws.String("token-2"),
	}

	gomock.InOrder(
		mockS3.EXPECT().ListObjectsV2(ctx, gomock.AssignableToTypeOf(&s3.ListObjectsV2Input{
			Bucket: aws.String(config.Bucket),
		})).Return(page1, nil),

		mockS3.EXPECT().ListObjectsV2(ctx, gomock.AssignableToTypeOf(&s3.ListObjectsV2Input{
			Bucket:            aws.String(config.Bucket),
			ContinuationToken: aws.String("token-1"),
		})).Return(page2, nil),
	)

	opts := ListOptions{
		MaxPages: 2,
	}
	keys, err := pbs.List(ctx, opts)
	assert.NoError(t, err)
	assert.Len(t, keys, 3)
	assert.ElementsMatch(t, []string{"file1.json", "file2.json", "file3.json"}, keys)
}

func TestPublicBlobStoreListWithAllOptions(t *testing.T) {
	t.Parallel()
	pbs, config, mockS3 := testPublicSetup(t)
	ctx := t.Context()

	mockS3.EXPECT().ListObjectsV2(ctx, gomock.AssignableToTypeOf(&s3.ListObjectsV2Input{
		Bucket:  aws.String(config.Bucket),
		Prefix:  aws.String("data/2024/"),
		MaxKeys: aws.Int32(100),
	})).DoAndReturn(func(_ context.Context, input *s3.ListObjectsV2Input, _ ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
		assert.Equal(t, "data/2024/", *input.Prefix)
		assert.Equal(t, int32(100), *input.MaxKeys)
		return &s3.ListObjectsV2Output{
			Contents: []types.Object{
				{Key: aws.String("data/2024/snapshot1.json")},
				{Key: aws.String("data/2024/snapshot2.json")},
			},
			IsTruncated: aws.Bool(false),
		}, nil
	})

	opts := ListOptions{
		Prefix:   "data/2024/",
		MaxKeys:  100,
		MaxPages: 1,
	}
	keys, err := pbs.List(ctx, opts)
	assert.NoError(t, err)
	assert.Len(t, keys, 2)
}

func TestPublicBlobStoreListContextCancel(t *testing.T) {
	t.Parallel()
	pbs, _, mockS3 := testPublicSetup(t)
	ctx, cancel := context.WithCancel(t.Context())

	cancel()

	mockS3.EXPECT().ListObjectsV2(gomock.Any(), gomock.Any()).Times(0)
	opts := ListOptions{Prefix: "test/"}
	keys, err := pbs.List(ctx, opts)
	assert.Nil(t, keys)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, context.Canceled))
}

func TestPublicBlobStoreListError(t *testing.T) {
	t.Parallel()
	pbs, config, mockS3 := testPublicSetup(t)
	ctx := t.Context()

	mockS3.EXPECT().ListObjectsV2(ctx, gomock.AssignableToTypeOf(&s3.ListObjectsV2Input{
		Bucket: aws.String(config.Bucket),
	})).Return(nil, assert.AnError)

	opts := ListOptions{}
	keys, err := pbs.List(ctx, opts)
	assert.Error(t, err)
	assert.Nil(t, keys)
}
