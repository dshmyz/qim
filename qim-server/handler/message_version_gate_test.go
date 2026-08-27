package handler

import (
	"testing"

	"github.com/dshmyz/qim/qim-server/di"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/sqlite"
	"github.com/dshmyz/qim/qim-server/service"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// TestClientSendBlocked 门槛判断纯函数：
// - 门槛未配置/格式非法 → 不拦截
// - 客户端未上报版本/格式非法 → 拦截（无法证明达门槛）
// - 客户端版本低于/等于/高于门槛
func TestClientSendBlocked(t *testing.T) {
	cases := []struct {
		name         string
		clientV, min string
		want         bool
	}{
		{"门槛为空不拦截", "2.0.30", "", false},
		{"门槛格式非法不拦截", "2.0.30", "not-a-version", false},
		{"客户端未上报版本拦截", "", "2.0.30", true},
		{"客户端格式非法拦截", "2.0", "2.0.30", true},
		{"低于门槛拦截", "2.0.29", "2.0.30", true},
		{"等于门槛放行", "2.0.30", "2.0.30", false},
		{"高于门槛放行", "2.0.31", "2.0.30", false},
		{"更高版本放行", "2.1.0", "2.0.30", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			require.Equal(t, c.want, clientSendBlocked(c.clientV, c.min))
		})
	}
}

// TestMinSendVersionPlatform 平台专属门槛解析：Windows 专属覆盖通用、未配置平台回退通用、全空返回空。
func TestMinSendVersionPlatform(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.SystemConfig{}))
	di.GlobalContainer = &di.Container{SystemConfigService: service.NewSystemConfigService(db)}
	invalidateMinSendVersionCache()

	// 未配置 → 空（不限制）
	require.Equal(t, "", minSendVersion("windows"))

	// 仅通用门槛 → 各平台回退通用
	require.NoError(t, db.Create(&model.SystemConfig{
		ConfigKey: "client:min_send_version", Value: "2.0.30", Type: "string",
	}).Error)
	invalidateMinSendVersionCache()
	require.Equal(t, "2.0.30", minSendVersion("windows"))
	require.Equal(t, "2.0.30", minSendVersion("macos"))
	require.Equal(t, "2.0.30", minSendVersion(""))
	// 未归一化平台名也可命中（后端统一 NormalizePlatform）
	require.Equal(t, "2.0.30", minSendVersion("Windows"))

	// Windows 专属覆盖通用；macOS 仍回退通用
	require.NoError(t, db.Create(&model.SystemConfig{
		ConfigKey: "client:min_send_version:windows", Value: "2.0.35", Type: "string",
	}).Error)
	invalidateMinSendVersionCache()
	require.Equal(t, "2.0.35", minSendVersion("windows"))
	require.Equal(t, "2.0.30", minSendVersion("macos"))

	// 清除通用门槛后：Windows 仍用专属，macOS 变回空（不限制）
	require.NoError(t, db.Where("config_key = ?", "client:min_send_version").Delete(&model.SystemConfig{}).Error)
	invalidateMinSendVersionCache()
	require.Equal(t, "2.0.35", minSendVersion("windows"))
	require.Equal(t, "", minSendVersion("macos"))
}
