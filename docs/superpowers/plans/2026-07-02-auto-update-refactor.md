# QIM 自动更新功能整改计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 修复 QIM 自动更新功能的 12 项缺陷与改进点，覆盖版本判断、哈希校验、DDL 同步、架构规范化、灰度发布、版本分布统计、遗留组件清理、健壮性增强、回滚与推送能力。

**架构：** 后端（qim-server）作为更新分发服务器提供 latest.yml 和安装包下载；桌面客户端（qim-client）基于 electron-updater 主动轮询；管理后台（qim-admin）负责版本记录管理。本次整改重点在 service 层抽象、复用 WebSocket 统计版本分布、补全灰度发布能力。

**技术栈：** Go + Gin + GORM + electron-updater + Vue 3 + Element Plus

---

## 阶段总览

| 阶段 | 任务 | 优先级 | 类型 |
|------|------|--------|------|
| 一 | T1 版本判断改 semver 排序 | P0 | 缺陷 |
| 一 | T2 SHA512 计算失败明确报错 | P0 | 缺陷 |
| 一 | T3 同步 DDL 与 GORM 模型 | P0 | 缺陷 |
| 二 | T4 抽出 VersionService 层 | P1 | 架构 |
| 二 | T5 实现灰度发布 | P1 | 功能补全 |
| 二 | T6 复用 WebSocket 统计版本分布 | P1 | 功能补全 |
| 三 | T7 删除遗留组件 | P1 | 整洁 |
| 三 | T8 修复超时配置不一致 | P1 | 健壮性 |
| 四 | T9 修复单实例锁清理逻辑 | P2 | 健壮性 |
| 四 | T10 版本回滚 API | P2 | 增强 |
| 四 | T11 增量更新（Type 字段） | P2 | 增强 |
| 四 | T12 主动推送（WebSocket） | P2 | 增强 |

---

## 阶段一：P0 功能缺陷修复

### 任务 T1：版本判断改 semver 语义排序

**问题：** `qim-server/handler/update_handler.go:168` 和 `qim-server/handler/version_handler.go:63` 用 `ORDER BY created_at DESC` 取最新版本，补录旧版本会被误判为最新。

**方案：** 引入 `golang.org/x/mod/semver` 做语义比较，在 Go 层排序取最新（DB 不易做 semver 排序）。

**文件：**
- 修改：`qim-server/handler/update_handler.go`
- 修改：`qim-server/handler/version_handler.go`
- 创建：`qim-server/service/version_compare.go`（版本比较工具）
- 创建：`qim-server/service/version_compare_test.go`

**关键代码：**

`version_compare.go`：
```go
package service

import (
    "errors"
    "golang.org/x/mod/semver"
    "github.com/dshmyz/qim/qim-server/model"
    "gorm.io/gorm"
)

// IsValidVersion 校验版本号是否符合 semver，前端要求格式 \d+\.\d+\.\d+
func IsValidVersion(v string) bool {
    return semver.IsValid("v" + v)
}

// CompareVersions 比较 semver 版本号，返回 -1/0/1
// a > b 返回 1，a < b 返回 -1，相等返回 0
func CompareVersions(a, b string) int {
    return semver.Compare("v"+a, "v"+b)
}

// LatestVersion 从已启用的版本列表中选出 semver 最大的
func LatestVersion(versions []model.ClientVersion) (*model.ClientVersion, error) {
    if len(versions) == 0 {
        return nil, gorm.ErrRecordNotFound
    }
    latest := &versions[0]
    for i := 1; i < len(versions); i++ {
        if CompareVersions(versions[i].Version, latest.Version) > 0 {
            latest = &versions[i]
        }
    }
    return latest, nil
}

var ErrVersionExists = errors.New("version already exists")
var ErrVersionNotFound = errors.New("version not found")
var ErrMissingDownloadURL = errors.New("download url is required")
```

`update_handler.go` 改造 `GetLatestYML` 第 165-190 行：
```go
// 查询该平台所有已启用版本，再在 Go 层做 semver 排序
var versions []model.ClientVersion
err := db.Where("platform = ? AND enabled = ?", platform, true).Find(&versions).Error
if err != nil {
    logger.WithModule("Update").Error("查询版本失败", "platform", platform, "error", err)
    c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "查询更新失败"})
    return
}
version, err := service.LatestVersion(versions)
if err != nil {
    logger.WithModule("Update").Warn("无可用版本记录", "platform", platform)
    c.Status(http.StatusNotFound)
    return
}
```

同样改造 `RedirectUpdateFile` 第 138-143 行。

**验证：** 单元测试覆盖 `LatestVersion`（含补录旧版本场景）；现有 `version_update_handler_test.go` 跑通。

- [ ] **步骤 1：编写失败的测试**

```go
// version_compare_test.go
package service

import (
    "testing"
    "github.com/dshmyz/qim/qim-server/model"
)

func TestLatestVersion_PicksSemverMax(t *testing.T) {
    versions := []model.ClientVersion{
        {Version: "2.0.0", Platform: "windows"},
        {Version: "2.1.0", Platform: "windows"},
        {Version: "1.9.9", Platform: "windows"},
    }
    latest, err := LatestVersion(versions)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if latest.Version != "2.1.0" {
        t.Errorf("expected 2.1.0, got %s", latest.Version)
    }
}

func TestLatestVersion_IgnoresCreatedAtOrder(t *testing.T) {
    // 补录旧版本场景：created_at 晚但版本号低
    versions := []model.ClientVersion{
        {Version: "2.0.0", Platform: "windows"}, // 假设 created_at 晚
        {Version: "2.1.0", Platform: "windows"}, // 假设 created_at 早
    }
    latest, _ := LatestVersion(versions)
    if latest.Version != "2.1.0" {
        t.Errorf("expected 2.1.0, got %s", latest.Version)
    }
}

func TestLatestVersion_EmptyReturnsErrRecordNotFound(t *testing.T) {
    _, err := LatestVersion([]model.ClientVersion{})
    if err == nil {
        t.Error("expected error for empty list")
    }
}
```

- [ ] **步骤 2：运行测试验证失败**

运行：`cd qim-server && go test ./service/ -run TestLatestVersion -v`
预期：FAIL，报错 "package service is not found" 或函数未定义

- [ ] **步骤 3：编写实现代码**

创建 `version_compare.go`（代码如上）。

- [ ] **步骤 4：运行测试验证通过**

运行：`cd qim-server && go test ./service/ -run TestLatestVersion -v`
预期：PASS

- [ ] **步骤 5：改造 update_handler.go GetLatestYML**

替换第 165-190 行的查询逻辑。

- [ ] **步骤 6：改造 update_handler.go RedirectUpdateFile**

替换第 138-143 行的查询逻辑。

- [ ] **步骤 7：运行 handler 测试**

运行：`cd qim-server && go test ./handler/ -run TestUpdate -v`
预期：PASS

