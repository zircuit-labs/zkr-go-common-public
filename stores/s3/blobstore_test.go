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
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
)

func testSetup(t *testing.T) (BlobStore, BlobStoreConfig, *MockS3Client) {
	t.Helper()
	// bucket, region, s3forcepathstyle, disableSSL in 1pw
	// accessKeyId, secretAccessKey in 1pw
	// endpoint search for S3_URL in k8s repo
	config := BlobStoreConfig{
		Endpoint:         "https://sls.s3.us-east-0.amazonaws.com",
		AccessKeyID:      "AJHSHGJEGJHGS",                        // random
		SecretAccessKey:  "asdjhka23z16jhs!sd.sadcjAKJ(1$jsadad", // random
		Bucket:           "snapshots",
		Region:           "us-east-0",
		S3ForcePathStyle: false,
		DisableSSL:       false,
	}

	ctrl := gomock.NewController(t)
	mockS3 := NewMockS3Client(ctrl)

	bs := BlobStore{
		bucket: config.Bucket,
		s3:     mockS3,
	}
	return bs, config, mockS3
}

func TestNewBlobStoreFromConfig(t *testing.T) {
	t.Parallel()
	_, config, _ := testSetup(t)
	ctx := t.Context()
	blobStore, err := NewBlobStoreFromConfig(ctx, config)
	assert.NoError(t, err)
	assert.Equal(t, blobStore.bucket, config.Bucket)
	assert.NotNil(t, blobStore.s3)
}

func TestNewBlobStoreFromConfigErrors(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	_, err := NewBlobStoreFromConfig(ctx, BlobStoreConfig{})
	assert.ErrorIs(t, err, ErrNoRegion)

	_, err = NewBlobStoreFromConfig(ctx, BlobStoreConfig{Region: "example"})
	assert.ErrorIs(t, err, ErrNoBucket)
}

func TestGetSetBucket(t *testing.T) {
	t.Parallel()
	bs, config, _ := testSetup(t)
	assert.Equal(t, config.Bucket, bs.GetBucket())

	newBucket := "diffs"
	bs.SetBucket(newBucket)
	assert.Equal(t, newBucket, bs.GetBucket())
}

func TestUpload(t *testing.T) {
	t.Parallel()
	bs, config, mockS3 := testSetup(t)
	ctx := t.Context()

	key := "0x3f5c394d3f3e89ea1a6f51e65f8b5d7cf055c7e8b19e1bc19b1db3b1a424e5e5.json.gz"
	data := []byte("world")

	mockS3.EXPECT().PutObject(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, input *s3.PutObjectInput, _ ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
		// assert that input parameters match expectations
		assert.Equal(t, config.Bucket, *input.Bucket)
		assert.Equal(t, key, *input.Key)

		// verify body content
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(input.Body)
		assert.Equal(t, data, buf.Bytes())

		return &s3.PutObjectOutput{}, nil
	})

	err := bs.Upload(ctx, key, data)
	require.NoError(t, err)
}

func TestGet(t *testing.T) {
	t.Parallel()
	bs, config, mockS3 := testSetup(t)
	ctx := t.Context()

	key1 := "example.txt"
	key2 := "missing.txt"
	expectedData := []byte("hello world")

	mockS3.EXPECT().GetObject(ctx, gomock.AssignableToTypeOf(&s3.GetObjectInput{
		Bucket: aws.String(config.Bucket),
		Key:    aws.String(key1),
	})).Return(&s3.GetObjectOutput{
		Body: io.NopCloser(bytes.NewReader(expectedData)),
	}, nil)
	mockS3.EXPECT().GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(config.Bucket),
		Key:    aws.String(key2),
	}).
		Return(nil, assert.AnError)

	// expect success
	data, err := bs.Get(ctx, key1)
	assert.NoError(t, err)
	assert.ElementsMatch(t, expectedData, data)

	// expect error
	result, err := bs.Get(ctx, key2)
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestExists(t *testing.T) {
	t.Parallel()
	bs, config, mockS3 := testSetup(t)
	ctx := t.Context()
	key := "example.txt"

	mockS3.EXPECT().HeadObject(ctx, gomock.AssignableToTypeOf(&s3.HeadObjectInput{
		Bucket: aws.String(config.Bucket),
		Key:    aws.String(key),
	})).Return(&s3.HeadObjectOutput{}, nil)

	err := bs.Exists(ctx, key)
	assert.NoError(t, err)
}

func TestGetAll(t *testing.T) {
	t.Parallel()
	bs, config, mockS3 := testSetup(t)
	ctx := t.Context()

	page1 := &s3.ListObjectsV2Output{
		Contents: []types.Object{
			{Key: aws.String("file1.txt")},
			{Key: aws.String("file2.txt")},
		},
		IsTruncated:           aws.Bool(true),
		NextContinuationToken: aws.String("token-123"),
	}

	page2 := &s3.ListObjectsV2Output{
		Contents: []types.Object{
			{Key: aws.String("file3.txt")},
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

	expected := []string{"file1.txt", "file2.txt", "file3.txt"}

	keyList, err := bs.GetAllList(ctx)
	assert.NoError(t, err)
	assert.ElementsMatch(t, expected, keyList)
}

func TestContextCancel(t *testing.T) {
	t.Parallel()
	bs, _, mockS3 := testSetup(t)
	ctx, cancel := context.WithCancel(t.Context())

	// immediate cancel
	cancel()

	// call 0 times
	mockS3.EXPECT().ListObjectsV2(gomock.Any(), gomock.Any()).Times(0)
	keys, err := bs.GetAllList(ctx)
	assert.Nil(t, keys)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, context.Canceled))

	// cancel context mid loop
	ctx, cancel = context.WithCancel(context.Background())

	mockS3.EXPECT().ListObjectsV2(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, input *s3.ListObjectsV2Input, _ ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
		// Cancel context after first page fetch
		defer cancel()
		// Return one page of results
		return &s3.ListObjectsV2Output{
			Contents:              []types.Object{{Key: aws.String("file1.txt")}},
			IsTruncated:           aws.Bool(true), // simulate pagination
			NextContinuationToken: aws.String("token-1"),
		}, nil
	}).Times(1)

	_, err = bs.GetAllList(ctx)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, context.Canceled))
}

func TestDelete(t *testing.T) {
	t.Parallel()
	bs, config, mockS3 := testSetup(t)
	ctx := t.Context()

	key1 := "file-to-delete.txt"
	key2 := "nonexistent.txt"

	mockS3.EXPECT().DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(config.Bucket),
		Key:    aws.String(key1),
	}).Return(&s3.DeleteObjectOutput{}, nil)
	mockS3.EXPECT().DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(config.Bucket),
		Key:    aws.String(key2),
	}).Return(nil, assert.AnError)

	// success
	err := bs.Delete(ctx, key1)
	assert.NoError(t, err)

	// error
	err = bs.Delete(ctx, key2)
	assert.Error(t, err)
}
