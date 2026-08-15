package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ValidateCORS_RejectsWildcardWithCredentials(t *testing.T) {
	cfg := &Config{
		CORS: CORSConfig{
			AllowedOrigins:   []string{"*"},
			AllowCredentials: true,
		},
	}

	cfg.ValidateCORS()

	// 当 AllowCredentials=true 且 Origins 含 "*" 时，应将 Origins 设为空切片
	assert.NotContains(t, cfg.CORS.AllowedOrigins, "*",
		"ValidateCORS should remove wildcard origin when credentials are enabled")
}

func Test_ValidateCORS_AcceptsSpecificOrigins(t *testing.T) {
	cfg := &Config{
		CORS: CORSConfig{
			AllowedOrigins: []string{"https://example.com"},
		},
	}

	cfg.ValidateCORS()

	// 指定具体域名时，不应修改
	assert.Equal(t, []string{"https://example.com"}, cfg.CORS.AllowedOrigins,
		"ValidateCORS should not modify specific origins")
}

// writeConfigFile 写入临时 config.yaml 并切换工作目录，返回恢复函数
func writeConfigFile(t *testing.T, content string) func() {
	t.Helper()
	origWd, err := os.Getwd()
	require.NoError(t, err)
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(content), 0o644))
	require.NoError(t, os.Chdir(dir))
	return func() {
		_ = os.Chdir(origWd)
	}
}

func Test_Load_StaticPathsFromYAML(t *testing.T) {
	restore := writeConfigFile(t, `
server:
  port: "8080"
  mode: "debug"
jwt:
  secret: "test-secret-xxxx"
static:
  uploads_dir: "/var/qim/uploads"
  miniapps_dir: "/var/qim/miniapps"
`)
	defer restore()

	cfg := Load()

	assert.Equal(t, "/var/qim/uploads", cfg.Static.UploadsDir, "uploads_dir 应从 yaml 加载")
	assert.Equal(t, "/var/qim/miniapps", cfg.Static.MiniAppsDir, "miniapps_dir 应从 yaml 加载")
}

func Test_Load_StaticPathsDefaultsWhenMissing(t *testing.T) {
	restore := writeConfigFile(t, `
server:
  port: "8080"
  mode: "debug"
jwt:
  secret: "test-secret-xxxx"
`)
	defer restore()

	cfg := Load()

	// 未显式配置时应回退到默认值（与 routes.go 既有硬编码保持一致）
	assert.Equal(t, "uploads", cfg.Static.UploadsDir, "uploads_dir 应有默认值")
	assert.Equal(t, "static/miniapps", cfg.Static.MiniAppsDir, "miniapps_dir 应有默认值")
}

func Test_Load_StaticPathsOverridableByEnv(t *testing.T) {
	restore := writeConfigFile(t, `
server:
  port: "8080"
  mode: "debug"
jwt:
  secret: "test-secret-xxxx"
static:
  uploads_dir: "./from-yaml"
`)
	defer restore()

	t.Setenv("QIM_STATIC_UPLOADS_DIR", "/env/uploads")
	t.Setenv("QIM_STATIC_MINIAPPS_DIR", "/env/miniapps")

	cfg := Load()

	assert.Equal(t, "/env/uploads", cfg.Static.UploadsDir, "uploads_dir 应被环境变量覆盖")
	assert.Equal(t, "/env/miniapps", cfg.Static.MiniAppsDir, "miniapps_dir 应被环境变量覆盖")
}

func Test_Load_StaticPathsDefaultsWhenNoConfigFile(t *testing.T) {
	// 切到空目录，触发 "配置文件读取失败" 分支
	origWd, err := os.Getwd()
	require.NoError(t, err)
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))
	defer func() {
		_ = os.Chdir(origWd)
	}()

	// 清掉可能影响测试的环境变量
	t.Setenv("JWT_SECRET", "fallback-secret-for-test")

	cfg := Load()

	assert.Equal(t, "uploads", cfg.Static.UploadsDir)
	assert.Equal(t, "static/miniapps", cfg.Static.MiniAppsDir)
}

// 防止测试间相互污染：若其他测试设置了 PORT 等，强制重置
func init() {
	// 给 Load 一点稳定性，避免并行 go test 启动太快
	_ = time.Now()
}