- [ ] **步骤 8：Commit**

```bash
git add qim-server/service/version_compare.go qim-server/service/version_compare_test.go qim-server/handler/update_handler.go
git commit -m "fix(update): 版本判断改用 semver 语义排序，避免补录旧版本被误判为最新"
```

---

### 任务 T2：SHA512 计算失败明确报错

**问题：** `qim-server/handler/version_handler.go:125-141` 中 `computeVersionFileSHA512` 对 HTTP URL 找不到本地缓存时静默返回 `"",0`，导致后续报错信息笼统。

**方案：** 让 `computeVersionFileSHA512` 返回 error，区分三种失败原因；HTTP URL 场景下增加实际下载计算的能力。

**文件：**
- 修改：`qim-server/handler/version_handler.go`（与 T4 协同迁移到 service 层）
- 创建：`qim-server/service/version_hash.go`
- 创建：`qim-server/service/version_hash_test.go`

**关键代码：**

`version_hash.go`：
```go
package service

import (
    "crypto/sha512"
    "fmt"
    "io"
    "net/http"
    "os"
    "path/filepath"
    "strconv"
    "strings"
    "time"
    "gorm.io/gorm"
    "github.com/dshmyz/qim/qim-server/model"
)

var httpClient = &http.Client{Timeout: 10 * time.Minute}

// ComputeFileHash 根据下载 URL 计算 SHA512 和文件大小
// 支持三种来源：公开文件 API、HTTP/HTTPS 远程、本地路径
func ComputeFileHash(db *gorm.DB, downloadURL, platform string) (sha512Hex string, size int64, err error) {
    // 1. 公开文件下载链接：从 DB 读 StoragePath
    if strings.Contains(downloadURL, "/api/v1/public/files/") && strings.HasSuffix(downloadURL, "/download") {
        return computeFromPublicFile(db, downloadURL)
    }
    // 2. HTTP/HTTPS：先尝试本地缓存，再走 HTTP 下载
    if strings.HasPrefix(downloadURL, "http://") || strings.HasPrefix(downloadURL, "https://") {
        if cached, ok := tryLocalCache(downloadURL, platform); ok {
            return cached.hash, cached.size, nil
        }
        return computeFromHTTP(downloadURL)
    }
    // 3. 本地路径
    return computeFromFile(downloadURL)
}

type hashResult struct {
    hash string
    size int64
}

func computeFromPublicFile(db *gorm.DB, downloadURL string) (string, int64, error) {
    parts := strings.Split(downloadURL, "/")
    for i, part := range parts {
        if part == "files" && i+1 < len(parts) {
            fileIDStr := parts[i+1]
            fileID, err := strconv.ParseUint(fileIDStr, 10, 32)
            if err != nil {
                return "", 0, fmt.Errorf("无效的文件 ID: %s", fileIDStr)
            }
            var file model.File
            if err := db.First(&file, uint(fileID)).Error; err != nil {
                return "", 0, fmt.Errorf("文件记录不存在: %w", err)
            }
            storagePath := file.StoragePath
            if strings.HasPrefix(storagePath, "/uploads/") {
                storagePath = "." + storagePath
            }
            return computeFromFile(storagePath)
        }
    }
    return "", 0, fmt.Errorf("无法解析公开文件 URL: %s", downloadURL)
}

func tryLocalCache(downloadURL, platform string) (hashResult, bool) {
    filename := filepath.Base(downloadURL)
    localPath := filepath.Join("./uploads/updates", platform, filename)
    if _, err := os.Stat(localPath); err != nil {
        return hashResult{}, false
    }
    h, s, err := computeFromFile(localPath)
    if err != nil {
        return hashResult{}, false
    }
    return hashResult{h, s}, true
}

func computeFromHTTP(url string) (string, int64, error) {
    resp, err := httpClient.Get(url)
    if err != nil {
        return "", 0, fmt.Errorf("下载安装包失败: %w", err)
    }
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusOK {
        return "", 0, fmt.Errorf("下载安装包失败: HTTP %d", resp.StatusCode)
    }
    hash := sha512.New()
    size, err := io.Copy(hash, resp.Body)
    if err != nil {
        return "", 0, fmt.Errorf("计算哈希失败: %w", err)
    }
    return fmt.Sprintf("%x", hash.Sum(nil)), size, nil
}

func computeFromFile(filePath string) (string, int64, error) {
    file, err := os.Open(filePath)
    if err != nil {
        return "", 0, fmt.Errorf("打开文件失败: %w", err)
    }
    defer file.Close()
    hash := sha512.New()
    size, err := io.Copy(hash, file)
    if err != nil {
        return "", 0, fmt.Errorf("计算哈希失败: %w", err)
    }
    return fmt.Sprintf("%x", hash.Sum(nil)), size, nil
}
```

`CreateVersion` 改造第 122-141 行，把 `computeVersionFileSHA512` 的 `(string,int64)` 改为 `(string,int64,error)`，失败时返回具体错误：
```go
sha512Hash, fileSize, hashErr := service.ComputeFileHash(db, req.DownloadUrl, req.Platform)
if hashErr != nil {
    logger.WithModule("Version").Error("计算安装包哈希失败", "version", req.Version, "error", hashErr)
    response.BadRequest(c, fmt.Sprintf("安装包校验信息计算失败: %v", hashErr))
    return
}
if sha512Hash != "" && fileSize > 0 {
    version.Sha512 = sha512Hash
    version.FileSize = fileSize
} else {
    response.BadRequest(c, "无法计算安装包校验信息")
    return
}
```

**验证：** 测试用例覆盖三种来源 + 失败场景（HTTP 404、文件不存在）。

- [ ] **步骤 1：编写失败的测试**

```go
// version_hash_test.go
package service

import (
    "os"
    "path/filepath"
    "testing"
)

func TestComputeFromFile_Success(t *testing.T) {
    tmpDir := t.TempDir()
    tmpFile := filepath.Join(tmpDir, "test.txt")
    os.WriteFile(tmpFile, []byte("hello world"), 0644)

    hash, size, err := computeFromFile(tmpFile)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if size != int64(len("hello world")) {
        t.Errorf("expected size %d, got %d", len("hello world"), size)
    }
    if hash == "" {
        t.Error("expected non-empty hash")
    }
}

func TestComputeFromFile_NotExist(t *testing.T) {
    _, _, err := computeFromFile("/nonexistent/path/file.txt")
    if err == nil {
        t.Error("expected error for nonexistent file")
    }
}
```

- [ ] **步骤 2：运行测试验证失败**

运行：`cd qim-server && go test ./service/ -run TestComputeFromFile -v`
预期：FAIL

- [ ] **步骤 3：编写实现代码**

创建 `version_hash.go`（代码如上）。

- [ ] **步骤 4：运行测试验证通过**

运行：`cd qim-server && go test ./service/ -run TestComputeFromFile -v`
预期：PASS

