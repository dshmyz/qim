package storage

import (
	"context"
	"io"
	"time"

	"qim-server/config"
	"qim-server/service"
)

type S3Storage struct {
	svc *service.S3Service
	cfg config.S3StorageConfig
}

func NewS3Storage(svc *service.S3Service, cfg config.S3StorageConfig) *S3Storage {
	return &S3Storage{svc: svc, cfg: cfg}
}

func (s *S3Storage) Put(ctx context.Context, key string, data io.Reader, size int64, mime string) error {
	return s.svc.UploadFile(ctx, key, data, mime)
}

func (s *S3Storage) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	return s.svc.DownloadFile(ctx, key)
}

func (s *S3Storage) Delete(ctx context.Context, key string) error {
	return s.svc.DeleteFile(ctx, key)
}

func (s *S3Storage) Exists(ctx context.Context, key string) (bool, error) {
	return s.svc.FileExists(ctx, key)
}

func (s *S3Storage) URL(ctx context.Context, key string, expires time.Duration) (string, error) {
	if expires > 0 {
		return s.svc.GetPresignedURL(ctx, key, expires)
	}
	return "https://" + s.cfg.Bucket + "." + s.cfg.Endpoint + "/" + key, nil
}

func (s *S3Storage) Kind() string {
	return "s3"
}
