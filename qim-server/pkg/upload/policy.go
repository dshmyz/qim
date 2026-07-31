// Package upload 提供统一的文件上传安全策略。
// 所有上传入口（普通上传、分片上传、反馈截图、CSV 导入）都应通过本包校验，
// 确保大小限制、类型校验、文件名清洗、MIME 检测策略一致。
package upload

import (
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
)

// 默认限制
const (
	DefaultMaxSize          = 500 * 1024 * 1024 // 500MB
	DefaultChunkMaxSize     = 10 * 1024 * 1024  // 10MB 单分片
	DefaultImportMaxSize    = 10 * 1024 * 1024  // 10MB CSV 导入
	DefaultScreenshotMaxSize = 5 * 1024 * 1024  // 5MB 截图
)

// blockedExtensions 是无论配置如何都禁止上传的扩展名（可执行/可渲染）。
// 防止通过上传 .html/.svg 等可渲染文件配合 inline 预览造成存储型 XSS，
// 以及防止 .exe 等可执行文件在内部 IM 中传播木马。
var blockedExtensions = map[string]bool{
	".html": true, ".htm": true, ".svg": true, ".js": true, ".mjs": true,
	".exe": true, ".msi": true, ".bat": true, ".cmd": true, ".sh": true,
	".php": true, ".jsp": true, ".asp": true, ".aspx": true,
}

// dangerousMimePrefixes 是通过 Content-Type 判断为危险的 MIME 前缀。
var dangerousMimePrefixes = []string{
	"text/html",
	"application/javascript",
	"application/x-msdownload",
	"application/x-sh",
	"application/x-php",
}

// ValidateError 描述校验失败的类型，便于 handler 返回不同的 HTTP 状态码。
type ValidateError struct {
	Field   string // size / type / filename / disabled
	Message string
}

func (e *ValidateError) Error() string { return e.Message }

var (
	// ErrUploadDisabled 文件上传功能已被管理员关闭。
	ErrUploadDisabled = &ValidateError{Field: "disabled", Message: "文件上传功能已关闭"}
	// ErrFileTooLarge 文件大小超过限制。
	ErrFileTooLarge = &ValidateError{Field: "size", Message: "文件大小超过限制"}
	// ErrFileTypeBlocked 文件类型被禁止上传。
	ErrFileTypeBlocked = &ValidateError{Field: "type", Message: "该文件类型不允许上传"}
	// ErrInvalidFilename 文件名非法。
	ErrInvalidFilename = &ValidateError{Field: "filename", Message: "文件名非法"}
)

// Policy 统一上传策略。所有上传入口应使用同一个 Policy 实例。
type Policy struct {
	MaxSize           int64
	AllowedExtensions map[string]bool
	EnableTypeCheck   bool
}

// NewPolicy 创建上传策略。参数为零值时使用安全默认值。
func NewPolicy(maxSize int64, allowedExtensions map[string]bool, enableTypeCheck bool) *Policy {
	if maxSize <= 0 {
		maxSize = DefaultMaxSize
	}
	if allowedExtensions == nil {
		allowedExtensions = defaultAllowedExtensions()
	}
	return &Policy{
		MaxSize:           maxSize,
		AllowedExtensions: allowedExtensions,
		EnableTypeCheck:   enableTypeCheck,
	}
}

// defaultAllowedExtensions 返回安全默认白名单（不含可执行文件）。
func defaultAllowedExtensions() map[string]bool {
	return map[string]bool{
		".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".bmp": true, ".webp": true,
		".pdf": true,
		".doc": true, ".docx": true,
		".xls": true, ".xlsx": true,
		".ppt": true, ".pptx": true,
		".txt": true, ".md": true, ".csv": true,
		".zip": true, ".rar": true, ".7z": true,
		".mp3": true, ".wav": true, ".mp4": true, ".avi": true, ".mov": true,
	}
}

