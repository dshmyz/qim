package handler

import (
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/logger"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
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

func defaultUpdateFilename(version model.ClientVersion) string {
	ext := ".zip"
	switch normalizePlatform(version.Platform) {
	case "macos":
		ext = ".dmg"
	case "windows":
		ext = ".exe"
	case "linux":
		ext = ".AppImage"
	}
	return fmt.Sprintf("QIM-%s%s", version.Version, ext)
}

func safeUpdatePathFilename(db *gorm.DB, version model.ClientVersion) string {
	if strings.Contains(version.DownloadURL, "/api/v1/public/files/") && strings.HasSuffix(version.DownloadURL, "/download") {
		parts := strings.Split(version.DownloadURL, "/")
		for i, part := range parts {
			if part == "files" && i+1 < len(parts) {
				if fileID, err := strconv.ParseUint(parts[i+1], 10, 32); err == nil {
					var file model.File
					if err := db.First(&file, uint(fileID)).Error; err == nil {
						if file.OriginalName != "" {
							return filepath.Base(file.OriginalName)
						}
						if file.Name != "" {
							return filepath.Base(file.Name)
						}
					}
				}
				break
			}
		}
	}

	if filename := getFilenameFromURL(version.DownloadURL); filename != "" && filename != "." && filename != "/" && filename != "download" {
		return filepath.Base(filename)
	}

	return defaultUpdateFilename(version)
}

func absoluteUpdateURL(c *gin.Context, downloadURL string) string {
	if isURL(downloadURL) {
		return downloadURL
	}
	if strings.HasPrefix(downloadURL, "/") {
		scheme := c.GetHeader("X-Forwarded-Proto")
		if scheme == "" {
			if c.Request.TLS != nil {
				scheme = "https"
			} else {
				scheme = "http"
			}
		}
		return fmt.Sprintf("%s://%s%s", scheme, c.Request.Host, downloadURL)
	}
	return downloadURL
}

// HandleUpdateRequest 统一处理更新请求
// GET /api/v1/updates/:platform/*action
func HandleUpdateRequest(c *gin.Context) {
	action := c.Param("action")
	platform := c.Param("platform")
	log.Printf("HandleUpdateRequest called: platform=%s, action=%s", platform, action)
	// action 格式: /latest.yml, /latest-mac.yml
	action = strings.TrimPrefix(action, "/")

	if strings.HasPrefix(action, "latest") && strings.HasSuffix(action, ".yml") {
		GetLatestYML(c)
	} else {
		RedirectUpdateFile(c, platform, action)
	}
}

func RedirectUpdateFile(c *gin.Context, platformParam string, filename string) {
	platform := normalizePlatform(platformParam)
	db := database.GetDB()
	var version model.ClientVersion
	if err := db.Where("platform = ? AND enabled = ?", platform, true).
		Order("created_at DESC").First(&version).Error; err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	if filename != safeUpdatePathFilename(db, version) {
		c.Status(http.StatusNotFound)
		return
	}

	c.Redirect(http.StatusFound, absoluteUpdateURL(c, version.DownloadURL))
}

// GetLatestYML 返回 electron-updater 需要的 latest.yml 格式
// GET /api/v1/updates/:platform/latest.yml
func GetLatestYML(c *gin.Context) {
	platformParam := c.Param("platform")
	platform := normalizePlatform(platformParam)

	logger.WithModule("Update").Info("检查更新请求",
		"platform_param", platformParam,
		"platform", platform,
		"client_ip", c.ClientIP(),
	)

	db := database.GetDB()
	var version model.ClientVersion
	err := db.Where("platform = ? AND enabled = ?", platform, true).
		Order("created_at DESC").First(&version).Error
	if err != nil {
		// 区分"无记录"和"数据库错误"
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.WithModule("Update").Warn("无可用版本记录",
				"platform", platform,
				"platform_param", platformParam,
			)
			// electron-updater 期望 404 时返回空内容，而不是 JSON
			// 这样它会触发 update-not-available 事件
			c.Status(http.StatusNotFound)
		} else {
			logger.WithModule("Update").Error("查询版本失败",
				"platform", platform,
				"error", err,
			)
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "查询更新失败",
			})
		}
		return
	}

	logger.WithModule("Update").Info("找到版本记录",
		"version", version.Version,
		"platform", version.Platform,
		"download_url", version.DownloadURL,
	)

	downloadURL := version.DownloadURL
	sha512Hash := version.Sha512
	fileSize := version.FileSize

	if downloadURL == "" || sha512Hash == "" || fileSize <= 0 {
		logger.WithModule("Update").Warn("版本元数据不完整，拒绝输出 latest.yml",
			"version", version.Version,
			"platform", version.Platform,
			"download_url", downloadURL,
			"has_sha512", sha512Hash != "",
			"file_size", fileSize,
		)
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	// 生成 latest.yml
	// electron-updater generic provider 要求：
	// 1. sha512 使用 base64 编码
	// 2. files.url 使用相对于 feed URL 的路径
	sha512Base64 := ""
	if sha512Hash != "" {
		// hex 转 base64
		hashBytes, err := hex.DecodeString(sha512Hash)
		if err == nil {
			sha512Base64 = base64.StdEncoding.EncodeToString(hashBytes)
		}
	}

	// electron-updater 使用 files.url 生成本地缓存文件名。
	// 这里必须输出扁平安装包文件名，不能输出绝对 URL 或 /api/v1/.../download 这类多级路径。
	// 实际下载由 /api/v1/updates/:platform/:filename 重定向到真实下载地址。
	updatePathName := safeUpdatePathFilename(db, version)

	forceUpdateStr := "false"
	if version.ForceUpdate {
		forceUpdateStr = "true"
	}

	yml := fmt.Sprintf(`version: %s
files:
  - url: %s
    sha512: %s
    size: %d
path: %s
sha512: %s
releaseDate: %s
releaseNotes: %s
forceUpdate: %s
`,
		version.Version,
		updatePathName,
		sha512Base64,
		fileSize,
		updatePathName,
		sha512Base64,
		version.CreatedAt.Format("2006-01-02T15:04:05.000Z"),
		formatYAMLBlockString(version.Changelog),
		forceUpdateStr,
	)

	c.Header("Content-Type", "text/yaml")
	c.String(http.StatusOK, yml)
}

