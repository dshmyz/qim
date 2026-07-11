package service

import (
	"testing"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupSystemConfigTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.SystemConfig{}))
	return db
}

// TestBatchUpdate_RoundTripInt64 验证 int64 数值（如 maxFileSize 转字节）不会被误存为 string。
// 回归：此前 BatchUpdate 只认 float64，导致 file_upload:max_size 被写成 type=string，
// 读取时 mapConfigToFrontend 的 v.(int) 断言失败，前端刷新后回退到默认 50。
func TestBatchUpdate_RoundTripInt64(t *testing.T) {
	db := setupSystemConfigTestDB(t)
	svc := NewSystemConfigService(db)

	const bytes = int64(104857600) // 100MB，模拟 mapConfigFromFrontend 转出的 int64
	require.NoError(t, svc.BatchUpdate(map[string]interface{}{
		"file_upload:max_size": bytes,
	}))

	cfg, err := svc.GetConfig("file_upload:max_size")
	require.NoError(t, err)
	assert.Equal(t, "number", cfg.Type, "int64 应识别为 number，而非 string")
	assert.Equal(t, "104857600", cfg.Value)

	all, err := svc.GetAllConfigs()
	require.NoError(t, err)
	n, ok := all["file_upload:max_size"].(int)
	require.True(t, ok, "读回应为 int，实际类型 %T", all["file_upload:max_size"])
	assert.Equal(t, 104857600, n)
}
