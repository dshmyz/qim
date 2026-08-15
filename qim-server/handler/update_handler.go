package handler

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/dshmyz/qim/qim-server/cache"
	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/logger"
	"github.com/dshmyz/qim/qim-server/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

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
	switch service.NormalizePlatform(version.Platform) {
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

// 更新检查结果缓存：发布新版本后大批客户端会同时发起检查，若每次都回源查询
// （尤其 SQLite 单连接场景），容易在洪峰下把最新的 latest.yml 验证/生成拖慢，
// 触发客户端 12 秒检查超时。这里做短 TTL 缓存把并发检查收敛到一次 DB 查询。
var updateVersionCache = cache.NewCacheWithTTL(1024, 5*time.Second)

// latestVersionCached 返回平台最新已启用版本（带灰度过滤），带 5 秒进程内缓存。
// key 含 platform + clientID，保证灰度分桶结果不被缓存串号。
func latestVersionCached(svc *service.VersionService, platform, clientID string) (*model.ClientVersion, error) {
	key := platform + "\x00" + clientID
	if v, ok := updateVersionCache.Get(key); ok {
		if ver, ok := v.(*model.ClientVersion); ok {
			return ver, nil
		}
	}

	version, err := svc.GetLatestEnabled(platform, clientID)
	if err != nil {
		return nil, err
	}
	updateVersionCache.Put(key, version)
	return version, nil
}

// HandleUpdateRequest 统一处理更新请求
// GET /api/v1/updates/:platform/*action
func HandleUpdateRequest(c *gin.Context) {
	action := c.Param("action")
	platform := c.Param("platform")
	logger.WithModule("Update").Debug("HandleUpdateRequest", "platform", platform, "action", action)
	// action 格式: /latest.yml, /latest-mac.yml
	action = strings.TrimPrefix(action, "/")

	if strings.HasPrefix(action, "latest") && strings.HasSuffix(action, ".yml") {
		GetLatestYML(c)
	} else {
		RedirectUpdateFile(c, platform, action)
	}
}

func RedirectUpdateFile(c *gin.Context, platformParam string, filename string) {
	platform := service.NormalizePlatform(platformParam)
	clientID := updateClientID(c)
	db := database.GetDB()
	svc := service.NewVersionService(db, versionStorageAccessor())
	version, err := latestVersionCached(svc, platform, clientID)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	if filename != safeUpdatePathFilename(db, *version) {
		c.Status(http.StatusNotFound)
		return
	}

	c.Redirect(http.StatusFound, absoluteUpdateURL(c, version.DownloadURL))
}

// GetLatestYML 返回 electron-updater 需要的 latest.yml 格式
// GET /api/v1/updates/:platform/latest.yml
func GetLatestYML(c *gin.Context) {
	platformParam := c.Param("platform")
	platform := service.NormalizePlatform(platformParam)
	clientID := updateClientID(c)

	logger.WithModule("Update").Info("检查更新请求",
		"platform_param", platformParam,
		"platform", platform,
		"client_id", clientID,
		"client_ip", c.ClientIP(),
	)

	db := database.GetDB()
	svc := service.NewVersionService(db, versionStorageAccessor())
	version, err := latestVersionCached(svc, platform, clientID)
	if err != nil {
		logger.WithModule("Update").Warn("无可用版本记录",
			"platform", platform,
			"platform_param", platformParam,
		)
		c.Status(http.StatusNotFound)
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
	updatePathName := safeUpdatePathFilename(db, *version)

	forceUpdateStr := "false"
	if version.ForceUpdate {
		forceUpdateStr = "true"
	}

	// 增量更新 blockmap 字段（仅预留，非空时才输出）
	blockmapLine := ""
	if version.BlockmapURL != "" {
		blockmapLine = fmt.Sprintf("blockmap: %s\n", version.BlockmapURL)
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
%s`,
		version.Version,
		updatePathName,
		sha512Base64,
		fileSize,
		updatePathName,
		sha512Base64,
		version.CreatedAt.Format("2006-01-02T15:04:05.000Z"),
		formatYAMLBlockString(version.Changelog),
		forceUpdateStr,
		blockmapLine,
	)

	c.Header("Content-Type", "text/yaml")
	c.String(http.StatusOK, yml)
}

func updateClientID(c *gin.Context) string {
	if clientID := c.Query("client"); clientID != "" {
		return clientID
	}
	return c.GetHeader("X-QIM-Update-Client")
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
