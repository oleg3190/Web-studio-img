package storage

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"
)

type ObjectInfo struct {
	Key         string
	Size        int64
	ContentType string
	ETag        string
}

type PutOptions struct {
	ContentType string
	Metadata    map[string]string
}

type PresignedURL struct {
	URL       string
	ExpiresAt time.Time
}

type StorageProvider interface {
	Put(ctx context.Context, key string, body io.Reader, size int64, options PutOptions) error
	Get(ctx context.Context, key string) (io.ReadCloser, ObjectInfo, error)
	Delete(ctx context.Context, key string) error
	PresignGet(ctx context.Context, key string, expiry time.Duration) (PresignedURL, error)
	PresignPut(ctx context.Context, key string, expiry time.Duration, contentType string) (PresignedURL, error)
}

func validateKey(key string) error {
	if key == "" || strings.TrimSpace(key) == "" {
		return fmt.Errorf("storage key must not be empty")
	}
	return nil
}

func validateExpiry(expiry time.Duration) error {
	if expiry <= 0 {
		return fmt.Errorf("presigned URL expiry must be positive")
	}
	return nil
}
