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

// TestParseClientUA 从 Electron 默认 UA 解析版本/平台（不校验产品名）：
// 版本取 " Chrome/" 前紧邻 token 的 x.y.z；浏览器/curl 解析不出返回空版本。
func TestParseClientUA(t *testing.T) {
	cases := []struct {
		name         string
		ua           string
		wantVersion  string
		wantPlatform string
	}{
		{
			name:         "macOS Electron",
			ua:           "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) qim/1.0.0 Chrome/148.0.7778.218 Electron/42.3.3 Safari/537.36",
			wantVersion:  "1.0.0",
			wantPlatform: "macos",
		},
		{
			name:         "Windows Electron",
			ua:           "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) qim/2.0.38 Chrome/134.0.6998.205 Electron/42.3.3 Safari/537.36",
			wantVersion:  "2.0.38",
			wantPlatform: "windows",
		},
		{
			name:         "Linux Electron",
			ua:           "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) qim/2.0.38 Chrome/134.0.6998.205 Electron/42.3.3 Safari/537.36",
			wantVersion:  "2.0.38",
			wantPlatform: "linux",
		},
		{
			name:         "Win7 Electron",
			ua:           "Mozilla/5.0 (Windows NT 6.1; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) qim/1.0.0 Chrome/108.0.5359.215 Electron/22.3.27 Safari/537.36",
			wantVersion:  "1.0.0",
			wantPlatform: "windows",
		},
		{
			name:         "浏览器取不到客户端版本",
			ua:           "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/151.0.0.0 Safari/537.36",
			wantVersion:  "",
			wantPlatform: "windows",
		},
		{
			name:         "macOS Safari 浏览器",
			ua:           "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Safari/605.1.15",
			wantVersion:  "",
			wantPlatform: "macos",
		},
		{
			name:         "curl",
			ua:           "curl/8.7.1",
			wantVersion:  "",
			wantPlatform: "",
		},
		{
			name:         "空 UA",
			ua:           "",
			wantVersion:  "",
			wantPlatform: "",
		},
		{
			name:         "非三段版本号不认",
			ua:           "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) qim/2.0.39.1 Chrome/134.0.6998.205 Electron/42.3.3 Safari/537.36",
			wantVersion:  "",
			wantPlatform: "windows",
		},
		{
			name:         "缺少 Chrome 标记不认",
			ua:           "qim/2.0.39",
			wantVersion:  "",
			wantPlatform: "",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			gotVersion, gotPlatform := parseClientUA(c.ua)
			require.Equal(t, c.wantVersion, gotVersion)
			require.Equal(t, c.wantPlatform, gotPlatform)
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
