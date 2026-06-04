package handler

import (
	"crypto/sha512"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/model"

	"github.com/gin-gonic/gin"
)

// platformAliasMap 平台别名映射，统一不同平台标识到数据库存储的标准平台名
var platformAliasMap = map[string]string{
	"win":     "windows",
	"win7":    "windows",
	"win10":   "windows",
	"windows": "windows",
	"mac":     "macos",
	"macos":   "macos",
	"linux":   "linux",
}

// normalizePlatform 将客户端传入的平台标识标准化
func normalizePlatform(platform string) string {
	if mapped, ok := platformAliasMap[strings.ToLower(platform)]; ok {
		return mapped
	}
	return strings.ToLower(platform)
}

// isURL 判断字符串是否为URL
func isURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

// getFilenameFromURL 从URL中提取文件名
func getFilenameFromURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return filepath.Base(parsed.Path)
}

// GetLatestYML 返回 electron-updater 需要的 latest.yml 格式
// GET /api/v1/updates/:platform/latest.yml
func GetLatestYML(c *gin.Context) {
	platformParam := c.Param("platform")
	platform := normalizePlatform(platformParam)

	db := database.GetDB()
	var version model.ClientVersion
	err := db.Where("platform = ? AND enabled = ?", platform, true).
		Order("created_at DESC").First(&version).Error
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	// 处理下载链接：如果是URL则直接使用，否则作为本地文件路径
	downloadURL := version.DownloadURL
	sha512Hash := ""
	fileSize := int64(0)

	if downloadURL != "" {
		if isURL(downloadURL) {
			// URL模式：尝试计算本地缓存文件的SHA512
			filename := getFilenameFromURL(downloadURL)
			localPath := filepath.Join("./uploads/updates", platform, filename)
			if _, err := os.Stat(localPath); err == nil {
				sha512Hash, fileSize = computeFileSHA512(localPath)
			}
		} else {
			// 本地文件路径模式
			sha512Hash, fileSize = computeFileSHA512(downloadURL)
		}
	}

	// 生成 latest.yml
	yml := fmt.Sprintf(`version: %s
files:
  - url: %s
    sha512: %s
    size: %d
path: %s
sha512: %s
releaseDate: %s
`,
		version.Version,
		downloadURL,
		sha512Hash,
		fileSize,
		downloadURL,
		sha512Hash,
		version.CreatedAt.Format("2006-01-02T15:04:05.000Z"),
	)

	c.Header("Content-Type", "text/yaml")
	c.String(http.StatusOK, yml)
}

// GetUpdateFile 下载更新文件
// GET /api/v1/updates/:platform/files/:filename
func GetUpdateFile(c *gin.Context) {
	platform := c.Param("platform")
	filename := c.Param("filename")

	// 构建文件路径
	baseDir := "./uploads/updates"
	filePath := filepath.Join(baseDir, platform, filename)

	// 安全检查：防止目录穿越
	absPath, err := filepath.Abs(filePath)
	if err != nil || absPath != filepath.Clean(absPath) || !filepath.HasPrefix(absPath, filepath.Join(baseDir, platform)) {
		c.Status(http.StatusForbidden)
		return
	}

	// 检查文件是否存在
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		c.Status(http.StatusNotFound)
		return
	}

	c.File(absPath)
}

// computeFileSHA512 计算文件的 SHA512 哈希和大小
func computeFileSHA512(filePath string) (string, int64) {
	// 如果是完整路径直接使用，否则拼接 uploads 目录
	if !filepath.IsAbs(filePath) {
		filePath = filepath.Join("./uploads/updates", filePath)
	}

	file, err := os.Open(filePath)
	if err != nil {
		return "", 0
	}
	defer file.Close()

	hash := sha512.New()
	size, err := io.Copy(hash, file)
	if err != nil {
		return "", 0
	}

	// electron-updater 需要 base64 格式的 SHA512
	sum := hash.Sum(nil)
	return fmt.Sprintf("%x", sum), size
}
