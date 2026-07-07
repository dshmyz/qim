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

	// 启动时探测 bucket 可达性，凭证/端点/bucket 错误立即暴露，避免 di 静默降级到 local
	probeCtx, probeCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer probeCancel()
	if _, err := client.HeadBucket(probeCtx, &s3.HeadBucketInput{Bucket: aws.String(cfg.Bucket)}); err != nil {
		return nil, fmt.Errorf("S3 bucket 不可达（检查端点/凭证/bucket）: %w", err)
	}

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
	// 直接用 s3 client 读取，返回真正的 io.ReadCloser，调用方 Close 即可释放底层 HTTP 连接。
	// 不用 transferClient：其 GetObject 返回的 Body 非 ReadCloser，io.NopCloser 会使 Close 变成空操作，
	// 客户端中断下载时底层连接/goroutine 泄漏。
	output, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	return output.Body, nil
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
