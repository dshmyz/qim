package service

import (
	"testing"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseBotConfigModelSource 验证 BotConfig 能解析「模型来源」字段
// （use_system_config / user_config_id），与前端 CreateBotWizard 写入的 key 一致。
func TestParseBotConfigModelSource(t *testing.T) {
	// 未配置 user_config_id 的旧数据：UseSystemConfig 为 Go 零值 false，但 UserConfigID 为 nil，
	// 回复路径以「!UseSystemConfig && UserConfigID != nil」双重条件判断自定义，nil 时安全回退系统默认。
	cfg := ParseBotConfig(`{"mode":"internal_ai"}`)
	assert.False(t, cfg.UseSystemConfig)
	assert.Nil(t, cfg.UserConfigID)

	// 系统默认（在存储时显式写 true）
	cfg = ParseBotConfig(`{"mode":"internal_ai","use_system_config":true}`)
	assert.True(t, cfg.UseSystemConfig)

	// 自定义配置
	cfg = ParseBotConfig(`{"mode":"internal_ai","use_system_config":false,"user_config_id":7}`)
	assert.False(t, cfg.UseSystemConfig)
	require.NotNil(t, cfg.UserConfigID)
	assert.Equal(t, uint(7), *cfg.UserConfigID)
}

// TestResolveUserAIConfigProvider 验证 bot/分身共用的自选 provider 解析：
// 按 configID+userID 取配置，校验启用态与归属，构建出 provider 名 + ProviderConfig。
func TestResolveUserAIConfigProvider(t *testing.T) {
	utils.InitEncryptionKey("")
	db := setupServiceTestDB(t)
	require.NoError(t, db.Migrator().CreateTable(&model.AIConfig{}))

	encKey, err := utils.EncryptAPIKey("sk-test-123")
	require.NoError(t, err)

	cfg := &model.AIConfig{
		UserID:          1,
		Provider:        "openai",
		ConfigName:      "我的配置",
		ModelName:       "gpt-4o",
		BaseURL:         "https://api.openai.com",
		APIKeyEncrypted: encKey,
		AIEnabled:       true,
		MaxTokens:       2048,
		Temperature:     0.5,
	}
	require.NoError(t, db.Create(cfg).Error)

	// 1) 有效配置：解析返回 provider 名 + 解密后的 key + 有效参数（零值不透传）
	got := resolveUserAIConfigProvider(db, 1, cfg.ID)
	require.NotNil(t, got)
	assert.Equal(t, "openai", got.ProviderName)
	assert.Equal(t, "sk-test-123", got.Config.APIKey)
	assert.Equal(t, "gpt-4o", got.Config.Model)
	assert.Equal(t, "https://api.openai.com", got.Config.BaseURL)
	assert.Equal(t, 2048, got.Config.ExtraParams["max_tokens"], "正向值应透传")
	assert.Equal(t, 0.5, got.Config.ExtraParams["temperature"])
	assert.NotContains(t, got.Config.ExtraParams, "missing")

	// 2) 配置被他人所有 → 回退（nil）
	assert.Nil(t, resolveUserAIConfigProvider(db, 999, cfg.ID))

	// 3) 配置不存在 → 回退（nil）
	assert.Nil(t, resolveUserAIConfigProvider(db, 1, 999999))

	// 4) 已禁用 → 回退（nil）
	require.NoError(t, db.Model(cfg).Update("ai_enabled", false).Error)
	assert.Nil(t, resolveUserAIConfigProvider(db, 1, cfg.ID))
}
