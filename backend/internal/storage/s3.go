package storage

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Config struct {
	Endpoint     string
	Region       string
	Bucket       string
	AccessKey    string
	SecretKey    string
	UsePathStyle bool
}

type S3Storage struct {
	bucket  string
	client  *s3.Client
	presign *s3.PresignClient
	uploader *manager.Uploader
}

func NewS3Storage(ctx context.Context, cfg S3Config) (*S3Storage, error) {
	if cfg.Bucket == "" {
		return nil, fmt.Errorf("S3 bucket must not be empty")
	}
	if cfg.Region == "" {
		cfg.Region = "ru-central1"
	}

	awsCfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(cfg.Region),
		config.WithCredentialsProvider(aws.NewCredentialsCache(staticCredentialsProvider{
			accessKey: cfg.AccessKey,
			secretKey: cfg.SecretKey,
		})),
	)
	if err != nil {
		return nil, fmt.Errorf("load S3 config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(options *s3.Options) {
		options.UsePathStyle = cfg.UsePathStyle
		if cfg.Endpoint != "" {
			options.BaseEndpoint = aws.String(cfg.Endpoint)
		}
	})
	return &S3Storage{
		bucket:  cfg.Bucket,
		client:  client,
		presign: s3.NewPresignClient(client),
		uploader: manager.NewUploader(client),
	}, nil
}

func (s *S3Storage) Put(ctx context.Context, key string, body io.Reader, size int64, options PutOptions) error {
	if err := validateKey(key); err != nil {
		return err
	}
	if body == nil {
		return fmt.Errorf("storage body must not be nil")
	}
	input := &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        body,
		ContentType: aws.String(options.ContentType),
		Metadata:    options.Metadata,
	}
	if size >= 0 {
		input.ContentLength = aws.Int64(size)
	}
	if _, err := s.uploader.Upload(ctx, input); err != nil {
		return fmt.Errorf("put object %q: %w", key, err)
	}
	return nil
}

func (s *S3Storage) Get(ctx context.Context, key string) (io.ReadCloser, ObjectInfo, error) {
	if err := validateKey(key); err != nil {
		return nil, ObjectInfo{}, err
	}
	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, ObjectInfo{}, fmt.Errorf("get object %q: %w", key, err)
	}
	return result.Body, ObjectInfo{
		Key:         key,
		Size:        aws.ToInt64(result.ContentLength),
		ContentType: aws.ToString(result.ContentType),
		ETag:        aws.ToString(result.ETag),
	}, nil
}

func (s *S3Storage) Delete(ctx context.Context, key string) error {
	if err := validateKey(key); err != nil {
		return err
	}
	if _, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}); err != nil {
		return fmt.Errorf("delete object %q: %w", key, err)
	}
	return nil
}

func (s *S3Storage) PresignGet(ctx context.Context, key string, expiry time.Duration) (PresignedURL, error) {
	if err := validateKey(key); err != nil {
		return PresignedURL{}, err
	}
	if err := validateExpiry(expiry); err != nil {
		return PresignedURL{}, err
	}
	result, err := s.presign.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}, func(options *s3.PresignOptions) {
		options.Expires = expiry
	})
	if err != nil {
		return PresignedURL{}, fmt.Errorf("presign get %q: %w", key, err)
	}
	return PresignedURL{URL: result.URL, ExpiresAt: time.Now().Add(expiry)}, nil
}

func (s *S3Storage) PresignPut(ctx context.Context, key string, expiry time.Duration, contentType string) (PresignedURL, error) {
	if err := validateKey(key); err != nil {
		return PresignedURL{}, err
	}
	if err := validateExpiry(expiry); err != nil {
		return PresignedURL{}, err
	}
	result, err := s.presign.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}, func(options *s3.PresignOptions) {
		options.Expires = expiry
	})
	if err != nil {
		return PresignedURL{}, fmt.Errorf("presign put %q: %w", key, err)
	}
	return PresignedURL{URL: result.URL, ExpiresAt: time.Now().Add(expiry)}, nil
}

type staticCredentialsProvider struct {
	accessKey string
	secretKey string
}

func (p staticCredentialsProvider) Retrieve(context.Context) (aws.Credentials, error) {
	return aws.Credentials{
		AccessKeyID:     p.accessKey,
		SecretAccessKey: p.secretKey,
		Source:          "web-studio-static",
	}, nil
}