- [ ] **步骤 5：改造 version_handler.go CreateVersion**

替换第 122-141 行的哈希计算逻辑，使用 `service.ComputeFileHash`。

- [ ] **步骤 6：运行 handler 测试**

运行：`cd qim-server && go test ./handler/ -run TestCreateVersion -v`
预期：PASS

- [ ] **步骤 7：Commit**

```bash
git add qim-server/service/version_hash.go qim-server/service/version_hash_test.go qim-server/handler/version_handler.go
git commit -m "fix(update): SHA512 计算失败时返回明确错误，HTTP URL 支持远程下载计算"
```

---

### 任务 T3：同步 DDL 与 GORM 模型

**问题：** `qim-server/ddl_mysql.sql:611-624` 和 `qim-server/ddl_sqlite.sql:610-623` 缺少 `sha512` 和 `file_size` 列，依赖 AutoMigrate。同时 T5 要加 `rollout_percentage`，一并补齐。当前模型唯一索引 `idx_version_deleted` 只含 version+deleted_at，不含 platform，与"version+platform 唯一"的校验逻辑不一致。

**文件：**
- 修改：`qim-server/ddl_mysql.sql` 第 611-624 行
- 修改：`qim-server/ddl_sqlite.sql` 第 610-623 行
- 创建：`qim-server/migrations/002_add_version_columns.sql`
- 修改：`qim-server/model/model.go` ClientVersion 结构（修正唯一索引）

**关键代码：**

`ddl_mysql.sql` 补字段：
```sql
CREATE TABLE IF NOT EXISTS `client_versions` (
  `id` INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  `version` VARCHAR(50) NOT NULL,
  `platform` VARCHAR(20) NOT NULL,
  `type` VARCHAR(20) DEFAULT 'full',
  `download_url` VARCHAR(500),
  `sha512` VARCHAR(200),
  `file_size` BIGINT DEFAULT 0,
  `changelog` TEXT,
  `force_update` BOOLEAN DEFAULT FALSE,
  `rollout_percentage` INT DEFAULT 100,
  `enabled` BOOLEAN DEFAULT TRUE,
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` DATETIME,
  UNIQUE KEY `idx_version_platform` (`version`, `platform`, `deleted_at`),
  INDEX `idx_client_versions_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

`ddl_sqlite.sql` 同步：
```sql
CREATE TABLE IF NOT EXISTS `client_versions` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `version` VARCHAR(50) NOT NULL,
  `platform` VARCHAR(20) NOT NULL,
  `type` VARCHAR(20) DEFAULT 'full',
  `download_url` VARCHAR(500),
  `sha512` VARCHAR(200),
  `file_size` INTEGER DEFAULT 0,
  `changelog` TEXT,
  `force_update` INTEGER DEFAULT 0,
  `rollout_percentage` INTEGER DEFAULT 100,
  `enabled` INTEGER DEFAULT 1,
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `deleted_at` DATETIME
);
CREATE UNIQUE INDEX IF NOT EXISTS `idx_version_platform` ON `client_versions`(`version`, `platform`, `deleted_at`);
CREATE INDEX IF NOT EXISTS `idx_client_versions_deleted_at` ON `client_versions`(`deleted_at`);
```

`migrations/002_add_version_columns.sql`：
```sql
-- MySQL 增量迁移
ALTER TABLE `client_versions` ADD COLUMN IF NOT EXISTS `sha512` VARCHAR(200);
ALTER TABLE `client_versions` ADD COLUMN IF NOT EXISTS `file_size` BIGINT DEFAULT 0;
ALTER TABLE `client_versions` ADD COLUMN IF NOT EXISTS `rollout_percentage` INT DEFAULT 100;

-- 修正唯一索引（删除旧的，创建新的）
ALTER TABLE `client_versions` DROP INDEX IF EXISTS `idx_version_deleted`;
ALTER TABLE `client_versions` ADD UNIQUE KEY `idx_version_platform` (`version`, `platform`, `deleted_at`);
```

`model.go` 修正 ClientVersion 唯一索引（第 797 行）：
```go
type ClientVersion struct {
    ID                uint           `json:"id" gorm:"primarykey"`
    Version           string         `json:"version" gorm:"size:50;uniqueIndex:idx_version_platform;not null"`
    Platform          string         `json:"platform" gorm:"size:20;uniqueIndex:idx_version_platform;not null"`
    Type              string         `json:"type" gorm:"size:20;default:'full'"`
    DownloadURL       string         `json:"download_url" gorm:"size:500"`
    Sha512            string         `json:"sha512" gorm:"size:200"`
    FileSize          int64          `json:"file_size" gorm:"default:0"`
    Changelog         string         `json:"changelog" gorm:"type:text"`
    ForceUpdate       bool           `json:"force_update" gorm:"default:false"`
    RolloutPercentage int            `json:"rollout_percentage" gorm:"default:100"`
    Enabled           bool           `json:"enabled" gorm:"default:true"`
    CreatedAt         time.Time      `json:"created_at"`
    UpdatedAt         time.Time      `json:"updated_at"`
    DeletedAt         gorm.DeletedAt `json:"-" gorm:"uniqueIndex:idx_version_platform"`
}
```

**验证：** 全新库执行 DDL，启动服务确认 AutoMigrate 不再补列；旧库执行 002 迁移脚本。

- [ ] **步骤 1：修改 ddl_mysql.sql 第 611-624 行**
- [ ] **步骤 2：修改 ddl_sqlite.sql 第 610-623 行**
- [ ] **步骤 3：创建 migrations/002_add_version_columns.sql**
- [ ] **步骤 4：修改 model.go ClientVersion 唯一索引和新增 RolloutPercentage 字段**
- [ ] **步骤 5：启动服务验证 AutoMigrate 无新增列**
- [ ] **步骤 6：Commit**

```bash
git add qim-server/ddl_mysql.sql qim-server/ddl_sqlite.sql qim-server/migrations/002_add_version_columns.sql qim-server/model/model.go
git commit -m "fix(update): 同步 DDL 与 GORM 模型，补齐 sha512/file_size/rollout_percentage 列，修正唯一索引"
```

---

## 阶段二：P1 架构重构与功能补全

### 任务 T4：抽出 VersionService 层

**问题：** `qim-server/service/version_service.go` 是死代码，handler 直接操作 DB，违反"后端功能复用要充分抽象"规范。

**方案：** 将 handler 中的业务逻辑（CRUD、SHA512 计算、平台映射、版本比较、灰度过滤）下沉到 service，handler 只做 HTTP 层解析和响应。

**文件：**
- 重写：`qim-server/service/version_service.go`
- 修改：`qim-server/handler/version_handler.go`
- 修改：`qim-server/handler/update_handler.go`
- 创建：`qim-server/service/version_service_test.go`

**关键代码：**

`version_service.go`：
```go
package service