// ValidateSize 校验文件大小是否在限制范围内。
func (p *Policy) ValidateSize(size int64) error {
	if size > p.MaxSize {
		return ErrFileTooLarge
	}
	return nil
}

// ValidateType 校验文件类型。先查黑名单，再查白名单（若开启）。
// filename 用于提取扩展名，detectedMime 用于辅助判断。
func (p *Policy) ValidateType(filename string, detectedMime string) error {
	ext := strings.ToLower(filepath.Ext(filename))

	// 1. 黑名单始终生效：禁止可执行/可渲染文件
	if blockedExtensions[ext] {
		return ErrFileTypeBlocked
	}

	// 2. MIME 黑名单：禁止危险的 MIME 类型
	if isDangerousMime(detectedMime) {
		return ErrFileTypeBlocked
	}

	// 3. 白名单校验（若开启）
	if p.EnableTypeCheck && !p.AllowedExtensions[ext] {
		return ErrFileTypeBlocked
	}

	return nil
}

// isDangerousMime 判断 MIME 是否危险。
func isDangerousMime(mime string) bool {
	mime = strings.ToLower(strings.TrimSpace(mime))
	// 去除 charset 等参数
	if idx := strings.Index(mime, ";"); idx >= 0 {
		mime = strings.TrimSpace(mime[:idx])
	}
	for _, prefix := range dangerousMimePrefixes {
		if strings.HasPrefix(mime, prefix) {
			return true
		}
	}
	return false
}

// SanitizeFilename 清洗用户提交的文件名，防止路径遍历和特殊字符注入。
// 1. 只取 basename（去掉路径）
// 2. 替换路径分隔符和特殊字符
// 3. 限制长度
// 返回清洗后的纯文件名（不含路径）。
func SanitizeFilename(filename string) string {
	if filename == "" {
		return "unnamed"
	}
	// 统一路径分隔符后取 basename，防止 ../ 逃逸
	cleaned := filepath.Base(filepath.Clean(strings.ReplaceAll(filename, "\\", "/")))
	// 替换剩余的特殊字符
	re := regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)
	cleaned = re.ReplaceAllString(cleaned, "_")
	// 防止以 . 开头（隐藏文件）或 .. 结尾
	cleaned = strings.TrimLeft(cleaned, ".")
	if cleaned == "" || cleaned == "." || cleaned == ".." {
		return "unnamed"
	}
	// 限制长度（保留扩展名）
	if len(cleaned) > 200 {
		ext := filepath.Ext(cleaned)
		name := strings.TrimSuffix(cleaned, ext)
		maxNameLen := 200 - len(ext)
		if maxNameLen > 0 {
			cleaned = name[:maxNameLen] + ext
		} else {
			cleaned = cleaned[:200]
		}
	}
	return cleaned
}

// DetectMimeType 通过读取文件头前 512 字节检测真实 MIME 类型。
// 不信任客户端提交的 Content-Type Header。
func DetectMimeType(data []byte) string {
	if len(data) == 0 {
		return "application/octet-stream"
	}
	// http.DetectContentType 最多读取前 512 字节
	sample := data
	if len(sample) > 512 {
		sample = sample[:512]
	}
	return http.DetectContentType(sample)
}

// ShouldForceDownload 判断文件是否应强制下载（而非 inline 渲染）。
// 可渲染的危险类型（html/svg/js等）在浏览器中直接展示会导致存储型 XSS；
// 可执行类型（exe/bat/sh等）也不应 inline 展示。
// 即便已通过上传黑名单拦截，仍应对历史数据或外部写入的文件强制下载。
func ShouldForceDownload(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return blockedExtensions[ext]
}

// IsUploadEnabled 检查文件上传功能是否已启用。
// enableFunc 应返回系统配置中 enableFileUpload 的值。
func IsUploadEnabled(enableFunc func() (string, error)) bool {
	if enableFunc == nil {
		return true
	}
	val, err := enableFunc()
	if err != nil {
		return true // 配置读取失败时默认允许，不因配置故障阻断业务
	}
	return val != "false"
}
