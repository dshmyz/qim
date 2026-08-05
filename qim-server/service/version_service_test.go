package service

import (
	"errors"
	"fmt"
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

	latest, err := svc.GetLatestCLI("darwin", "arm64", "")
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

// TestGetLatestCLI_ProductIsolation 验证 cli 与 mcp 产物互不干扰：
// 两产物在同平台可各存一份，且需按 product 精确命中。
func TestGetLatestCLI_ProductIsolation(t *testing.T) {
	db := newVersionServiceTestDB(t)
	svc := NewVersionService(db, nil)

	for _, c := range []struct {
		version string
		appType string
		os      string
		arch    string
	}{
		{"1.0.0", "cli", "darwin", "arm64"},
		{"1.0.0", "mcp", "darwin", "arm64"},
		{"2.0.0", "mcp", "darwin", "arm64"},
	} {
		if _, err := svc.Create(CreateVersionInput{
			Version:     c.version,
			AppType:     c.appType,
			Os:          c.os,
			Arch:        c.arch,
			DownloadURL: "/api/v1/public/files/1/download",
			Sha256:      "sha-" + c.appType + "-" + c.version,
			FileSize:    100,
		}); err != nil {
			t.Fatalf("Create(%s %s) 失败: %v", c.appType, c.version, err)
		}
	}

	// 默认 product（cli）命中 app_type=cli，且不受 mcp 影响
	cli, err := svc.GetLatestCLI("darwin", "arm64", "")
	if err != nil {
		t.Fatalf("GetLatestCLI 失败: %v", err)
	}
	if cli == nil || cli.Version != "1.0.0" {
		t.Fatalf("期望 cli 命中 1.0.0，实际 %+v", cli)
	}
	if cli.AppType != "cli" {
		t.Fatalf("期望 cli app_type=cli，实际 %q", cli.AppType)
	}

	// mcp 命中 app_type=mcp 的最新版本 2.0.0
	mcp, err := svc.GetLatestCLI("darwin", "arm64", "mcp")
	if err != nil {
		t.Fatalf("GetLatestCLI(mcp) 失败: %v", err)
	}
	if mcp == nil || mcp.Version != "2.0.0" {
		t.Fatalf("期望 mcp 命中 2.0.0，实际 %+v", mcp)
	}
	if mcp.AppType != "mcp" {
		t.Fatalf("期望 mcp app_type=mcp，实际 %q", mcp.AppType)
	}

	// SHA256 map 的 key：测试库无 File 记录 → 回退规则名 "{os}-{arch}"（不含品牌前缀）
	_, cliSha, err := svc.GetLatestCLIVersion("")
	if err != nil {
		t.Fatalf("GetLatestCLIVersion 失败: %v", err)
	}
	if _, ok := cliSha["darwin-arm64"]; !ok {
		t.Fatalf("期望 cli sha256 map 含 darwin-arm64，实际 %v", cliSha)
	}

	_, mcpSha, err := svc.GetLatestCLIVersion("mcp")
	if err != nil {
		t.Fatalf("GetLatestCLIVersion(mcp) 失败: %v", err)
	}
	if v, ok := mcpSha["darwin-arm64"]; !ok || v == "" {
		t.Fatalf("期望 mcp sha256 map 含 darwin-arm64，实际 %v", mcpSha)
	}
}

// TestDownloadFilename_UsesUploadedName 验证下载文件名取上传时的 OriginalName，
// 查不到文件记录时才回退规则名 "{os}-{arch}"（不含品牌前缀）。
func TestDownloadFilename_UsesUploadedName(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "version-test.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&model.ClientVersion{}, &model.File{}); err != nil {
		t.Fatalf("AutoMigrate 失败: %v", err)
	}
	svc := NewVersionService(db, nil)

	// 有文件记录：文件名取管理员上传原名
	file := model.File{Name: "mcp-darwin-arm64", OriginalName: "my-mcp-server-darwin-arm64", StoragePath: "/x", Size: 100}
	if err := db.Create(&file).Error; err != nil {
		t.Fatalf("创建 File 失败: %v", err)
	}
	withFile, err := svc.Create(CreateVersionInput{
		Version:     "1.0.0",
		AppType:     "mcp",
		Os:          "darwin",
		Arch:        "arm64",
		DownloadURL: fmt.Sprintf("/api/v1/public/files/%d/download", file.ID),
		Sha256:      "abc",
		FileSize:    100,
	})
	if err != nil {
		t.Fatalf("Create(withFile) 失败: %v", err)
	}
	if got := svc.DownloadFilename(withFile); got != "my-mcp-server-darwin-arm64" {
		t.Fatalf("期望文件名用上传原名 my-mcp-server-darwin-arm64，实际 %q", got)
	}

	// 无文件记录（指向不存在的文件 ID）：回退规则名
	noFile, err := svc.Create(CreateVersionInput{
		Version:     "1.1.0",
		AppType:     "mcp",
		Os:          "windows",
		Arch:        "amd64",
		DownloadURL: "/api/v1/public/files/9999/download",
		Sha256:      "def",
		FileSize:    100,
	})
	if err != nil {
		t.Fatalf("Create(noFile) 失败: %v", err)
	}
	if got := svc.DownloadFilename(noFile); got != "windows-amd64.exe" {
		t.Fatalf("期望回退规则名 windows-amd64.exe，实际 %q", got)
	}
}