import (
    "fmt"
    "strings"
    "gorm.io/gorm"
    "github.com/dshmyz/qim/qim-server/model"
)

type VersionService struct {
    db *gorm.DB
}

func NewVersionService(db *gorm.DB) *VersionService {
    return &VersionService{db: db}
}

// CreateVersionInput 创建版本的入参
type CreateVersionInput struct {
    Version           string
    Platform          string
    DownloadURL       string
    Changelog         string
    ForceUpdate       bool
    RolloutPercentage int
    Sha512            string
    FileSize          int64
}

// Create 创建版本，自动校验唯一性、计算哈希
func (s *VersionService) Create(input CreateVersionInput) (*model.ClientVersion, error) {
    // 唯一性校验
    var existing model.ClientVersion
    if err := s.db.Where("version = ? AND platform = ? AND deleted_at IS NULL",
        input.Version, input.Platform).First(&existing).Error; err == nil {
        return nil, ErrVersionExists
    }
    // 平台标准化
    platform := NormalizePlatform(input.Platform)
    version := model.ClientVersion{
        Version:           input.Version,
        Platform:          platform,
        Type:              "full",
        DownloadURL:       input.DownloadURL,
        Changelog:         input.Changelog,
        ForceUpdate:       input.ForceUpdate,
        RolloutPercentage: input.RolloutPercentage,
        Enabled:           true,
    }
    if version.RolloutPercentage == 0 {
        version.RolloutPercentage = 100
    }
    // 哈希处理
    if input.Sha512 != "" && input.FileSize > 0 {
        version.Sha512 = input.Sha512
        version.FileSize = input.FileSize
    } else if input.DownloadURL != "" {
        hash, size, err := ComputeFileHash(s.db, input.DownloadURL, platform)
        if err != nil {
            return nil, fmt.Errorf("计算安装包哈希失败: %w", err)
        }
        version.Sha512 = hash
        version.FileSize = size
    } else {
        return nil, ErrMissingDownloadURL
    }
    if err := s.db.Create(&version).Error; err != nil {
        return nil, err
    }
    return &version, nil
}

// GetLatestEnabled 获取平台最新已启用版本（semver 排序 + 灰度过滤）
func (s *VersionService) GetLatestEnabled(platform string, clientID string) (*model.ClientVersion, error) {
    platform = NormalizePlatform(platform)
    var versions []model.ClientVersion
    if err := s.db.Where("platform = ? AND enabled = ?", platform, true).Find(&versions).Error; err != nil {
        return nil, err
    }
    versions = FilterByRollout(versions, clientID)
    return LatestVersion(versions)
}

// List 分页查询
func (s *VersionService) List(page, pageSize int, platform string) ([]model.ClientVersion, int64, error) {
    if page < 1 {
        page = 1
    }
    if pageSize < 1 || pageSize > 100 {
        pageSize = 20
    }
    query := s.db.Model(&model.ClientVersion{})
    if platform != "" {
        query = query.Where("platform = ?", platform)
    }
    var total int64
    query.Count(&total)
    var versions []model.ClientVersion
    offset := (page - 1) * pageSize
    if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&versions).Error; err != nil {
        return nil, 0, err
    }
    return versions, total, nil
}

// UpdateVersionInput 更新版本的入参
type UpdateVersionInput struct {
    UpdateNotes       *string
    ForceUpdate       *bool
    Status            *string
    RolloutPercentage *int
}

func (s *VersionService) Update(id uint, input UpdateVersionInput) (*model.ClientVersion, error) {
    var version model.ClientVersion
    if err := s.db.First(&version, id).Error; err != nil {
        return nil, ErrVersionNotFound
    }
    if input.UpdateNotes != nil {
        version.Changelog = *input.UpdateNotes
    }
    if input.ForceUpdate != nil {
        version.ForceUpdate = *input.ForceUpdate
    }
    if input.Status != nil {
        version.Enabled = *input.Status == "active"
    }
    if input.RolloutPercentage != nil {
        version.RolloutPercentage = *input.RolloutPercentage
    }
    if err := s.db.Save(&version).Error; err != nil {
        return nil, err
    }
    return &version, nil
}

func (s *VersionService) Delete(id uint) error {
    var version model.ClientVersion
    if err := s.db.First(&version, id).Error; err != nil {
        return ErrVersionNotFound
    }
    return s.db.Delete(&version).Error
}

func (s *VersionService) ToggleStatus(id uint, enabled bool) (*model.ClientVersion, error) {
    var version model.ClientVersion
    if err := s.db.First(&version, id).Error; err != nil {
        return nil, ErrVersionNotFound
    }
    version.Enabled = enabled
    if err := s.db.Save(&version).Error; err != nil {
        return nil, err
    }
    return &version, nil
}

// Rollback 回滚到指定版本：禁用比它新的所有已启用版本
func (s *VersionService) Rollback(id uint) error {
    var target model.ClientVersion
    if err := s.db.First(&target, id).Error; err != nil {
        return ErrVersionNotFound
    }
    var newer []model.ClientVersion
    s.db.Where("platform = ? AND enabled = ? AND id != ?", target.Platform, true, id).Find(&newer)
    for _, v := range newer {
        if CompareVersions(v.Version, target.Version) > 0 {
            s.db.Model(&v).Update("enabled", false)
        }
    }
    return s.db.Model(&target).Update("enabled", true).Error
}

