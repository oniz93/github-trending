package database

import (
	"context"
	"fmt"
	"io"
	"log"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinioConnection holds the MinIO client.
type MinioConnection struct {
	Client *minio.Client
	Bucket string
}

// NewMinioConnection creates a new connection to the MinIO server.
func NewMinioConnection(endpoint, accessKeyID, secretAccessKey string) (*MinioConnection, error) {
	// Use SSL if endpoint starts with https://
	useSSL := false
	if len(endpoint) > 8 && endpoint[0:8] == "https://" {
		useSSL = true
		endpoint = endpoint[8:] // Remove https://
	} else if len(endpoint) > 7 && endpoint[0:7] == "http://" {
		endpoint = endpoint[7:] // Remove http://
	}

	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO client: %w", err)
	}

	// Check if the bucket exists. If not, create it.
	bucketName := "readmes"
	exists, err := minioClient.BucketExists(context.Background(), bucketName)
	if err != nil {
		return nil, fmt.Errorf("failed to check MinIO bucket existence: %w", err)
	}

	if !exists {
		log.Printf("MinIO bucket '%s' does not exist, creating it...", bucketName)
		err = minioClient.MakeBucket(context.Background(), bucketName, minio.MakeBucketOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to create MinIO bucket: %w", err)
		}
		log.Printf("MinIO bucket '%s' created successfully.", bucketName)
	}

	return &MinioConnection{Client: minioClient, Bucket: bucketName}, nil
}

// UploadFile uploads a file to MinIO.
func (mc *MinioConnection) UploadFile(ctx context.Context, objectName string, reader io.Reader, objectSize int64, contentType string) (minio.UploadInfo, error) {
	info, err := mc.Client.PutObject(ctx, mc.Bucket, objectName, reader, objectSize, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return minio.UploadInfo{}, fmt.Errorf("failed to upload file to MinIO: %w", err)
	}
	return info, nil
}

// GetFile retrieves a file from MinIO.
func (mc *MinioConnection) GetFile(ctx context.Context, objectName string) ([]byte, error) {
	obj, err := mc.Client.GetObject(ctx, mc.Bucket, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get object from MinIO: %w", err)
	}
	defer obj.Close()

	data, err := io.ReadAll(obj)
	if err != nil {
		return nil, fmt.Errorf("failed to read object from MinIO: %w", err)
	}

	return data, nil
}
