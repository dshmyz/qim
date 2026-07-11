package handler

import (
	"encoding/json"
	"testing"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/sqlite"
	"github.com/dshmyz/qim/qim-server/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// TestSystemConfig_FullRoundTrip 模拟前端真实保存→刷新回显的完整链路：
// 前端表单 → mapConfigFromFrontend → BatchUpdate 落库 → GetAllConfigs → mapConfigToFrontend → 前端回显。
// 用以定位"保存后刷新值恢复"到底发生在哪一步。
func TestSystemConfig_FullRoundTrip(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.SystemConfig{}))
	// 预置种子（与 app/init.go seedFileUploadConfig 一致）
	require.NoError(t, db.Create(&model.SystemConfig{
		ConfigKey: "file_upload:max_size", Value: "524288000", Type: "number",
	}).Error)

	cfgSvc := service.NewSystemConfigService(db)

	// === 1. 前端点保存，发出与 SystemConfig.vue handleSubmit 一致的 payload ===
	// 数字字段在 JSON 里是 float64；allowedFileTypes 已被 JSON.stringify 成字符串。
	frontendPayload := map[string]interface{}{
		"messageRecallTime":         float64(300), // 改成 300 秒
		"enableReadReceipt":         false,
		"enableAI":                  false,
		"maxFileSize":               float64(100), // 100MB
		"imageQuality":              float64(90),
		"enableRegistration":        false,
		"enable2FA":                 true,
		"enableFileUpload":          false,
		"allowedFileTypes":          `[".jpg",".png"]`,
		"rateLimitGlobalRate":       float64(600),
		"rateLimitGlobalWindow":     float64(120),
		"rateLimitLoginMaxAttempts": float64(8),
		"rateLimitLoginWindow":      float64(90),
		"rateLimitLoginBan":         float64(1800),
	}
	mapped := mapConfigFromFrontend(frontendPayload)
	t.Logf("mapConfigFromFrontend 输出: %s", toJSON(mapped))

	// === 2. 落库 ===
	require.NoError(t, cfgSvc.BatchUpdate(mapped))

	// 打印实际入库的每一行，定位 type/value 是否被写坏
	var rows []model.SystemConfig
	require.NoError(t, db.Order("config_key").Find(&rows).Error)
	t.Logf("DB 行:")
	for _, r := range rows {
		t.Logf("  key=%-32s type=%-8s value=%s", r.ConfigKey, r.Type, r.Value)
	}

	// === 3. 刷新读取（GetSystemConfig 的等价路径）===
	all, err := cfgSvc.GetAllConfigs()
	require.NoError(t, err)
	displayed := mapConfigToFrontend(all)
	t.Logf("回显给前端的 map: %s", toJSON(displayed))

	// === 4. 断言前端 Object.assign 后能拿到的值 ===
	assert.Equal(t, 300, displayed["messageRecallTime"], "messageRecallTime 应回显 300")
	assert.Equal(t, 100, displayed["maxFileSize"], "maxFileSize 应回显 100 (MB)")
	assert.Equal(t, false, displayed["enableReadReceipt"])
	assert.Equal(t, false, displayed["enableAI"])
	assert.Equal(t, false, displayed["enableRegistration"])
	assert.Equal(t, true, displayed["enable2FA"])
	assert.Equal(t, false, displayed["enableFileUpload"])
	assert.Equal(t, 90, displayed["imageQuality"])
	assert.Equal(t, 600, displayed["rateLimitGlobalRate"])
}

func toJSON(m map[string]interface{}) string {
	b, _ := json.Marshal(m)
	return string(b)
}