// NormalizePlatform 平台别名映射
func NormalizePlatform(platform string) string {
    alias := map[string]string{
        "win": "windows", "win7": "windows", "win10": "windows", "windows": "windows",
        "mac": "macos", "macos": "macos",
        "linux": "linux",
    }
    if mapped, ok := alias[strings.ToLower(platform)]; ok {
        return mapped
    }
    return strings.ToLower(platform)
}
```

handler 改为薄层（示例 CreateVersion）：
```go
func CreateVersion(c *gin.Context) {
    var req struct {
        Version           string `json:"version" binding:"required"`
        Platform          string `json:"platform" binding:"required"`
        ReleaseDate       string `json:"releaseDate"`
        DownloadUrl       string `json:"downloadUrl"`
        UpdateNotes       string `json:"updateNotes"`
        ForceUpdate       bool   `json:"forceUpdate"`
        RolloutPercentage int    `json:"rolloutPercentage"`
        Sha512            string `json:"sha512"`
        FileSize          int64  `json:"fileSize"`
    }
    if err := c.ShouldBindJSON(&req); err != nil {
        response.BadRequest(c, "参数错误")
        return
    }
    if !service.IsValidVersion(req.Version) {
        response.BadRequest(c, "版本号格式不正确（如：2.1.0）")
        return
    }
    svc := service.NewVersionService(database.GetDB())
    version, err := svc.Create(service.CreateVersionInput{
        Version:           req.Version,
        Platform:          req.Platform,
        DownloadURL:       req.DownloadUrl,
        Changelog:         req.UpdateNotes,
        ForceUpdate:       req.ForceUpdate,
        RolloutPercentage: req.RolloutPercentage,
        Sha512:            req.Sha512,
        FileSize:          req.FileSize,
    })
    if err != nil {
        switch {
        case errors.Is(err, service.ErrVersionExists):
            response.BadRequest(c, "该版本已存在")
        case errors.Is(err, service.ErrMissingDownloadURL):
            response.BadRequest(c, "下载链接不能为空")
        default:
            response.InternalServerError(c, err.Error())
        }
        return
    }
    response.Success(c, versionToFrontend(*version))
}
```

**验证：** service 层单元测试覆盖所有方法；handler 测试保持原有行为。

- [ ] **步骤 1：编写 service 层测试**（覆盖 Create/GetLatestEnabled/List/Update/Delete/ToggleStatus/Rollback）
- [ ] **步骤 2：运行测试验证失败**
- [ ] **步骤 3：重写 version_service.go**（代码如上）
- [ ] **步骤 4：运行测试验证通过**
- [ ] **步骤 5：改造 version_handler.go 所有方法为薄层**
- [ ] **步骤 6：改造 update_handler.go GetLatestYML 和 RedirectUpdateFile 使用 service**
- [ ] **步骤 7：运行所有 handler 测试**
- [ ] **步骤 8：Commit**

```bash
git add qim-server/service/version_service.go qim-server/service/version_service_test.go qim-server/handler/version_handler.go qim-server/handler/update_handler.go
git commit -m "refactor(update): 抽出 VersionService 层，handler 改为薄层，业务逻辑下沉"
```

---

### 任务 T5：实现灰度发布

**问题：** `qim-admin/src/views/ClientManagement/components/VersionFormDialog.vue:66-76` 前端有灰度百分比输入，但后端模型无此字段，`GetLatestYML` 不做过滤。

**方案：** 模型加 `RolloutPercentage` 字段（已在 T3 添加），service 层基于客户端稳定 ID 做哈希分桶过滤。

**文件：**
- 修改：`qim-server/service/version_service.go`（`GetLatestEnabled` 已含灰度过滤调用）
- 创建：`qim-server/service/rollout.go`
- 创建：`qim-server/service/rollout_test.go`
- 修改：`qim-server/handler/update_handler.go`（latest.yml 透传 clientID）
- 修改：`qim-client/electron/auto-update.js`（请求带 clientID）
- 修改：`qim-server/handler/version_handler.go`（versionToFrontend 输出 rolloutPercentage）

**关键代码：**

`rollout.go`：
```go
package service

import (
    "crypto/md5"
    "encoding/binary"
    "github.com/dshmyz/qim/qim-server/model"
)

// FilterByRollout 根据客户端 ID 的哈希分桶判断是否命中灰度
// clientID 为空时仅放行 100% 全量版本
func FilterByRollout(versions []model.ClientVersion, clientID string) []model.ClientVersion {
    var result []model.ClientVersion
    bucket := uint16(0)
    if clientID != "" {
        sum := md5.Sum([]byte(clientID))
        bucket = binary.BigEndian.Uint16(sum[:2]) % 100 // 0-99
    }
    for _, v := range versions {
        if v.RolloutPercentage >= 100 {
            result = append(result, v)
            continue
        }
        if clientID == "" {
            continue // 未携带 clientID 时跳过灰度版本
        }
        if bucket < uint16(v.RolloutPercentage) {
            result = append(result, v)
        }
    }
    return result
}
```

客户端请求带 clientID（首次启动生成持久化到 userData/client_id.txt）：

`auto-update.js` 改造 `resolveUpdateFeedUrl`：
```javascript
const { readFileSync, writeFileSync, existsSync } = require('fs')
const { join } = require('path')

function getClientId() {
  const idPath = join(app.getPath('userData'), 'client_id.txt')
  if (existsSync(idPath)) {
    return readFileSync(idPath, 'utf-8').trim()
  }
  const id = require('crypto').randomUUID()
  writeFileSync(idPath, id)
  return id
}

function resolveUpdateFeedUrl() {
  const baseUrl = getUpdateBaseUrl()
  const clientId = getClientId()
  // ... 现有平台判断
  return `${baseUrl}/api/v1/updates/${platform}/?client=${clientId}`
}
```

后端 `update_handler.go` 解析 query 参数传给 service：
```go
clientID := c.Query("client")
version, err := versionSvc.GetLatestEnabled(platform, clientID)
```

`version_handler.go` 的 `versionToFrontend` 输出 `rolloutPercentage`：
```go
func versionToFrontend(v model.ClientVersion) gin.H {
    // ...
    "rolloutPercentage": v.RolloutPercentage,
    // ...
}
```

**验证：** rollout_test 覆盖边界（0%、50%、100%、空 clientID）；集成测试确认 100% 全量版本始终返回。

- [ ] **步骤 1：编写 rollout 失败测试**

```go
// rollout_test.go
package service

import (
    "testing"
    "github.com/dshmyz/qim/qim-server/model"
)

func TestFilterByRollout_FullRelease(t *testing.T) {
    versions := []model.ClientVersion{{Version: "2.0.0", RolloutPercentage: 100}}
    result := FilterByRollout(versions, "")
    if len(result) != 1 {
        t.Errorf("expected 1, got %d", len(result))
    }
}

func TestFilterByRollout_GrayReleaseWithClientID(t *testing.T) {
    versions := []model.ClientVersion{{Version: "2.0.0", RolloutPercentage: 50}}
    // 同一 clientID 应稳定命中或不命中
    result1 := FilterByRollout(versions, "client-1")
    result2 := FilterByRollout(versions, "client-1")
    if len(result1) != len(result2) {
        t.Error("same clientID should produce stable result")
    }
}

