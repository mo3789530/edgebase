package storage

import (
	"context"
	"log"

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
