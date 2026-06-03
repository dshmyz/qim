package service

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/dshmyz/qim/qim-server/config"

	"github.com/aws/aws-sdk-go-v2/aws"
	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Service struct {
	client         *s3.Client
	transferClient *transfermanager.Client
	bucket         string
	cfg            config.S3StorageConfig
}

func NewS3Service(cfg config.S3StorageConfig) (*S3Service, error) {
	r2Resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL: cfg.Endpoint,
		}, nil
	})

	awsCfg, err := awscfg.LoadDefaultConfig(context.TODO(),
		awscfg.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, "")),
		awscfg.WithRegion(cfg.Region),
		awscfg.WithEndpointResolverWithOptions(r2Resolver),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load S3 config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	transferClient := transfermanager.New(client)

	return &S3Service{
		client:         client,
		transferClient: transferClient,
		bucket:         cfg.Bucket,
		cfg:            cfg,
	}, nil
}

func (s *S3Service) UploadFile(ctx context.Context, key string, data io.Reader, mimeType string) error {
	_, err := s.transferClient.UploadObject(ctx, &transfermanager.UploadObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        data,
		ContentType: aws.String(mimeType),
	})

	return err
}

func (s *S3Service) DownloadFile(ctx context.Context, key string) (io.ReadCloser, error) {
	output, err := s.transferClient.GetObject(ctx, &transfermanager.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}

	return io.NopCloser(output.Body), nil
}

func (s *S3Service) DeleteFile(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})

	return err
}

func (s *S3Service) GetPresignedURL(ctx context.Context, key string, expires time.Duration) (string, error) {
	presignClient := s3.NewPresignClient(s.client)

	req, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expires))

	if err != nil {
		return "", err
	}

	return req.URL, nil
}

func (s *S3Service) FileExists(ctx context.Context, key string) (bool, error) {
	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		return false, err
	}

	return true, nil
}