func TestFilterByRollout_GrayReleaseNoClientID(t *testing.T) {
    versions := []model.ClientVersion{{Version: "2.0.0", RolloutPercentage: 50}}
    result := FilterByRollout(versions, "")
    if len(result) != 0 {
        t.Error("gray release without clientID should return empty")
    }
}
```

- [ ] **步骤 2：运行测试验证失败**
- [ ] **步骤 3：创建 rollout.go**
- [ ] **步骤 4：运行测试验证通过**
- [ ] **步骤 5：改造 update_handler.go 传递 clientID**
- [ ] **步骤 6：改造 version_handler.go versionToFrontend 输出 rolloutPercentage**
- [ ] **步骤 7：改造 auto-update.js 生成并携带 clientID**
- [ ] **步骤 8：集成测试验证灰度过滤**
- [ ] **步骤 9：Commit**

```bash
git add qim-server/service/rollout.go qim-server/service/rollout_test.go qim-server/handler/update_handler.go qim-server/handler/version_handler.go qim-client/electron/auto-update.js
git commit -m "feat(update): 实现灰度发布，基于客户端 ID 哈希分桶过滤"
```

---

### 任务 T6：复用 WebSocket 统计版本分布

**问题：** `qim-server/handler/version_handler.go:294-311` `GetVersionDistribution` 查 `users.client_version` 字段，但 User 表无此字段，统计恒为空。

**方案：** 复用现有 WebSocket 连接，客户端连接时携带 `version` 和 `platform`，Hub 维护内存计数器，分布统计改为读 Hub 内存。DB 改动为 0，无需额外上报请求。

**文件：**
- 修改：`qim-server/ws/ws.go`（Client 结构、Hub 结构、ServeWs、register/unregister、asyncBroadcast 失败清理）
- 修改：`qim-server/handler/version_handler.go`（`GetVersionDistribution` 改为读 Hub）
- 修改：`qim-server/app/routes.go`（distribution 路由注入 hub）
- 修改：`qim-client/src/utils/websocketManager.ts`（WS URL 加 version/platform）
- 修改：`qim-client/src/composables/useWebSocket.ts`（WS URL 加 version/platform）
- 创建：`qim-server/ws/version_stats_test.go`

**关键代码：**

**1. ws.go - Client 与 Hub 结构扩展**

Client 加字段：
```go
type Client struct {
    hub      *Hub
    conn     *websocket.Conn
    send     chan []byte
    userID   uint
    authed   bool
    jwtToken string
    version  string // 客户端版本号
    platform string // 客户端平台
}
```

Hub 加字段：
```go
type Hub struct {
    // ... 现有字段
    versionStats sync.Map // key: "version|platform" → *int64
}
```

**2. ws.go - 计数辅助方法**

```go
import "sync/atomic"

// versionStatsKey 生成版本统计的 map key
func versionStatsKey(version, platform string) string {
    return version + "|" + platform
}

// incVersionStats 版本计数 +1
func (h *Hub) incVersionStats(version, platform string) {
    if version == "" {
        return
    }
    key := versionStatsKey(version, platform)
    if v, ok := h.versionStats.Load(key); ok {
        atomic.AddInt64(v.(*int64), 1)
        return
    }
    var count int64 = 1
    actual, loaded := h.versionStats.LoadOrStore(key, &count)
    if loaded {
        // 已存在，重新 +1
        atomic.AddInt64(actual.(*int64), 1)
    }
}

// decVersionStats 版本计数 -1，不低于 0
func (h *Hub) decVersionStats(version, platform string) {
    if version == "" {
        return
    }
    key := versionStatsKey(version, platform)
    if v, ok := h.versionStats.Load(key); ok {
        atomic.AddInt64(v.(*int64), -1)
    }
}

// VersionStat 版本统计项
type VersionStat struct {
    Version  string `json:"version"`
    Platform string `json:"platform"`
    Count    int64  `json:"count"`
}

// GetVersionStats 返回版本分布统计快照
func (h *Hub) GetVersionStats() []VersionStat {
    var stats []VersionStat
    h.versionStats.Range(func(key, value interface{}) bool {
        count := atomic.LoadInt64(value.(*int64))
        if count <= 0 {
            return true
        }
        parts := strings.Split(key.(string), "|")
        if len(parts) != 2 {
            return true
        }
        stats = append(stats, VersionStat{
            Version:  parts[0],
            Platform: parts[1],
            Count:    count,
        })
        return true
    })
    return stats
}
```

**3. ws.go - ServeWs 读取 query 参数**

```go
func ServeWs(hub *Hub, c *gin.Context) {
    conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
    if err != nil {
        logger.WithModule("WS").Error("WebSocket升级失败", "error", err)
        return
    }

    userID, exists := c.Get("user_id")
    client := &Client{
        hub:      hub,
        conn:     conn,
        send:     make(chan []byte, 1024),
        version:  c.Query("version"),
        platform: c.Query("platform"),
    }
    if exists {
        client.userID = userID.(uint)
        client.authed = true
        client.hub.register <- client
    }

    utils.SafeGoWithLabel("ws-write", func() { client.writePump() })
    utils.SafeGoWithLabel("ws-read", func() { client.readPump() })
}
```

**4. ws.go - register/unregister 维护计数**

register 分支加：
```go
case client := <-h.register:
    h.clients.Store(client, true)
    if existingClients, ok := h.userClients.Load(client.userID); ok {
        clients := existingClients.([]*Client)
        clients = append(clients, client)
        h.userClients.Store(client.userID, clients)
    } else {
        h.userClients.Store(client.userID, []*Client{client})
    }
    h.incVersionStats(client.version, client.platform) // 新增
    logger.WithModule("WS").Info("用户连接", "userID", client.userID)
    h.UpdateUserStatus(client.userID, StatusOnline)
```

unregister 分支加：
```go
case client := <-h.unregister:
    h.clients.LoadAndDelete(client)
    safeCloseSend(client.send)
    h.decVersionStats(client.version, client.platform) // 新增
    // ... 现有逻辑不变
```

`asyncBroadcast` 中失败客户端清理（第 265-285 行）也要加 `h.decVersionStats`，避免计数泄漏。

**5. version_handler.go - 改为读 Hub**

```go
func GetVersionDistribution(c *gin.Context) {
    hub, ok := c.MustGet("hub").(*ws.Hub)
    if !ok {
        response.InternalServerError(c, "WebSocket Hub 未初始化")
        return
    }
    stats := hub.GetVersionStats()
    if stats == nil {
        stats = []ws.VersionStat{}
    }
    response.Success(c, stats)
}
```

**6. routes.go - 注入 hub**

确认 distribution 路由所在 group 能拿到 hub，通过中间件 `c.Set("hub", hub)` 或闭包捕获。

**7. 客户端 WS URL 加参数**

`websocketManager.ts:33`：
```typescript
import { APP_CONFIG } from '@/config/appConfig'

const platform = navigator.userAgent.toLowerCase().includes('mac') ? 'macos'
  : navigator.userAgent.toLowerCase().includes('linux') ? 'linux'
  : 'windows'
const wsUrl = `ws${serverUrl.startsWith('https') ? 's' : ''}://${serverUrl.replace(/^https?:\/\//, '')}/api/v1/ws?token=${token}&version=${encodeURIComponent(APP_CONFIG.version)}&platform=${platform}`;
```

`useWebSocket.ts:283` 同样改造。

**验证：**
1. 单元测试 `version_stats_test.go`：并发 register/unregister，验证计数准确性
2. 集成测试：两个客户端连接（不同版本），调用 `/api/v1/client/versions/distribution`，确认返回两条记录
3. 断开测试：客户端断开后再次查询，确认计数归零
4. 空参测试：客户端不带 version 参数连接，不影响计数

- [ ] **步骤 1：编写并发计数失败测试**

```go
// version_stats_test.go
package ws

import (
    "sync"
    "testing"
)

