package handler

import (
	"strconv"

	"qim-server/database"
	"qim-server/model"
	"qim-server/pkg/response"

	"github.com/gin-gonic/gin"
)

func GetVersions(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
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

	response.Success(c, gin.H{
		"list":     versions,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

func CreateVersion(c *gin.Context) {
	var req struct {
		Version     string `json:"version" binding:"required"`
		Platform    string `json:"platform" binding:"required"`
		Type        string `json:"type"`
		DownloadURL string `json:"download_url"`
		Changelog   string `json:"changelog"`
		ForceUpdate bool   `json:"force_update"`
		Enabled     bool   `json:"enabled"`
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

	versionType := req.Type
	if versionType == "" {
		versionType = "full"
	}

	version := model.ClientVersion{
		Version:     req.Version,
		Platform:    req.Platform,
		Type:        versionType,
		DownloadURL: req.DownloadURL,
		Changelog:   req.Changelog,
		ForceUpdate: req.ForceUpdate,
		Enabled:     req.Enabled,
	}

	if err := db.Create(&version).Error; err != nil {
		response.InternalServerError(c, "创建失败")
		return
	}

	response.Success(c, version)
}

func UpdateVersion(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var req struct {
		Version     string `json:"version"`
		Platform    string `json:"platform"`
		Type        string `json:"type"`
		DownloadURL string `json:"download_url"`
		Changelog   string `json:"changelog"`
		ForceUpdate *bool  `json:"force_update"`
		Enabled     *bool  `json:"enabled"`
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

	if req.Version != "" {
		version.Version = req.Version
	}
	if req.Platform != "" {
		version.Platform = req.Platform
	}
	if req.Type != "" {
		version.Type = req.Type
	}
	if req.DownloadURL != "" {
		version.DownloadURL = req.DownloadURL
	}
	if req.Changelog != "" {
		version.Changelog = req.Changelog
	}
	if req.ForceUpdate != nil {
		version.ForceUpdate = *req.ForceUpdate
	}
	if req.Enabled != nil {
		version.Enabled = *req.Enabled
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

	db := database.GetDB()
	var version model.ClientVersion
	if err := db.First(&version, uint(id)).Error; err != nil {
		response.NotFound(c, "版本不存在")
		return
	}

	version.Enabled = !version.Enabled
	if err := db.Save(&version).Error; err != nil {
		response.InternalServerError(c, "更新失败")
		return
	}

	response.Success(c, version)
}
