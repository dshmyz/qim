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

// UpsertConfig 幂等 upsert：首次创建，再次写入更新同一行（不新增）。
func TestSystemConfigUpsertConfigIdempotent(t *testing.T) {
	db := setupServiceTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.SystemConfig{}))
	svc := NewSystemConfigService(db)

	require.NoError(t, svc.UpsertConfig("ai.ui.suggested_prompts", `["a","b"]`, "json", "推荐提示词"))
	cfg, err := svc.GetConfig("ai.ui.suggested_prompts")
	require.NoError(t, err)
	assert.Equal(t, `["a","b"]`, cfg.Value)
	assert.Equal(t, "json", cfg.Type)

	// 覆盖写入：值更新，行数不变（唯一键存在则更新而非新建）
	require.NoError(t, svc.UpsertConfig("ai.ui.suggested_prompts", `["a","b","c"]`, "json", "推荐提示词"))
	cfg, err = svc.GetConfig("ai.ui.suggested_prompts")
	require.NoError(t, err)
	assert.Equal(t, `["a","b","c"]`, cfg.Value)

	var count int64
	db.Model(&model.SystemConfig{}).Count(&count)
	assert.EqualValues(t, 1, count, "upsert 应更新既有行而非新增")

	// 未配置键 → GetConfig 返回 ErrRecordNotFound
	_, err = svc.GetConfig("not_configured")
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

// TestPublicConfigs_IncludesClientUpdateBaseURL 验证客户端更新服务器地址在公开配置白名单内。
// 客户端启动/登录时拉取公开配置并据此校正更新地址，若不在白名单内则该机制静默失效。
func TestPublicConfigs_IncludesClientUpdateBaseURL(t *testing.T) {
	found := false
	for _, k := range publicConfigKeys {
		if k == "client:update_base_url" {
			found = true
			break
		}
	}
	assert.True(t, found, "publicConfigKeys 应包含 client:update_base_url")

	db := setupSystemConfigTestDB(t)
	svc := NewSystemConfigService(db)
	require.NoError(t, svc.UpsertConfig("client:update_base_url", "https://updates.example.com", "string", "客户端更新服务器地址"))

	cfg, err := svc.GetPublicConfigs()
	require.NoError(t, err)
	assert.Equal(t, "https://updates.example.com", cfg["client:update_base_url"], "公开配置应透出 client:update_base_url 的值")
}