func TestVersionStats_ConcurrentIncDec(t *testing.T) {
    hub := &Hub{versionStats: sync.Map{}}
    var wg sync.WaitGroup
    // 100 个并发 +1
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            hub.incVersionStats("2.0.0", "windows")
        }()
    }
    wg.Wait()
    // 再 50 个并发 -1
    for i := 0; i < 50; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            hub.decVersionStats("2.0.0", "windows")
        }()
    }
    wg.Wait()
    stats := hub.GetVersionStats()
    if len(stats) != 1 {
        t.Fatalf("expected 1 stat, got %d", len(stats))
    }
    if stats[0].Count != 50 {
        t.Errorf("expected count 50, got %d", stats[0].Count)
    }
}

func TestVersionStats_EmptyVersionSkipped(t *testing.T) {
    hub := &Hub{versionStats: sync.Map{}}
    hub.incVersionStats("", "windows")
    stats := hub.GetVersionStats()
    if len(stats) != 0 {
        t.Errorf("empty version should be skipped, got %d stats", len(stats))
    }
}
```

- [ ] **步骤 2：运行测试验证失败**
- [ ] **步骤 3：修改 ws.go Client 和 Hub 结构**
- [ ] **步骤 4：实现 incVersionStats/decVersionStats/GetVersionStats**
- [ ] **步骤 5：改造 ServeWs 读取 query 参数**
- [ ] **步骤 6：改造 register/unregister/asyncBroadcast 维护计数**
- [ ] **步骤 7：运行单元测试验证通过**
- [ ] **步骤 8：改造 version_handler.go GetVersionDistribution**
- [ ] **步骤 9：改造 routes.go 注入 hub**
- [ ] **步骤 10：改造 websocketManager.ts 和 useWebSocket.ts 加 version/platform**
- [ ] **步骤 11：集成测试验证**
- [ ] **步骤 12：Commit**

```bash
git add qim-server/ws/ws.go qim-server/ws/version_stats_test.go qim-server/handler/version_handler.go qim-server/app/routes.go qim-client/src/utils/websocketManager.ts qim-client/src/composables/useWebSocket.ts
git commit -m "feat(update): 复用 WebSocket 内存统计版本分布，替代无效的 DB 查询"
```

---

## 阶段三：P1 前端清理与健壮性

### 任务 T7：删除遗留组件

**问题：** `qim-client/src/components/modals/VersionUpdateDialog.vue` 引用了 `utils/version.ts` 中不存在的 `openUpdateLink`，实际更新 UI 在 `MainDialogs.vue`。

**文件：**
- 删除：`qim-client/src/components/modals/VersionUpdateDialog.vue`
- 修改：所有引用该组件的位置（需先 grep 确认）

**步骤：**

- [ ] **步骤 1：grep 查找所有引用**

运行：`grep -r "VersionUpdateDialog" qim-client/src/`

- [ ] **步骤 2：删除引用和导入**

根据 grep 结果修改对应文件。

- [ ] **步骤 3：删除组件文件**

删除 `qim-client/src/components/modals/VersionUpdateDialog.vue`

- [ ] **步骤 4：构建验证**

运行：`cd qim-client && npm run build`
预期：无报错

- [ ] **步骤 5：Commit**

```bash
git add -A qim-client/src/
git commit -m "chore(client): 删除遗留的 VersionUpdateDialog 组件"
```

---

### 任务 T8：修复超时配置不一致

**问题：** 主进程 `qim-client/electron/auto-update.js:5` 设 10 秒，渲染进程 `qim-client/src/views/Main.vue:3009` 设 12 秒，两者不协调。

**方案：** 统一主进程超时为 12 秒，主进程超时后主动发送 `update-error` 事件；渲染进程本地 timer 改为兜底（30 秒，仅防主进程卡死）。

**文件：**
- 修改：`qim-client/electron/auto-update.js`
- 修改：`qim-client/src/views/Main.vue`

**关键代码：**

`auto-update.js`：
```javascript
const CHECK_UPDATE_TIMEOUT_MS = 12000 // 与渲染进程一致

function checkForUpdates() {
  // ...
  const timeout = setTimeout(() => {
    updatePhase = 'idle'
    sendToWindow('update-error', '检查更新超时，请检查网络连接或服务器地址')
  }, CHECK_UPDATE_TIMEOUT_MS)
  // ...
}
```

`Main.vue` 渲染进程本地 timer 改为兜底：
```typescript
window.electron.ipcRenderer.send('check-for-updates')
// 兜底超时：30 秒（远大于主进程 12 秒），仅防主进程卡死
window.setTimeout(() => {
  if (!isCheckingUpdate.value) return
  isCheckingUpdate.value = false
  updateResult.value = '检查更新超时，请稍后重试'
}, 30000)
```

**验证：** 模拟网络超时，确认主进程先触发事件、渲染进程正确清理状态。

- [ ] **步骤 1：修改 auto-update.js CHECK_UPDATE_TIMEOUT_MS 为 12000**
- [ ] **步骤 2：确保超时后 sendToWindow('update-error', ...)**
- [ ] **步骤 3：修改 Main.vue 渲染进程超时为 30000 兜底**
- [ ] **步骤 4：手动测试网络超时场景**
- [ ] **步骤 5：Commit**

```bash
git add qim-client/electron/auto-update.js qim-client/src/views/Main.vue
git commit -m "fix(update): 统一主进程与渲染进程超时配置，修复不协调问题"
```

---

## 阶段四：P2 增强（建议独立立项）

### 任务 T9：修复单实例锁清理逻辑

**问题：** `qim-client/electron/main.js:42-55` 拿不到锁时直接 `fs.unlinkSync` 删除 `SingletonLock`，若前实例正在 `quitAndInstall` 中可能造成竞争。

**方案：** 不主动删锁，改为：拿不到锁时等待短超时，仍失败则聚焦已有窗口并退出；仅在确认进程已死（通过 PID 检查）时清理。

**文件：**
- 修改：`qim-client/electron/main.js`

**关键代码：**
```javascript
if (app.isPackaged) {
  let gotTheLock = app.requestSingleInstanceLock()
  if (!gotTheLock) {
    // 不直接删锁，先等待前实例退出（quitAndInstall 场景）
    for (let i = 0; i < 5; i++) {
      await new Promise(r => setTimeout(r, 500))
      gotTheLock = app.requestSingleInstanceLock()
      if (gotTheLock) break
    }
  }
  if (!gotTheLock) {
    // 确认锁文件对应的进程已死才清理
    if (isLockStale()) {
      const lockPath = path.join(app.getPath('userData'), 'SingletonLock')
      try { fs.unlinkSync(lockPath) } catch (e) {}
      gotTheLock = app.requestSingleInstanceLock()
    }
  }
  if (!gotTheLock) {
    app.quit()
    process.exit(0)
  }
}
```

`isLockStale` 通过读取锁文件中的 PID 判断进程是否存在（macOS/Linux 用 `process.kill(pid, 0)`，Windows 用 `tasklist`）。

**验证：** 模拟双启动、模拟 quitAndInstall 过程中再次启动。

- [ ] **步骤 1：实现 isLockStale 函数**
- [ ] **步骤 2：改造 main.js 锁清理逻辑**
- [ ] **步骤 3：手动测试双启动场景**
- [ ] **步骤 4：手动测试 quitAndInstall 过程中再次启动**
- [ ] **步骤 5：Commit**

```bash
git add qim-client/electron/main.js
git commit -m "fix(client): 修复单实例锁清理逻辑，避免与 quitAndInstall 竞争"
```

---

### 任务 T10：版本回滚 API

**问题：** 当前 `version + platform` 唯一索引导致回滚必须先删新版本记录，风险大；无回滚 API。

**方案：** 新增 `POST /api/v1/admin/client/versions/:id/rollback`，将指定旧版本标记为"当前生效"，把比它新的版本批量 `enabled=false`。Service 层已在 T4 实现 `Rollback` 方法。

**文件：**
- 修改：`qim-server/app/routes.go`（新增路由）
- 修改：`qim-server/handler/version_handler.go`（新增 handler）
- 修改：`qim-admin/src/views/ClientManagement/components/VersionTable.vue`（加回滚按钮）
- 修改：`qim-admin/src/api/versions.ts`（加 API 封装）
- 修改：`qim-admin/src/stores/client.ts`（加 store 方法）

**关键代码：**

`version_handler.go`：
```go
func RollbackVersion(c *gin.Context) {
    id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
    svc := service.NewVersionService(database.GetDB())
    if err := svc.Rollback(uint(id)); err != nil {
        if errors.Is(err, service.ErrVersionNotFound) {
            response.NotFound(c, "版本不存在")
            return
        }
        response.InternalServerError(c, err.Error())
        return
    }
    response.Success(c, gin.H{"message": "回滚成功"})
}
```

`routes.go`：
```go
adminRoutes.POST("/client/versions/:id/rollback", handler.RollbackVersion)
```

`VersionTable.vue` 加回滚按钮（仅对已启用的非最新版本显示）：
```vue
<el-button
  v-if="row.status === 'active'"
  size="small"
  type="warning"
  @click="handleRollback(row)"
