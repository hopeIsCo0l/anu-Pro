package storage

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	appcfg "github.com/hopeIsCo0l/anu-pro/internal/platform/config"
)

// Store is the S3-compatible object storage interface.
// Works with Cloudflare R2 (prod), MinIO (local), and standard AWS S3.
type Store interface {
	Upload(ctx context.Context, key, contentType string, body io.Reader, size int64) error
	Delete(ctx context.Context, key string) error
	PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error)
}

type s3Store struct {
	client  *s3.Client
	presign *s3.PresignClient
	bucket  string
}

func New(cfg *appcfg.Config) (Store, error) {
	sdkCfg, err := awscfg.LoadDefaultConfig(context.Background(),
		awscfg.WithRegion(cfg.StorageRegion),
		awscfg.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.StorageAccessKey, cfg.StorageSecretKey, "",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("storage: load sdk config: %w", err)
	}

	client := s3.NewFromConfig(sdkCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(cfg.StorageEndpoint)
		o.UsePathStyle = true // required for MinIO and R2
	})

	return &s3Store{
		client:  client,
		presign: s3.NewPresignClient(client),
		bucket:  cfg.StorageBucket,
	}, nil
}

func (s *s3Store) Upload(ctx context.Context, key, contentType string, body io.Reader, size int64) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(key),
		Body:          body,
		ContentType:   aws.String(contentType),
		ContentLength: &size,
	})
	if err != nil {
		return fmt.Errorf("storage: upload %s: %w", key, err)
	}
	return nil
}

func (s *s3Store) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("storage: delete %s: %w", key, err)
	}
	return nil
}

func (s *s3Store) PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error) {
	req, err := s.presign.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", fmt.Errorf("storage: presign %s: %w", key, err)
	}
	return req.URL, nil
}
