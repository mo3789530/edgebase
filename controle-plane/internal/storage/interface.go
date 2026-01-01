package storage

import (
	"context"
	"time"
)

type Client interface {
	Upload(ctx context.Context, objectName string, data []byte, contentType string) error
	GetPresignedURL(ctx context.Context, objectName string, expiry time.Duration) (string, error)
	Download(ctx context.Context, objectName string) ([]byte, error)
}
