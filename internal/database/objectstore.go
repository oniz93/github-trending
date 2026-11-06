package database

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// DefaultReadmeBucket is the bucket README files are stored in when none is configured.
const DefaultReadmeBucket = "readmes"

// ObjectStore is the subset of S3 operations the services rely on. It is
// implemented by S3Connection and can be mocked in tests.
type ObjectStore interface {
	UploadFile(ctx context.Context, objectName string, reader io.Reader, objectSize int64, contentType string) error
	GetFile(ctx context.Context, objectName string) ([]byte, error)
	Close() error
}

// S3Connection talks to any S3-compatible object store (SeaweedFS, Garage,
// AWS S3, ...) through the AWS SDK.
type S3Connection struct {
	Client *s3.Client
	Bucket string
}

var _ ObjectStore = (*S3Connection)(nil)

// normalizeEndpoint turns "host:port" into "http://host:port" and leaves
// endpoints that already carry a scheme untouched.
func normalizeEndpoint(endpoint string) string {
	endpoint = strings.TrimRight(strings.TrimSpace(endpoint), "/")
	if strings.HasPrefix(endpoint, "http://") || strings.HasPrefix(endpoint, "https://") {
		return endpoint
	}
	return "http://" + endpoint
}

// NewS3Connection connects to an S3-compatible endpoint and makes sure the
// bucket exists, creating it if needed.
func NewS3Connection(endpoint, accessKeyID, secretAccessKey, bucket string) (*S3Connection, error) {
	if endpoint == "" {
		return nil, errors.New("S3 endpoint is not configured")
	}
	if bucket == "" {
		bucket = DefaultReadmeBucket
	}

	cfg := aws.Config{
		Region:      "us-east-1", // required by SigV4; ignored by single-region self-hosted stores
		Credentials: credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, ""),
		// Only send/verify checksums when the API requires them: not every
		// S3-compatible server supports the SDK's default trailing checksums.
		RequestChecksumCalculation: aws.RequestChecksumCalculationWhenRequired,
		ResponseChecksumValidation: aws.ResponseChecksumValidationWhenRequired,
	}
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(normalizeEndpoint(endpoint))
		o.UsePathStyle = true
	})

	ctx := context.Background()
	if _, err := client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(bucket)}); err != nil {
		var notFound *types.NotFound
		if !errors.As(err, &notFound) {
			return nil, fmt.Errorf("failed to check S3 bucket %q: %w", bucket, err)
		}
		log.Printf("S3 bucket '%s' does not exist, creating it...", bucket)
		if _, err := client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(bucket)}); err != nil {
			var owned *types.BucketAlreadyOwnedByYou
			if !errors.As(err, &owned) {
				return nil, fmt.Errorf("failed to create S3 bucket %q: %w", bucket, err)
			}
		} else {
			log.Printf("S3 bucket '%s' created successfully.", bucket)
		}
	}

	return &S3Connection{Client: client, Bucket: bucket}, nil
}

// Close is a no-op: the S3 client holds no resources that need releasing.
func (c *S3Connection) Close() error {
	return nil
}

// UploadFile stores an object in the bucket.
func (c *S3Connection) UploadFile(ctx context.Context, objectName string, reader io.Reader, objectSize int64, contentType string) error {
	// SigV4 over plain HTTP needs a seekable body to hash the payload.
	body, ok := reader.(io.ReadSeeker)
	if !ok {
		data, err := io.ReadAll(reader)
		if err != nil {
			return fmt.Errorf("failed to read object %q: %w", objectName, err)
		}
		body = bytes.NewReader(data)
		objectSize = int64(len(data))
	}

	_, err := c.Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(c.Bucket),
		Key:           aws.String(objectName),
		Body:          body,
		ContentLength: aws.Int64(objectSize),
		ContentType:   aws.String(contentType),
	})
	if err != nil {
		return fmt.Errorf("failed to upload object %q to S3: %w", objectName, err)
	}
	return nil
}

// GetFile retrieves an object from the bucket.
func (c *S3Connection) GetFile(ctx context.Context, objectName string) ([]byte, error) {
	out, err := c.Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.Bucket),
		Key:    aws.String(objectName),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get object %q from S3: %w", objectName, err)
	}
	defer out.Body.Close()

	data, err := io.ReadAll(out.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read object %q from S3: %w", objectName, err)
	}
	return data, nil
}
