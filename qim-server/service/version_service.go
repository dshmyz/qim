package service

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"

	"github.com/dshmyz/qim/qim-server/model"
	"gorm.io/gorm"
)

// cliFileIDRe 从 DownloadURL 中提取文件 ID（"/files/:id/download"）。
var cliFileIDRe = regexp.MustCompile(`/files/(\d+)/download`)

// CLI/MCP 分发产物。同一张 ClientVersion 表用 app_type 区分不同产物，
// 避免 (app_type, version, platform) 唯一索引在 cli 与 mcp 间冲突。
// 产物标识只用 cli / mcp 两个值，不含任何品牌名；二进制实际文件名取自上传时的原名。
const (
	ProductCLI = "cli" // 命令行工具，app_type = "cli"
	ProductMCP = "mcp" // MCP Server，app_type = "mcp"

	legacyMCPAppType = "nuim-mcp" // 历史遗留 app_type，仅用于读取兼容，新写入一律用 "mcp"
)

// CLIAppType 返回指定产物在 ClientVersion.app_type 中的取值（默认 → "cli"）。
// 兼容历史别名：product 传入 "nuim-mcp" 或 "mcp" 均命中 mcp。
func CLIAppType(product string) string {
	if product == ProductMCP || product == legacyMCPAppType {
		return ProductMCP
	}
	return ProductCLI
}

// IsCLIAppType 判断该 app_type 是否为 CLI/MCP 家族产物
// （走 SHA256 + os/arch→platform 派生逻辑）。兼容历史 "nuim-mcp"。
func IsCLIAppType(appType string) bool {
	return appType == ProductCLI || appType == ProductMCP || appType == legacyMCPAppType
}

// cliAppTypes 返回指定产物应匹配的 app_type 取值列表。
// mcp 产物额外兼容历史遗留的 "nuim-mcp" app_type，避免老记录在改版后消失。
func cliAppTypes(product string) []string {
	if CLIAppType(product) == ProductMCP {
		return []string{ProductMCP, legacyMCPAppType}
	}
	return []string{ProductCLI}
}

// fallbackBinaryName 返回不含品牌前缀的规则名 "{os}-{arch}"（windows 加 .exe）。
// 仅在无法解析到上传原名时作为兜底。
func fallbackBinaryName(os, arch string) string {
	name := os + "-" + arch
	if os == "windows" {
		name += ".exe"
	}
	return name
}

// DownloadFilename 解析版本二进制的下载文件名。
// 优先取下载文件记录的上传原名（管理员上传时叫什么就叫什么）；
// 查不到文件/外部 URL 时回退到规则名 "{os}-{arch}"。
//
// 适用单次场景（如 CLIDownload）。批量场景（如 GetLatestCLIVersion 跨平台汇总）
// 请用 batchLoadFileOriginalNames 预加载后再调 downloadFilenameFromCache，避免 N+1。
func (s *VersionService) DownloadFilename(v *model.ClientVersion) string {
	if v.DownloadURL != "" {
		if matches := cliFileIDRe.FindStringSubmatch(v.DownloadURL); len(matches) == 2 {
			fileID, err := strconv.ParseUint(matches[1], 10, 32)
			if err == nil {
				var file model.File
				if err := s.db.Select("original_name").First(&file, uint(fileID)).Error; err == nil && file.OriginalName != "" {
					return file.OriginalName
				}
			}
		}
	}
	return fallbackBinaryName(v.Os, v.Arch)
}

// extractFileID 从 DownloadURL 中提取文件 ID；无匹配返回 0、false。
func extractFileID(downloadURL string) (uint, bool) {
	if downloadURL == "" {
		return 0, false
	}
	matches := cliFileIDRe.FindStringSubmatch(downloadURL)
	if len(matches) != 2 {
		return 0, false
	}
	id, err := strconv.ParseUint(matches[1], 10, 32)
	if err != nil {
		return 0, false
	}
	return uint(id), true
}

