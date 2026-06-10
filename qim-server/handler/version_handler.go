package handler

import (
	"crypto/sha512"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/logger"
	"github.com/dshmyz/qim/qim-server/pkg/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GetVersions(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	platform := c.Query("platform")

	db := database.GetDB()

	query := db.Model(&model.ClientVersion{})
	if platform != "" {
		query = query.Where("platform = ?", platform)
	}

	var total int64
	query.Count(&total)

	var versions []model.ClientVersion
	offset := (page - 1) * pageSize
	query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&versions)

	frontendList := make([]gin.H, 0, len(versions))
	for _, v := range versions {
		status := "inactive"
		if v.Enabled {
			status = "active"
		}

		frontendList = append(frontendList, gin.H{
			"id":           v.ID,
			"version":      v.Version,
			"platform":     v.Platform,
			"downloadUrl":  v.DownloadURL,
			"updateNotes":  v.Changelog,
			"forceUpdate":  v.ForceUpdate,
			"status":       status,
			"releaseDate":  v.CreatedAt.Format("2006-01-02"),
			"createdAt":    v.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	response.Success(c, gin.H{
		"list":     frontendList,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

func CreateVersion(c *gin.Context) {
	var req struct {
		Version      string `json:"version" binding:"required"`
		Platform     string `json:"platform" binding:"required"`
		ReleaseDate  string `json:"releaseDate"`
		DownloadUrl  string `json:"downloadUrl"`
		UpdateNotes  string `json:"updateNotes"`
		ForceUpdate  bool   `json:"forceUpdate"`
		Sha512       string `json:"sha512"`
		FileSize     int64  `json:"fileSize"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	db := database.GetDB()

	var existing model.ClientVersion
	if err := db.Where("version = ? AND platform = ? AND deleted_at IS NULL", req.Version, req.Platform).First(&existing).Error; err == nil {
		response.BadRequest(c, "该版本已存在")
		return
	}

	version := model.ClientVersion{
		Version:     req.Version,
		Platform:    req.Platform,
		DownloadURL: req.DownloadUrl,
		Changelog:   req.UpdateNotes,
		ForceUpdate: req.ForceUpdate,
		Enabled:     true,
	}

	if (req.Sha512 == "") != (req.FileSize == 0) {
		response.BadRequest(c, "SHA512 和文件大小必须同时提供")
		return
	}

	if req.Sha512 != "" && req.FileSize > 0 {
		version.Sha512 = req.Sha512
		version.FileSize = req.FileSize
	}

	// 计算文件的 SHA512 和大小
	if req.DownloadUrl != "" {
		if version.Sha512 == "" || version.FileSize == 0 {
			sha512Hash, fileSize := computeVersionFileSHA512(db, req.DownloadUrl, req.Platform)
			if sha512Hash != "" && fileSize > 0 {
				version.Sha512 = sha512Hash
				version.FileSize = fileSize
				logger.WithModule("Version").Info("已计算文件SHA512",
					"version", req.Version,
					"hash", sha512Hash,
					"size", fileSize,
				)
			}
		}

		if version.Sha512 == "" || version.FileSize <= 0 {
			response.BadRequest(c, "下载链接缺少可用的 SHA512 和文件大小")
			return
		}
	}

	if err := db.Create(&version).Error; err != nil {
		response.InternalServerError(c, "创建失败")
		return
	}

	response.Success(c, version)
}

// computeVersionFileSHA512 计算版本文件的 SHA512 和大小
func computeVersionFileSHA512(db *gorm.DB, downloadURL string, platform string) (string, int64) {
	// 检查是否是公开文件下载链接
	if strings.Contains(downloadURL, "/api/v1/public/files/") && strings.HasSuffix(downloadURL, "/download") {
		// 从 URL 中提取文件 ID
		parts := strings.Split(downloadURL, "/")
		for i, part := range parts {
			if part == "files" && i+1 < len(parts) {
				fileIDStr := parts[i+1]
				if fileID, err := strconv.ParseUint(fileIDStr, 10, 32); err == nil {
					// 从数据库中获取文件信息
					var file model.File
					if err := db.First(&file, uint(fileID)).Error; err == nil {
						// 处理存储路径
						storagePath := file.StoragePath
						if strings.HasPrefix(storagePath, "/uploads/") {
							storagePath = "." + storagePath
						}
						return computeFileSHA512V2(storagePath)
					}
				}
				break
			}
		}
	} else if strings.HasPrefix(downloadURL, "http://") || strings.HasPrefix(downloadURL, "https://") {
		// 其他 URL：尝试在 updates 目录查找缓存文件
		filename := filepath.Base(downloadURL)
		localPath := filepath.Join("./uploads/updates", platform, filename)
		if _, err := os.Stat(localPath); err == nil {
			return computeFileSHA512V2(localPath)
		}
	} else {
		// 本地文件路径模式
		return computeFileSHA512V2(downloadURL)
	}

	return "", 0
}

// computeFileSHA512V2 计算文件的 SHA512 哈希和大小
func computeFileSHA512V2(filePath string) (string, int64) {
	file, err := os.Open(filePath)
	if err != nil {
		logger.WithModule("Version").Error("打开文件失败", "path", filePath, "error", err)
		return "", 0
	}
	defer file.Close()

	hash := sha512.New()
	size, err := io.Copy(hash, file)
	if err != nil {
		logger.WithModule("Version").Error("计算哈希失败", "path", filePath, "error", err)
		return "", 0
	}

	sum := hash.Sum(nil)
	return fmt.Sprintf("%x", sum), size
}

func UpdateVersion(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var req struct {
		UpdateNotes *string `json:"updateNotes"`
		ForceUpdate *bool   `json:"forceUpdate"`
		Status      *string `json:"status"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	db := database.GetDB()
	var version model.ClientVersion
	if err := db.First(&version, uint(id)).Error; err != nil {
		response.NotFound(c, "版本不存在")
		return
	}

	if req.UpdateNotes != nil {
		version.Changelog = *req.UpdateNotes
	}
	if req.ForceUpdate != nil {
		version.ForceUpdate = *req.ForceUpdate
	}
	if req.Status != nil {
		version.Enabled = *req.Status == "active"
	}

	if err := db.Save(&version).Error; err != nil {
		response.InternalServerError(c, "更新失败")
		return
	}

	response.Success(c, version)
}

func DeleteVersion(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	db := database.GetDB()
	var version model.ClientVersion
	if err := db.First(&version, uint(id)).Error; err != nil {
		response.NotFound(c, "版本不存在")
		return
	}

	if err := db.Delete(&version).Error; err != nil {
		response.InternalServerError(c, "删除失败")
		return
	}

	response.Success(c, gin.H{"message": "删除成功"})
}

func ToggleVersionStatus(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var req struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	db := database.GetDB()
	var version model.ClientVersion
	if err := db.First(&version, uint(id)).Error; err != nil {
		response.NotFound(c, "版本不存在")
		return
	}

	version.Enabled = req.Status == "active"
	if err := db.Save(&version).Error; err != nil {
		response.InternalServerError(c, "更新失败")
		return
	}

	response.Success(c, version)
}

func GetVersionDistribution(c *gin.Context) {
	db := database.GetDB()

	type VersionDistribution struct {
		Version string `json:"version"`
		Count   int64  `json:"count"`
	}

	var distributions []VersionDistribution

	db.Table("users").
		Select("client_version as version, COUNT(*) as count").
		Where("client_version IS NOT NULL AND client_version != ''").
		Group("client_version").
		Order("count DESC").
		Find(&distributions)

	response.Success(c, distributions)
}
