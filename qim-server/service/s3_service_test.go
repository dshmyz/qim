package service

import (
	"testing"

	"github.com/dshmyz/qim/qim-server/config"

	"github.com/stretchr/testify/assert"
)

// TestNewS3Service_HeadBucketFailsOnUnreachableEndpoint 验证 S3 端点不可达时
// NewS3Service 启动即失败（HeadBucket 探测），而非创建 client 成功、
// 等到首次 Put/Get 才暴露错误——后者会导致 di 静默降级到 local 而运维无感。
func TestNewS3Service_HeadBucketFailsOnUnreachableEndpoint(t *testing.T) {
	_, err := NewS3Service(config.S3StorageConfig{
		Endpoint:  "http://127.0.0.1:1", // 不可达端口，连接立即拒绝
		AccessKey: "x",
		SecretKey: "x",
		Bucket:    "nonexistent",
		Region:    "us-east-1",
		UseSSL:    false,
	})
	assert.Error(t, err, "不可达端点应 HeadBucket 失败，NewS3Service 应返回 error")
}
