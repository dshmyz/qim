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

	// 转换为前端期望格式
	var frontendList []gin.H
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

	if err := db.Create(&version).Error; err != nil {
		response.InternalServerError(c, "创建失败")
		return
	}

	response.Success(c, version)
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
