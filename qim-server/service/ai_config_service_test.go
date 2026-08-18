package service

import (
	"testing"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/sqlite"
	"github.com/dshmyz/qim/qim-server/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupAIConfigTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)

	require.NoError(t, db.AutoMigrate(
		&model.AIConfig{},
	))
	return db
}

// 初始化加密密钥（每次测试前重置，避免依赖全局状态）
func initEncryptionKeyForTest(t *testing.T) {
	// utils.InitEncryptionKey 在测试环境用固定 key，这里只需确保 encryptionKey 非 nil
	// 通过直接调用一次确保初始化（测试环境下用空 ENCRYPTION_KEY 也能正常工作）
	utils.InitEncryptionKey("")
}

// UpdateConfig 传空 apiKey 时应保留原密钥，不覆盖为空
func TestUpdateConfig_EmptyAPIKey_PreservesOriginalKey(t *testing.T) {
	initEncryptionKeyForTest(t)
	db := setupAIConfigTestDB(t)
	svc := NewAIConfigService(db, ai.NewProviderFactory())

	// 先创建一条配置（provider 用 non-openai 避免 TestConnection 联网）
	original, err := svc.CreateConfig(1, "原配置", "baidu", "sk-original-key", "model-x", "https://api.baidu.com", "")
	require.NoError(t, err)
	require.NotEmpty(t, original.APIKeyEncrypted)

	// 编辑：apiKey 传空字符串（模拟前端编辑场景）
	updated, err := svc.UpdateConfig(1, original.ID, "改名", "baidu", "", "model-x", "https://api.baidu.com")
	require.NoError(t, err)

	// 原密钥应保留，不被空字符串覆盖
	assert.Equal(t, original.APIKeyEncrypted, updated.APIKeyEncrypted,
		"空 apiKey 时应保留原密钥，不应覆盖为空")
	assert.Equal(t, "改名", updated.ConfigName, "config_name 应更新")
}

// UpdateConfig 传新 apiKey 时应更新密钥
func TestUpdateConfig_NewAPIKey_UpdatesKey(t *testing.T) {
	initEncryptionKeyForTest(t)
	db := setupAIConfigTestDB(t)
	svc := NewAIConfigService(db, ai.NewProviderFactory())

	original, err := svc.CreateConfig(1, "原配置", "baidu", "sk-old-key", "model-x", "", "")
	require.NoError(t, err)

	updated, err := svc.UpdateConfig(1, original.ID, "原配置", "baidu", "sk-new-key", "model-x", "")
	require.NoError(t, err)

	// 密钥应更新（加密后值不同）
	assert.NotEqual(t, original.APIKeyEncrypted, updated.APIKeyEncrypted,
		"新 apiKey 时应更新密钥")
}

// UpdateConfig 不存在的 configID 应返回 ErrConfigNotFound
func TestUpdateConfig_NotFound(t *testing.T) {
	initEncryptionKeyForTest(t)
	db := setupAIConfigTestDB(t)
	svc := NewAIConfigService(db, ai.NewProviderFactory())

	_, err := svc.UpdateConfig(1, 9999, "配置", "baidu", "sk-key", "model-x", "")
	assert.ErrorIs(t, err, ErrConfigNotFound)
}

// UpdateConfig 应校验配置归属当前用户（不能改别人的配置）
func TestUpdateConfig_OwnershipCheck(t *testing.T) {
	initEncryptionKeyForTest(t)
	db := setupAIConfigTestDB(t)
	svc := NewAIConfigService(db, ai.NewProviderFactory())

	// 用户 1 创建配置
	original, err := svc.CreateConfig(1, "用户1的配置", "baidu", "sk-key", "model-x", "", "")
	require.NoError(t, err)

	// 用户 2 尝试修改用户 1 的配置 → 应返回 NotFound（不泄露存在性）
	_, err = svc.UpdateConfig(2, original.ID, "改名", "baidu", "", "model-x", "")
	assert.ErrorIs(t, err, ErrConfigNotFound)
}
