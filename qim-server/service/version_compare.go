package service

import (
	"errors"
	"fmt"

	"github.com/dshmyz/qim/qim-server/model"
	"golang.org/x/mod/semver"
)

// IsValidVersion 校验版本号是否符合 semver
// 前端要求格式 \d+\.\d+\.\d+，semver 库要求 "v" 前缀，内部自动补
func IsValidVersion(v string) bool {
	return semver.IsValid("v" + v)
}

// CompareVersions 比较 semver 版本号
// a > b 返回 1，a < b 返回 -1，相等返回 0
func CompareVersions(a, b string) int {
	return semver.Compare("v"+a, "v"+b)
}

// LatestVersion 从版本列表中选出 semver 最大的
// 修复原先按 created_at DESC 排序导致的补录旧版本误判问题
func LatestVersion(versions []model.ClientVersion) (*model.ClientVersion, error) {
	if len(versions) == 0 {
		return nil, errors.New("无可用版本")
	}
	latest := &versions[0]
	for i := 1; i < len(versions); i++ {
		if CompareVersions(versions[i].Version, latest.Version) > 0 {
			latest = &versions[i]
		}
	}
	return latest, nil
}

// NormalizePlatform 将客户端传入的平台标识标准化
// win/win7/win10 → windows, mac → macos
func NormalizePlatform(platform string) string {
	return normalizePlatformImpl(platform)
}

// normalizePlatformImpl 被 NormalizePlatform 调用，具体实现在 handler 包中通过注入
// 为避免循环依赖，这里自行实现一份
func normalizePlatformImpl(platform string) string {
	switch lower := toLower(platform); lower {
	case "win", "win7", "win10", "windows":
		return "windows"
	case "mac", "macos":
		return "macos"
	case "linux":
		return "linux"
	default:
		return lower
	}
}

func toLower(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}

// ValidateVersionFormat 校验版本号格式，返回 error 描述具体问题
func ValidateVersionFormat(v string) error {
	if v == "" {
		return fmt.Errorf("版本号不能为空")
	}
	if !IsValidVersion(v) {
		return fmt.Errorf("版本号格式无效: %s（应为 x.y.z 格式，如 2.1.0）", v)
	}
	return nil
}
