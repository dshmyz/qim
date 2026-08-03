package storage

import "strings"

// StaticPrefix 是静态资源路由的统一前缀。
// 所有存储路径（StoragePath）都以 /static/ 开头，前端通过 serverUrl + storagePath 访问。
const StaticPrefix = "/static/"

// ParsePath 从 storagePath 中解析出存储 key。
// 兼容三种历史格式：
//   - /static/xxx        → 新格式，key=xxx
//   - /s3/uploads/xxx     → 旧S3格式，key=uploads/xxx
//   - /uploads/xxx        → 旧本地格式，key=uploads/xxx
//
// 不再通过前缀区分存储类型——系统只有一种默认存储（见 Manager.ByPath）。
func ParsePath(storagePath string) (kind, key string) {
	// 新格式
	if strings.HasPrefix(storagePath, StaticPrefix) {
		return "", strings.TrimPrefix(storagePath, StaticPrefix)
	}
	// 旧S3格式
	if strings.HasPrefix(storagePath, "/s3/") {
		return "s3", strings.TrimPrefix(storagePath, "/s3/")
	}
	// 旧本地格式
	if strings.HasPrefix(storagePath, "/") {
		return "local", strings.TrimPrefix(storagePath, "/")
	}
	return "local", storagePath
}

// BuildPath 生成统一的存储路径。
// 不再根据存储类型加不同前缀，统一返回 /static/key。
func BuildPath(kind, key string) string {
	return StaticPrefix + key
}

// ToNewKey 从 storagePath 提取存储 key。
func ToNewKey(storagePath string) string {
	_, key := ParsePath(storagePath)
	return key
}

// FromNewKey 生成存储路径。
func FromNewKey(kind, key string) string {
	return BuildPath(kind, key)
}

// NeedsMigration 判断 storagePath 是否为旧格式（需要迁移）。
func NeedsMigration(storagePath string) bool {
	return strings.HasPrefix(storagePath, "/s3/") ||
		(strings.HasPrefix(storagePath, "/uploads/") && !strings.HasPrefix(storagePath, StaticPrefix))
}

// MigratePath 将旧格式 storagePath 转为新格式。
// /s3/uploads/xxx → /static/uploads/xxx
// /uploads/xxx    → /static/uploads/xxx
// /static/xxx     → /static/xxx（不变）
func MigratePath(storagePath string) string {
	if strings.HasPrefix(storagePath, StaticPrefix) {
		return storagePath
	}
	if strings.HasPrefix(storagePath, "/s3/") {
		return StaticPrefix + strings.TrimPrefix(storagePath, "/s3/")
	}
	if strings.HasPrefix(storagePath, "/") {
		return StaticPrefix + strings.TrimPrefix(storagePath, "/")
	}
	return StaticPrefix + storagePath
}
