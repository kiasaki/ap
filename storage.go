package ap

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Storage interface {
	Write(ctx context.Context, key string, data []byte, contentType string) error
	Read(ctx context.Context, key string) ([]byte, error)
}

type StorageLocal struct {
	dir string
}

func NewStorageLocal(dir string) Storage {
	return &StorageLocal{dir: dir}
}

func (s *StorageLocal) Write(_ context.Context, key string, data []byte, _ string) error {
	cleanKey, err := storageKey(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(s.dir, cleanKey), data, 0o644)
}

func (s *StorageLocal) Read(_ context.Context, key string) ([]byte, error) {
	cleanKey, err := storageKey(key)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(filepath.Join(s.dir, cleanKey))
}

type StorageS3 struct {
	client *s3.Client
	bucket string
	prefix string
}

func NewStorageS3(bucket, clientID, secretID, region string) Storage {
	Assert(bucket != "", "bucket is required for s3 storage")
	Assert(clientID != "", "clientID is required for s3 storage")
	Assert(secretID != "", "secretID is required for s3 storage")
	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(Or(region, "us-east-1")),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(clientID, secretID, "")),
	)
	Checkm(err, "load aws config")
	return &StorageS3{
		client: s3.NewFromConfig(cfg),
		bucket: strings.TrimSpace(bucket),
		prefix: "i",
	}
}

func (s *StorageS3) Write(ctx context.Context, key string, data []byte, contentType string) error {
	cleanKey, err := storageKey(key)
	if err != nil {
		return err
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      &s.bucket,
		Key:         StringPtr(s.key(cleanKey)),
		Body:        bytes.NewReader(data),
		ContentType: &contentType,
	})
	return err
}

func (s *StorageS3) Read(ctx context.Context, key string) ([]byte, error) {
	cleanKey, err := storageKey(key)
	if err != nil {
		return nil, err
	}
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &s.bucket,
		Key:    StringPtr(s.key(cleanKey)),
	})
	if err != nil {
		return nil, err
	}
	defer out.Body.Close()
	return io.ReadAll(out.Body)
}

func (s *StorageS3) key(key string) string {
	if s.prefix == "" {
		return key
	}
	return s.prefix + "/" + key
}

func storageKey(key string) (string, error) {
	clean := filepath.Base(strings.TrimSpace(key))
	if clean == "." || clean == "" {
		return "", fmt.Errorf("invalid storage key")
	}
	return clean, nil
}

func StringPtr(v string) *string {
	return &v
}