// batchLoadFileOriginalNames 批量查询给定版本列表引用的 File.original_name。
// 返回 fileID → original_name 的映射；查不到的 fileID 不出现在 map 中。
// 用一次 IN 查询替代每个版本单独 First，避免 N+1。
func (s *VersionService) batchLoadFileOriginalNames(versions []*model.ClientVersion) map[uint]string {
	ids := make(map[uint]struct{}, len(versions))
	for _, v := range versions {
		if id, ok := extractFileID(v.DownloadURL); ok {
			ids[id] = struct{}{}
		}
	}
	if len(ids) == 0 {
		return nil
	}
	list := make([]uint, 0, len(ids))
	for id := range ids {
		list = append(list, id)
	}
	var files []model.File
	if err := s.db.Select("id, original_name").Where("id IN ?", list).Find(&files).Error; err != nil {
		return nil
	}
	out := make(map[uint]string, len(files))
	for _, f := range files {
		if f.OriginalName != "" {
			out[f.ID] = f.OriginalName
		}
	}
	return out
}

// downloadFilenameFromCache 与 DownloadFilename 语义一致，但用预加载的 fileNames
// 替代 DB 查询；fileNames 为 nil 或未命中时回退到规则名（不再访问 DB）。
func downloadFilenameFromCache(v *model.ClientVersion, fileNames map[uint]string) string {
	if id, ok := extractFileID(v.DownloadURL); ok {
		if name, hit := fileNames[id]; hit {
			return name
		}
	}
	return fallbackBinaryName(v.Os, v.Arch)
}

// 版本相关错误
var (
	ErrVersionExists            = errors.New("该版本已存在")
	ErrVersionNotFound          = errors.New("版本不存在")
	ErrMissingDownloadURL       = errors.New("下载链接不能为空")
	ErrMissingSha256            = errors.New("SHA256 不能为空")
	ErrHashComputeFailed        = errors.New("SHA512 和文件大小计算失败")
	ErrInvalidRolloutPercentage = errors.New("灰度百分比必须在 0 到 100 之间")
)

// CreateVersionInput 创建版本的入参
type CreateVersionInput struct {
	Version           string
	Platform          string
	DownloadURL       string
	Changelog         string
	ForceUpdate       bool
	RolloutPercentage *int
	MinVersion        string
	Sha512            string
	Sha256            string // CLI 专用
	FileSize          int64
	// CLI 专用字段
	AppType string // "client" | "cli"，空值默认 "client"
	Os      string // CLI: darwin/linux/windows
	Arch    string // CLI: amd64/arm64
}

// UpdateVersionInput 更新版本的入参（所有字段可选）
type UpdateVersionInput struct {
	Changelog         *string
	ForceUpdate       *bool
	Status            *string // "active" / "inactive"
	RolloutPercentage *int
	MinVersion        *string
	DownloadURL       *string
	Sha256            *string
	FileSize          *int64
}

type VersionService struct {
	db      *gorm.DB
	storage StorageAccessor
}

func NewVersionService(db *gorm.DB, storage StorageAccessor) *VersionService {
	return &VersionService{db: db, storage: storage}
}

// Create 创建版本，自动校验唯一性、计算哈希
func (s *VersionService) Create(input CreateVersionInput) (*model.ClientVersion, error) {
	// 版本号格式校验
	if err := ValidateVersionFormat(input.Version); err != nil {
		return nil, err
	}

	appType := input.AppType
	if appType == "" {
		appType = "client"
	}

	// 平台标准化。CLI 可由 os/arch 派生，避免调用方漏填 platform 后创建出的版本无法查询。
	platform := input.Platform
	if appType == "client" {
		platform = NormalizePlatform(platform)
	} else if IsCLIAppType(appType) && platform == "" && input.Os != "" && input.Arch != "" {
		platform = input.Os + "-" + input.Arch
	}

	// 唯一性校验（app_type + version + platform）
	var existing model.ClientVersion
	if err := s.db.Where("app_type = ? AND version = ? AND platform = ? AND deleted_at IS NULL",
		appType, input.Version, platform).First(&existing).Error; err == nil {
		return nil, ErrVersionExists
	}

	rolloutPercentage, err := NormalizeRolloutPercentage(input.RolloutPercentage)
	if err != nil {
		return nil, err
	}

	version := model.ClientVersion{
		AppType:           appType,
		Version:           input.Version,
		Platform:          platform,
		Type:              "full",
		DownloadURL:       input.DownloadURL,
		Changelog:         input.Changelog,
		ForceUpdate:       input.ForceUpdate,
		RolloutPercentage: rolloutPercentage,
		MinVersion:        input.MinVersion,
		Enabled:           true,
		Os:                input.Os,
		Arch:              input.Arch,
	}

	if IsCLIAppType(appType) {
		if input.DownloadURL == "" {
			return nil, ErrMissingDownloadURL
		}
		if input.Sha256 == "" {
			return nil, ErrMissingSha256
		}
		// CLI 版本：使用 SHA256
		version.Sha256 = input.Sha256
		if input.FileSize > 0 {
			version.FileSize = input.FileSize
		}
	} else {
		// Client 版本：使用 SHA512，自动计算
		if input.Sha512 != "" && input.FileSize > 0 {
			version.Sha512 = input.Sha512
			version.FileSize = input.FileSize
		} else if input.DownloadURL != "" {
			hash, size, err := MustComputeFileHash(s.db, s.storage, input.DownloadURL, platform)
			if err != nil {
				return nil, fmt.Errorf("%w: %v", ErrHashComputeFailed, err)
			}
			version.Sha512 = hash
			version.FileSize = size
		} else {
			return nil, ErrMissingDownloadURL
		}
	}

	if err := s.db.Create(&version).Error; err != nil {
		return nil, err
	}
	return &version, nil
}

