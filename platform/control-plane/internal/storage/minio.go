package storage

import (
	"bytes"
	"context"
	"log"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinIOClient struct {
	Client *minio.Client
	Bucket string
}

func Init(endpoint, accessKey, secretKey, bucket string) (*MinIOClient, error) {
	// Initialize minio client object.
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: false, // TODO: Make configurable
	})
	if err != nil {
		return nil, err
	}

	// Check if bucket exists
	ctx := context.Background()
	exists, err := minioClient.BucketExists(ctx, bucket)
	if err != nil {
		return nil, err
	}

	if !exists {
		err = minioClient.MakeBucket(ctx, bucket, minio.MakeBucketOptions{})
		if err != nil {
			return nil, err
		}
		log.Printf("Bucket %s created", bucket)
	}

	log.Println("Connected to MinIO")
	return &MinIOClient{
		Client: minioClient,
		Bucket: bucket,
	}, nil
}

func (m *MinIOClient) Upload(ctx context.Context, objectName string, data []byte, contentType string) error {
	_, err := m.Client.PutObject(ctx, m.Bucket, objectName, bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{
		ContentType: contentType,
	})
	return err
}

func (m *MinIOClient) GetPresignedURL(ctx context.Context, objectName string, expiry time.Duration) (string, error) {
	url, err := m.Client.PresignedGetObject(ctx, m.Bucket, objectName, expiry, nil)
	if err != nil {
		return "", err
	}
	return url.String(), nil
}
