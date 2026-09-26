package storage

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestValidateKey(t *testing.T) {
	for _, key := range []string{"", " "} {
		if err := validateKey(key); err == nil {
			t.Fatalf("validateKey(%q) expected error", key)
		}
	}
	if err := validateKey("projects/123/image.png"); err != nil {
		t.Fatalf("validateKey() unexpected error: %v", err)
	}
}

func TestValidateExpiry(t *testing.T) {
	if err := validateExpiry(0); err == nil {
		t.Fatal("validateExpiry(0) expected error")
	}
	if err := validateExpiry(-time.Second); err == nil {
		t.Fatal("validateExpiry(-1s) expected error")
	}
}

func TestNewS3StorageRequiresBucket(t *testing.T) {
	_, err := NewS3Storage(context.Background(), S3Config{})
	if err == nil || !strings.Contains(err.Error(), "bucket") {
		t.Fatalf("expected bucket error, got %v", err)
	}
}