// TestGetLatestCLIVersion_BatchLoadsFileOriginalNames 验证跨平台场景下
// SHA256 map 的 key 取自 File.original_name，且查不到 file 记录时回退规则名。
// 该用例同时回归 N+1 修复：所有 file 名应通过一次 IN 查询加载完成。
func TestGetLatestCLIVersion_BatchLoadsFileOriginalNames(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "version-batch.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&model.ClientVersion{}, &model.File{}); err != nil {
		t.Fatalf("AutoMigrate 失败: %v", err)
	}
	svc := NewVersionService(db, nil)

	// 三个平台各一份版本：两个指向 file 记录，一个指向不存在的 file（验证回退）
	file1 := model.File{Name: "m1", OriginalName: "mcp-darwin-arm64-v1", StoragePath: "/x", Size: 1}
	file2 := model.File{Name: "m2", OriginalName: "mcp-linux-amd64-v1", StoragePath: "/x", Size: 1}
	if err := db.Create(&file1).Error; err != nil {
		t.Fatalf("创建 file1 失败: %v", err)
	}
	if err := db.Create(&file2).Error; err != nil {
		t.Fatalf("创建 file2 失败: %v", err)
	}

	for _, c := range []struct {
		os, arch, ver, sha string
		url                string
	}{
		{"darwin", "arm64", "1.0.0", "sha-darwin", fmt.Sprintf("/api/v1/public/files/%d/download", file1.ID)},
		{"linux", "amd64", "1.0.0", "sha-linux", fmt.Sprintf("/api/v1/public/files/%d/download", file2.ID)},
		{"windows", "amd64", "1.0.0", "sha-windows", "/api/v1/public/files/9999/download"},
	} {
		if _, err := svc.Create(CreateVersionInput{
			Version:     c.ver,
			AppType:     "mcp",
			Os:          c.os,
			Arch:        c.arch,
			DownloadURL: c.url,
			Sha256:      c.sha,
			FileSize:    1,
		}); err != nil {
			t.Fatalf("Create(%s/%s) 失败: %v", c.os, c.arch, err)
		}
	}

	_, shaMap, err := svc.GetLatestCLIVersion("mcp")
	if err != nil {
		t.Fatalf("GetLatestCLIVersion 失败: %v", err)
	}
	if len(shaMap) != 3 {
		t.Fatalf("期望 sha256 map 含 3 个平台，实际 %d (%v)", len(shaMap), shaMap)
	}
	if shaMap["mcp-darwin-arm64-v1"] != "sha-darwin" {
		t.Fatalf("期望 darwin key 取 file original_name，实际 %v", shaMap)
	}
	if shaMap["mcp-linux-amd64-v1"] != "sha-linux" {
		t.Fatalf("期望 linux key 取 file original_name，实际 %v", shaMap)
	}
	// windows 指向不存在的 file ID → 回退规则名 "windows-amd64.exe"
	if shaMap["windows-amd64.exe"] != "sha-windows" {
		t.Fatalf("期望 windows 回退规则名 windows-amd64.exe，实际 %v", shaMap)
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

	_, shaMap, err := svc.GetLatestCLIVersion("")
	if err != nil {
		t.Fatalf("GetLatestCLIVersion 失败: %v", err)
	}
	if shaMap["windows-amd64.exe"] != "win-sha" {
		t.Fatalf("期望 Windows hash key 含 .exe（回退规则名），实际 map=%v", shaMap)
	}
}