func formatYAMLBlockString(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	value = strings.TrimRight(value, "\n")
	if value == "" {
		return "''"
	}

	lines := strings.Split(value, "\n")
	for i, line := range lines {
		lines[i] = "  " + line
	}
	return "|-\n" + strings.Join(lines, "\n")
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
	// 如果是相对路径，直接使用
	// 如果是绝对路径，也直接使用
	file, err := os.Open(filePath)
	if err != nil {
		log.Printf("打开文件失败: %s, error: %v", filePath, err)
		return "", 0
	}
	defer file.Close()

	hash := sha512.New()
	size, err := io.Copy(hash, file)
	if err != nil {
		log.Printf("计算哈希失败: %s, error: %v", filePath, err)
		return "", 0
	}

	// electron-updater 需要 base64 格式的 SHA512
	sum := hash.Sum(nil)
	return fmt.Sprintf("%x", sum), size
}

// CheckUpdateHealth 检查更新服务健康状态
// GET /api/v1/updates/health
func CheckUpdateHealth(c *gin.Context) {
	db := database.GetDB()

	// 统计各平台的可用版本数
	var stats []struct {
		Platform string
		Count    int64
	}

	db.Model(&model.ClientVersion{}).
		Select("platform, count(*) as count").
		Where("enabled = ?", true).
		Group("platform").
		Scan(&stats)

	// 转换为 map
	platformStats := make(map[string]int64)
	for _, stat := range stats {
		platformStats[stat.Platform] = stat.Count
	}

	c.JSON(http.StatusOK, gin.H{
		"status":              "ok",
		"platform_stats":      platformStats,
		"supported_platforms": []string{"windows", "macos", "linux"},
		"timestamp":           time.Now().Unix(),
	})
}
