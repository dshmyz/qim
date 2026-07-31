package service

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/sqlite"

	"gorm.io/gorm"
)

func newVersionServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	// 用文件型 SQLite 而非 :memory:，避免连接池跨连接看不到同一内存库
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "version-test.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&model.ClientVersion{}); err != nil {
		t.Fatalf("AutoMigrate 失败: %v", err)
	}
	return db
}

// TestCreate_RolloutPercentageZero_PersistedAsZero 验证管理员显式设置 0% 灰度时，
// DB 真正持久化为 0 而非被 GORM default:100 吞成 100。
// 复现审查发现的安全事故级 bug：前端显示 0%，后端实际全量发布。
func TestCreate_RolloutPercentageZero_PersistedAsZero(t *testing.T) {
	db := newVersionServiceTestDB(t)
	svc := NewVersionService(db, nil)

	zero := 0
	version, err := svc.Create(CreateVersionInput{
		Version:           "2.1.0",
		Platform:          "windows",
		DownloadURL:       "https://example.com/QIM-2.1.0.exe",
		Sha512:            "abc",
		FileSize:          1024,
		RolloutPercentage: &zero,
	})
	if err != nil {
		t.Fatalf("Create 失败: %v", err)
	}

	// 直接用原生 SQL 读取持久化值，绕过 Go 层的字段语义
	var stored int
	if err := db.Raw("SELECT rollout_percentage FROM client_versions WHERE id = ?", version.ID).Scan(&stored).Error; err != nil {
		t.Fatalf("读取持久化值失败: %v", err)
	}
	if stored != 0 {
		t.Fatalf("期望 rollout_percentage 持久化为 0（管理员止血），实际为 %d（GORM default:100 吞掉了零值）", stored)
	}
}

// TestToggleStatus_ReturnsUpdatedVersion 验证 ToggleStatus 返回更新后的版本对象，
// 使 handler 无需再二次 GetByID（原实现忽略二次 GetByID 的错误，并发删除时 nil 解引用 panic）。
func TestToggleStatus_ReturnsUpdatedVersion(t *testing.T) {
	db := newVersionServiceTestDB(t)
	svc := NewVersionService(db, nil)

	created, err := svc.Create(CreateVersionInput{
		Version:     "2.1.0",
		Platform:    "windows",
		DownloadURL: "https://example.com/QIM-2.1.0.exe",
		Sha512:      "abc",
		FileSize:    1024,
	})
	if err != nil {
		t.Fatalf("Create 失败: %v", err)
	}

	updated, err := svc.ToggleStatus(created.ID, false)
	if err != nil {
		t.Fatalf("ToggleStatus 禁用失败: %v", err)
	}
	if updated == nil {
		t.Fatal("ToggleStatus 应返回更新后的版本，实际返回 nil（handler 仍需二次 GetByID，并发删除时会 nil 解引用 panic）")
	}
	if updated.Enabled != false {
		t.Fatalf("期望禁用后 Enabled=false，实际 %v", updated.Enabled)
	}

	updated, err = svc.ToggleStatus(created.ID, true)
	if err != nil {
		t.Fatalf("ToggleStatus 启用失败: %v", err)
	}
	if updated == nil || updated.Enabled != true {
		t.Fatalf("期望启用后 Enabled=true，实际 %v", updated)
	}
}

// TestToggleStatus_MissingVersionReturnsNotFound 验证不存在的版本返回 ErrVersionNotFound，
// handler 据此返回 404 而非 panic。
func TestToggleStatus_MissingVersionReturnsNotFound(t *testing.T) {
	db := newVersionServiceTestDB(t)
	svc := NewVersionService(db, nil)

	_, err := svc.ToggleStatus(99999, true)
	if !errors.Is(err, ErrVersionNotFound) {
		t.Fatalf("期望 ErrVersionNotFound，实际 %v", err)
	}
}

func TestCreateCLI_RequiresSha256(t *testing.T) {
	db := newVersionServiceTestDB(t)
	svc := NewVersionService(db, nil)

	_, err := svc.Create(CreateVersionInput{
		Version:     "1.2.0",
		Platform:    "darwin-arm64",
		AppType:     "cli",
		Os:          "darwin",
		Arch:        "arm64",
		DownloadURL: "/api/v1/public/files/1/download",
	})
	if !errors.Is(err, ErrMissingSha256) {
		t.Fatalf("期望 ErrMissingSha256，实际 %v", err)
	}
}

func TestCreateCLI_DerivesPlatformFromOSAndArch(t *testing.T) {
	db := newVersionServiceTestDB(t)
	svc := NewVersionService(db, nil)

	_, err := svc.Create(CreateVersionInput{
		Version:     "1.2.0",
		AppType:     "cli",
		Os:          "darwin",
		Arch:        "arm64",
		DownloadURL: "/api/v1/public/files/1/download",
		Sha256:      "darwin-sha",
		FileSize:    100,
	})
	if err != nil {
		t.Fatalf("Create 失败: %v", err)
	}

	latest, err := svc.GetLatestCLI("darwin", "arm64")
	if err != nil {
		t.Fatalf("GetLatestCLI 失败: %v", err)
	}
	if latest == nil {
		t.Fatal("期望通过 os/arch 查到刚创建的 CLI 版本，实际为 nil")
	}
	if latest.Platform != "darwin-arm64" {
		t.Fatalf("期望 platform=darwin-arm64，实际 %q", latest.Platform)
	}
}

func TestUpdateCLI_UpdatesDownloadURLAndSha256(t *testing.T) {
	db := newVersionServiceTestDB(t)
	svc := NewVersionService(db, nil)

	created, err := svc.Create(CreateVersionInput{
		Version:     "1.2.0",
		Platform:    "darwin-arm64",
		AppType:     "cli",
		Os:          "darwin",
		Arch:        "arm64",
		DownloadURL: "/api/v1/public/files/1/download",
		Sha256:      "old-sha",
		FileSize:    100,
	})
	if err != nil {
		t.Fatalf("Create 失败: %v", err)
	}

	newURL := "/api/v1/public/files/2/download"
	newSha := "new-sha"
	updated, err := svc.Update(created.ID, UpdateVersionInput{
		DownloadURL: &newURL,
		Sha256:      &newSha,
	})
	if err != nil {
		t.Fatalf("Update 失败: %v", err)
	}
	if updated.DownloadURL != newURL {
		t.Fatalf("期望 download_url=%q，实际 %q", newURL, updated.DownloadURL)
	}
	if updated.Sha256 != newSha {
		t.Fatalf("期望 sha256=%q，实际 %q", newSha, updated.Sha256)
	}
}

func TestGetLatestCLIVersion_WindowsSha256KeyIncludesExe(t *testing.T) {
	db := newVersionServiceTestDB(t)
	svc := NewVersionService(db, nil)

	_, err := svc.Create(CreateVersionInput{
		Version:     "1.2.0",
		Platform:    "windows-amd64",
		AppType:     "cli",
		Os:          "windows",
		Arch:        "amd64",
		DownloadURL: "/api/v1/public/files/1/download",
		Sha256:      "win-sha",
		FileSize:    100,
	})
	if err != nil {
		t.Fatalf("Create 失败: %v", err)
	}

	_, shaMap, err := svc.GetLatestCLIVersion()
	if err != nil {
		t.Fatalf("GetLatestCLIVersion 失败: %v", err)
	}
	if shaMap["qim-windows-amd64.exe"] != "win-sha" {
		t.Fatalf("期望 Windows hash key 包含 .exe，实际 map=%v", shaMap)
	}
}