// List 分页查询版本列表（可按 appType 和 platform 过滤）
func (s *VersionService) List(page, pageSize int, platform string, appType ...string) ([]model.ClientVersion, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	query := s.db.Model(&model.ClientVersion{})
	// appType 可传多个：IN 匹配（单值退化为 =，行为不变）
	var apps []string
	for _, a := range appType {
		if a != "" {
			apps = append(apps, a)
		}
	}
	if len(apps) > 0 {
		query = query.Where("app_type IN ?", apps)
	}
	if platform != "" {
		query = query.Where("platform = ?", platform)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var versions []model.ClientVersion
	offset := (page - 1) * pageSize
	err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&versions).Error
	return versions, total, err
}

// GetByID 按 ID 查询版本
func (s *VersionService) GetByID(id uint) (*model.ClientVersion, error) {
	var version model.ClientVersion
	if err := s.db.First(&version, id).Error; err != nil {
		return nil, ErrVersionNotFound
	}
	return &version, nil
}

// Update 更新版本信息
func (s *VersionService) Update(id uint, input UpdateVersionInput) (*model.ClientVersion, error) {
	version, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	if input.Changelog != nil {
		version.Changelog = *input.Changelog
	}
	if input.ForceUpdate != nil {
		version.ForceUpdate = *input.ForceUpdate
	}
	if input.Status != nil {
		version.Enabled = *input.Status == "active"
	}
	if err := ApplyRolloutPercentageUpdate(version, input.RolloutPercentage); err != nil {
		return nil, err
	}
	if input.MinVersion != nil {
		version.MinVersion = *input.MinVersion
	}
	if input.DownloadURL != nil {
		version.DownloadURL = *input.DownloadURL
	}
	if input.Sha256 != nil {
		version.Sha256 = *input.Sha256
	}
	if input.FileSize != nil {
		version.FileSize = *input.FileSize
	}

	if err := s.db.Save(version).Error; err != nil {
		return nil, err
	}
	return version, nil
}

// Delete 删除版本（软删除）
func (s *VersionService) Delete(id uint) error {
	version, err := s.GetByID(id)
	if err != nil {
		return err
	}
	return s.db.Delete(version).Error
}

// ToggleStatus 切换版本启用状态，返回更新后的版本对象。
// 返回版本对象可让 handler 无需二次 GetByID，避免并发删除时 nil 解引用 panic。
func (s *VersionService) ToggleStatus(id uint, enabled bool) (*model.ClientVersion, error) {
	version, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}
	version.Enabled = enabled
	if err := s.db.Save(version).Error; err != nil {
		return nil, err
	}
	return version, nil
}

// GetLatestEnabled 获取平台最新已启用版本（semver 排序 + 灰度过滤）
// clientID 为空时仅放行 100% 全量版本
func (s *VersionService) GetLatestEnabled(platform, clientID string) (*model.ClientVersion, error) {
	platform = NormalizePlatform(platform)
	var versions []model.ClientVersion
	if err := s.db.Where("app_type = ? AND platform = ? AND enabled = ?", "client", platform, true).Find(&versions).Error; err != nil {
		return nil, err
	}
	// 灰度过滤
	versions = FilterByRollout(versions, clientID)
	return LatestVersion(versions)
}

