package service

import (
	"errors"
	"fmt"

	"github.com/dshmyz/qim/qim-server/model"
	"gorm.io/gorm"
)

// 版本相关错误
var (
	ErrVersionExists            = errors.New("该版本已存在")
	ErrVersionNotFound          = errors.New("版本不存在")
	ErrMissingDownloadURL       = errors.New("下载链接不能为空")
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
	FileSize          int64
}

// UpdateVersionInput 更新版本的入参（所有字段可选）
type UpdateVersionInput struct {
	Changelog         *string
	ForceUpdate       *bool
	Status            *string // "active" / "inactive"
	RolloutPercentage *int
	MinVersion        *string
}

type VersionService struct {
	db *gorm.DB
}

func NewVersionService(db *gorm.DB) *VersionService {
	return &VersionService{db: db}
}

// Create 创建版本，自动校验唯一性、计算哈希
func (s *VersionService) Create(input CreateVersionInput) (*model.ClientVersion, error) {
	// 版本号格式校验
	if err := ValidateVersionFormat(input.Version); err != nil {
		return nil, err
	}

	// 平台标准化
	platform := NormalizePlatform(input.Platform)

	// 唯一性校验
	var existing model.ClientVersion
	if err := s.db.Where("version = ? AND platform = ? AND deleted_at IS NULL",
		input.Version, platform).First(&existing).Error; err == nil {
		return nil, ErrVersionExists
	}

	rolloutPercentage, err := NormalizeRolloutPercentage(input.RolloutPercentage)
	if err != nil {
		return nil, err
	}

	version := model.ClientVersion{
		Version:           input.Version,
		Platform:          platform,
		Type:              "full",
		DownloadURL:       input.DownloadURL,
		Changelog:         input.Changelog,
		ForceUpdate:       input.ForceUpdate,
		RolloutPercentage: rolloutPercentage,
		MinVersion:        input.MinVersion,
		Enabled:           true,
	}

	// SHA512 和文件大小处理
	if input.Sha512 != "" && input.FileSize > 0 {
		version.Sha512 = input.Sha512
		version.FileSize = input.FileSize
	} else if input.DownloadURL != "" {
		hash, size, err := MustComputeFileHash(s.db, input.DownloadURL, platform)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrHashComputeFailed, err)
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

// List 分页查询版本列表
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

// ToggleStatus 切换版本启用状态
func (s *VersionService) ToggleStatus(id uint, enabled bool) error {
	version, err := s.GetByID(id)
	if err != nil {
		return err
	}
	version.Enabled = enabled
	return s.db.Save(version).Error
}

// GetLatestEnabled 获取平台最新已启用版本（semver 排序 + 灰度过滤）
// clientID 为空时仅放行 100% 全量版本
func (s *VersionService) GetLatestEnabled(platform, clientID string) (*model.ClientVersion, error) {
	platform = NormalizePlatform(platform)
	var versions []model.ClientVersion
	if err := s.db.Where("platform = ? AND enabled = ?", platform, true).Find(&versions).Error; err != nil {
		return nil, err
	}
	// 灰度过滤
	versions = FilterByRollout(versions, clientID)
	return LatestVersion(versions)
}

// Rollback 回滚到指定版本：禁用同平台比它新的所有已启用版本，启用目标版本
func (s *VersionService) Rollback(id uint) error {
	target, err := s.GetByID(id)
	if err != nil {
		return err
	}

	// 查询同平台所有已启用版本
	var newer []model.ClientVersion
	s.db.Where("platform = ? AND enabled = ? AND id != ?", target.Platform, true, id).Find(&newer)

	for _, v := range newer {
		if CompareVersions(v.Version, target.Version) > 0 {
			s.db.Model(&v).Update("enabled", false)
		}
	}

	// 启用目标版本
	return s.db.Model(target).Update("enabled", true).Error
}

// NormalizeRolloutPercentage 确保灰度百分比在 0-100 范围内；未传时默认 100。
func NormalizeRolloutPercentage(p *int) (int, error) {
	if p == nil {
		return 100, nil
	}
	if *p < 0 || *p > 100 {
		return 0, ErrInvalidRolloutPercentage
	}
	return *p, nil
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