>
  回滚到此版本
</el-button>
```

**验证：** 测试回滚后 `GetLatestYML` 返回目标版本；管理员 UI 操作回滚按钮。

- [ ] **步骤 1：新增 handler RollbackVersion**
- [ ] **步骤 2：新增路由**
- [ ] **步骤 3：admin 前端加 API 封装和 store 方法**
- [ ] **步骤 4：VersionTable.vue 加回滚按钮和确认弹窗**
- [ ] **步骤 5：集成测试**
- [ ] **步骤 6：Commit**

```bash
git add qim-server/app/routes.go qim-server/handler/version_handler.go qim-admin/src/
git commit -m "feat(update): 新增版本回滚 API，禁用比目标版本新的所有版本"
```

---

### 任务 T11：增量更新（Type 字段启用）

**问题：** `qim-server/model/model.go:799` 定义了 `Type` 字段（full/patch），但更新逻辑始终按完整包处理，大版本浪费带宽。

**方案：** 支持 electron-updater 的 blockmap 增量更新。需在 latest.yml 输出 `.blockmap` 文件引用，管理员上传时同时上传 blockmap。

**说明：** 此任务较重，涉及打包流程改造（electron-builder 自动生成 blockmap，但需后端存储和分发），建议作为独立项目立项，不在本次整改范围内强制完成。本次仅：①在 latest.yml 中输出 blockmap 字段（若存在）；②模型补 `BlockmapURL` 字段。

- [ ] **步骤 1：模型补 BlockmapURL 字段**
- [ ] **步骤 2：latest.yml 输出 blockmap 字段（若存在）**
- [ ] **步骤 3：admin 前端支持 blockmap 上传**
- [ ] **步骤 4：独立立项完整增量更新方案**

---

### 任务 T12：主动推送（WebSocket）

**问题：** 完全靠客户端 4 小时轮询，紧急安全更新无法即时触达。

**方案：** 复用项目已有 WebSocket 连接，新增"版本发布"事件推送；客户端收到后立即触发 `checkForUpdates`。

**说明：** 此任务涉及 WS 通道消息类型扩展，建议作为独立项目立项。本次仅预留接口。

- [ ] **步骤 1：定义 WS 消息类型 `version_released`**
- [ ] **步骤 2：版本创建/启用时通过 Hub 广播**
- [ ] **步骤 3：客户端监听 `version_released` 触发 checkForUpdates**
- [ ] **步骤 4：独立立项完整主动推送方案**

---

## 自检结果

**1. 规格覆盖度：** 12 项整改全部有对应任务。T11/T12 标注为独立立项，本次仅做接口预留。

**2. 占位符扫描：** T4 的 `List/Update/Delete/ToggleStatus` 实现已展开完整代码，无占位符。T11/T12 明确标注为独立立项，不属于规格缺失。

**3. 类型一致性：**
- `CreateVersionInput`、`UpdateVersionInput`、`ErrVersionExists`、`ErrVersionNotFound`、`ErrMissingDownloadURL` 在 T4 定义，T5/T10 复用
- `RolloutPercentage` 字段名在模型（T3）、DDL（T3）、service（T4/T5）、前端（T5）保持一致
- `FilterByRollout` 在 T5 定义，T4 的 `GetLatestEnabled` 调用
- `LatestVersion` 在 T1 定义，T4 的 `GetLatestEnabled` 和 `Rollback` 复用
- `ComputeFileHash` 在 T2 定义，T4 的 `Create` 调用
- `VersionStat` 在 T6 定义，`version_handler.go` 复用
- `Rollback` 方法在 T4 定义，T10 路由调用

**4. 依赖关系：**
- T1 是基础，T4 依赖 T1
- T2 是基础，T4 依赖 T2
- T3 是基础，T4/T5 依赖 T3 的模型字段
- T4 依赖 T1/T2/T3 完成
- T5 依赖 T3（模型字段）和 T4（service 层）
- T6 独立，不依赖其他任务
- T7/T8 独立
- T9 独立
- T10 依赖 T4（Rollback 方法）
- T11/T12 独立立项

---

## 执行建议

**计划已保存。** 两种执行方式：

**1. 子代理驱动（推荐）** — 每个任务调度一个新的子代理，任务间审查，快速迭代。适合本次整改（任务多、相互独立）。

**2. 内联执行** — 在当前会话中按阶段批量执行，设检查点审查。

**建议执行顺序：**
- 阶段一（T1-T3）→ 阶段二（T4-T6）→ 阶段三（T7-T8）→ 阶段四（T9-T12 按需）

阶段四的 T11/T12 建议独立立项，本次仅完成 T9-T10。