// GetLatestCLI 获取指定 os/arch 的最新已启用 CLI/MCP 版本。
// product 指定产物（cli | mcp），空值默认 cli。
func (s *VersionService) GetLatestCLI(os, arch, product string) (*model.ClientVersion, error) {
	platform := os + "-" + arch
	var versions []model.ClientVersion
	if err := s.db.Where("app_type IN ? AND platform = ? AND enabled = ?", cliAppTypes(product), platform, true).Find(&versions).Error; err != nil {
		return nil, err
	}
	return LatestVersion(versions)
}

// GetLatestCLIVersion 获取指定产物最新 CLI 版本号（跨所有平台，用于版本查询端点）。
func (s *VersionService) GetLatestCLIVersion(product string) (string, map[string]string, error) {
	var versions []model.ClientVersion
	if err := s.db.Where("app_type IN ? AND enabled = ?", cliAppTypes(product), true).Find(&versions).Error; err != nil {
		return "", nil, err
	}
	if len(versions) == 0 {
		return "", nil, nil
	}

	latest, err := LatestVersion(versions)
	if err != nil {
		return "", nil, err
	}

	// 每个平台只取最新版本的 sha256，避免同平台多版本启用时因 map 迭代顺序不确定导致
	// 返回的哈希与 CLIDownload（返回最新版本二进制）不匹配，客户端校验失败。
	latestByPlatform := make(map[string]*model.ClientVersion) // platform → 该平台最新版本
	for i := range versions {
		v := &versions[i]
		if v.Sha256 == "" {
			continue
		}
		cur, ok := latestByPlatform[v.Platform]
		if !ok || CompareVersions(v.Version, cur.Version) > 0 {
			latestByPlatform[v.Platform] = v
		}
	}

	// 批量预加载各版本引用的 File.original_name，避免每个平台一次 DB 查询（N+1）。
	// 此接口为 /api/v1/cli/version 公开端点，可能被高频探测，必须避免线性 DB 抖动。
	latestList := make([]*model.ClientVersion, 0, len(latestByPlatform))
	for _, v := range latestByPlatform {
		latestList = append(latestList, v)
	}
	fileNames := s.batchLoadFileOriginalNames(latestList)

	// 构建 sha256 map，key 取该版本二进制上传时的原名
	// （如 "mcp-darwin-arm64" / "cli-darwin-amd64.exe"，由管理员上传文件时的名字决定）。
	sha256Map := make(map[string]string, len(latestByPlatform))
	for _, v := range latestByPlatform {
		sha256Map[downloadFilenameFromCache(v, fileNames)] = v.Sha256
	}

	return latest.Version, sha256Map, nil
}

// Rollback 回滚到指定版本：禁用同平台比它新的所有已启用版本，启用目标版本
func (s *VersionService) Rollback(id uint) error {
	target, err := s.GetByID(id)
	if err != nil {
		return err
	}

	// 查询同 app_type + 平台所有已启用版本
	var newer []model.ClientVersion
	s.db.Where("app_type = ? AND platform = ? AND enabled = ? AND id != ?",
		target.AppType, target.Platform, true, id).Find(&newer)

	for _, v := range newer {
		if CompareVersions(v.Version, target.Version) > 0 {
			s.db.Model(&v).Update("enabled", false)
		}
	}

	// 启用目标版本
	return s.db.Model(target).Update("enabled", true).Error
}

// NormalizeRolloutPercentage 确保灰度百分比在 0-100 范围内；未传时默认 100。
// 返回 *int 以便 GORM 把显式 0 写入 DB（int 零值会被 GORM 省略而触发 default:100）。
func NormalizeRolloutPercentage(p *int) (*int, error) {
	if p == nil {
		defaultValue := 100
		return &defaultValue, nil
	}
	if *p < 0 || *p > 100 {
		return nil, ErrInvalidRolloutPercentage
	}
	return p, nil
}

func ApplyRolloutPercentageUpdate(version *model.ClientVersion, p *int) error {
	if p == nil {
		return nil
	}
	rolloutPercentage, err := NormalizeRolloutPercentage(p)
	if err != nil {
		return err
	}
	version.RolloutPercentage = rolloutPercentage
	return nil
}
