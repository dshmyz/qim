package service

import (
	"errors"
	"fmt"
	"strings"

	"github.com/dshmyz/qim/qim-server/model"
	"gorm.io/gorm"
)

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
	} else if appType == "cli" && platform == "" && input.Os != "" && input.Arch != "" {
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

	if appType == "cli" {
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
	if len(appType) > 0 && appType[0] != "" {
		query = query.Where("app_type = ?", appType[0])
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

// GetLatestCLI 获取指定 os/arch 的最新已启用 CLI 版本
func (s *VersionService) GetLatestCLI(os, arch string) (*model.ClientVersion, error) {
	platform := os + "-" + arch
	var versions []model.ClientVersion
	if err := s.db.Where("app_type = ? AND platform = ? AND enabled = ?", "cli", platform, true).Find(&versions).Error; err != nil {
		return nil, err
	}
	return LatestVersion(versions)
}

// GetLatestCLIVersion 获取最新 CLI 版本号（跨所有平台，用于版本查询端点）
func (s *VersionService) GetLatestCLIVersion() (string, map[string]string, error) {
	var versions []model.ClientVersion
	if err := s.db.Where("app_type = ? AND enabled = ?", "cli", true).Find(&versions).Error; err != nil {
		return "", nil, err
	}
	if len(versions) == 0 {
		return "", nil, nil
	}

	latest, err := LatestVersion(versions)
	if err != nil {
		return "", nil, err
	}

	// 构建 sha256 map: {"qim-darwin-arm64": "abc...", ...}
	// 每个平台只取最新版本的 sha256，避免同平台多版本启用时因 map 迭代顺序不确定导致
	// 返回的哈希与 CLIDownload（返回最新版本二进制）不匹配，客户端校验失败。
	sha256Map := make(map[string]string)
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
	for _, v := range latestByPlatform {
		binaryName := fmt.Sprintf("qim-%s", v.Platform)
		if v.Os == "windows" || strings.HasPrefix(v.Platform, "windows-") {
			binaryName += ".exe"
		}
		sha256Map[binaryName] = v.Sha256
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
