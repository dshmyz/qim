package handler

import (
	"crypto/sha512"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/model"

	"github.com/gin-gonic/gin"
)

// GetLatestYML 返回 electron-updater 需要的 latest.yml 格式
// GET /api/v1/updates/:platform/latest.yml
func GetLatestYML(c *gin.Context) {
	platform := c.Param("platform") // win7, win10, linux, mac

	db := database.GetDB()
	var version model.ClientVersion
	err := db.Where("platform = ? AND enabled = ?", platform, true).
		Order("created_at DESC").First(&version).Error
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	// 计算安装包的 SHA512 和大小
	filePath := version.DownloadURL
	sha512Hash := ""
	fileSize := int64(0)

	if filePath != "" {
		sha512Hash, fileSize = computeFileSHA512(filePath)
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
		filePath,
		sha512Hash,
		fileSize,
		filePath,
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
