package handler

import (
	"errors"
	"strconv"

	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/di"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/response"
	"github.com/dshmyz/qim/qim-server/service"
	"github.com/dshmyz/qim/qim-server/ws"

	"github.com/gin-gonic/gin"
)

// versionStorageAccessor 从全局容器获取存储访问器；容器未初始化时返回 nil（测试环境）。
func versionStorageAccessor() service.StorageAccessor {
	if di.GlobalContainer == nil || di.GlobalContainer.StorageManager == nil {
		return nil
	}
	return di.NewStorageAccessor(di.GlobalContainer.StorageManager)
}

// versionToFrontend 将 ClientVersion 模型转换为前端期望的 camelCase 格式。
func versionToFrontend(v model.ClientVersion) gin.H {
	status := "inactive"
	if v.Enabled {
		status = "active"
	}
	return gin.H{
		"id":                v.ID,
		"version":           v.Version,
		"platform":          v.Platform,
		"downloadUrl":       v.DownloadURL,
		"updateNotes":       v.Changelog,
		"forceUpdate":       v.ForceUpdate,
		"rolloutPercentage": v.GetRolloutPercentage(),
		"minVersion":        v.MinVersion,
		"status":            status,
		"releaseDate":       v.CreatedAt.Format("2006-01-02"),
		"createdAt":         v.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}

func GetVersions(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	platform := c.Query("platform")

	svc := service.NewVersionService(database.GetDB(), versionStorageAccessor())
	versions, total, err := svc.List(page, pageSize, platform, "client")
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}

	frontendList := make([]gin.H, 0, len(versions))
	for _, v := range versions {
		frontendList = append(frontendList, versionToFrontend(v))
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
		Version           string `json:"version" binding:"required"`
		Platform          string `json:"platform" binding:"required"`
		ReleaseDate       string `json:"releaseDate"`
		DownloadUrl       string `json:"downloadUrl"`
		UpdateNotes       string `json:"updateNotes"`
		ForceUpdate       bool   `json:"forceUpdate"`
		Sha512            string `json:"sha512"`
		FileSize          int64  `json:"fileSize"`
		RolloutPercentage *int   `json:"rolloutPercentage"`
		MinVersion        string `json:"minVersion"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	svc := service.NewVersionService(database.GetDB(), versionStorageAccessor())
	version, err := svc.Create(service.CreateVersionInput{
		Version:           req.Version,
		Platform:          req.Platform,
		DownloadURL:       req.DownloadUrl,
		Changelog:         req.UpdateNotes,
		ForceUpdate:       req.ForceUpdate,
		RolloutPercentage: req.RolloutPercentage,
		MinVersion:        req.MinVersion,
		Sha512:            req.Sha512,
		FileSize:          req.FileSize,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrVersionExists):
			response.BadRequest(c, "该版本已存在")
		case errors.Is(err, service.ErrMissingDownloadURL):
			response.BadRequest(c, "下载链接不能为空")
		case errors.Is(err, service.ErrHashComputeFailed):
			response.BadRequest(c, err.Error())
		case errors.Is(err, service.ErrInvalidRolloutPercentage):
			response.BadRequest(c, err.Error())
		case errors.Is(err, service.ErrMissingDownloadURL), errors.Is(err, service.ErrMissingSha256):
			response.BadRequest(c, err.Error())
		default:
			response.InternalServerError(c, "创建失败")
		}
		return
	}

	response.Success(c, versionToFrontend(*version))
}

func UpdateVersion(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var req struct {
		UpdateNotes       *string `json:"updateNotes"`
		ForceUpdate       *bool   `json:"forceUpdate"`
		Status            *string `json:"status"`
		RolloutPercentage *int    `json:"rolloutPercentage"`
		MinVersion        *string `json:"minVersion"`
		DownloadUrl       *string `json:"downloadUrl"`
		Sha256            *string `json:"sha256"`
		FileSize          *int64  `json:"fileSize"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	svc := service.NewVersionService(database.GetDB(), versionStorageAccessor())
	version, err := svc.Update(uint(id), service.UpdateVersionInput{
		Changelog:         req.UpdateNotes,
		ForceUpdate:       req.ForceUpdate,
		Status:            req.Status,
		RolloutPercentage: req.RolloutPercentage,
		MinVersion:        req.MinVersion,
		DownloadURL:       req.DownloadUrl,
		Sha256:            req.Sha256,
		FileSize:          req.FileSize,
	})
	if err != nil {
		if errors.Is(err, service.ErrVersionNotFound) {
			response.NotFound(c, "版本不存在")
			return
		}
		if errors.Is(err, service.ErrInvalidRolloutPercentage) {
			response.BadRequest(c, err.Error())
			return
		}
		response.InternalServerError(c, "更新失败")
		return
	}

	response.Success(c, versionToFrontend(*version))
}

func DeleteVersion(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	svc := service.NewVersionService(database.GetDB(), versionStorageAccessor())
	if err := svc.Delete(uint(id)); err != nil {
		if errors.Is(err, service.ErrVersionNotFound) {
			response.NotFound(c, "版本不存在")
			return
		}
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

	svc := service.NewVersionService(database.GetDB(), versionStorageAccessor())
	version, err := svc.ToggleStatus(uint(id), req.Status == "active")
	if err != nil {
		if errors.Is(err, service.ErrVersionNotFound) {
			response.NotFound(c, "版本不存在")
			return
		}
		response.InternalServerError(c, "更新失败")
		return
	}

	response.Success(c, versionToFrontend(*version))
}

// RollbackVersion 回滚到指定版本
// POST /api/v1/admin/client/versions/:id/rollback
func RollbackVersion(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	svc := service.NewVersionService(database.GetDB(), versionStorageAccessor())
	if err := svc.Rollback(uint(id)); err != nil {
		if errors.Is(err, service.ErrVersionNotFound) {
			response.NotFound(c, "版本不存在")
			return
		}
		response.InternalServerError(c, "回滚失败")
		return
	}

	response.Success(c, gin.H{"message": "回滚成功"})
}

func GetVersionDistribution(c *gin.Context) {
	// T6: 改为复用 WebSocket 内存统计在线设备的版本分布
	// 仅统计当前在线设备，断开即减，无僵尸数据
	hubVal, exists := c.Get("hub")
	if !exists {
		response.InternalServerError(c, "WebSocket Hub 未初始化")
		return
	}
	hub, ok := hubVal.(*ws.Hub)
	if !ok {
		response.InternalServerError(c, "WebSocket Hub 类型错误")
		return
	}
	stats := hub.GetVersionStats()
	if stats == nil {
		stats = []ws.VersionStat{}
	}
	response.Success(c, stats)
}

// ========== CLI 版本管理 (Admin) ==========

func cliVersionToFrontend(v model.ClientVersion) gin.H {
	status := "inactive"
	if v.Enabled {
		status = "active"
	}
	return gin.H{
		"id":                v.ID,
		"version":           v.Version,
		"platform":          v.Platform,
		"os":                v.Os,
		"arch":              v.Arch,
		"downloadUrl":       v.DownloadURL,
		"sha256":            v.Sha256,
		"fileSize":          v.FileSize,
		"updateNotes":       v.Changelog,
		"forceUpdate":       v.ForceUpdate,
		"rolloutPercentage": v.GetRolloutPercentage(),
		"minVersion":        v.MinVersion,
		"status":            status,
		"releaseDate":       v.CreatedAt.Format("2006-01-02"),
		"createdAt":         v.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}

// GetCLIVersions 获取 CLI 版本列表
// GET /api/v1/admin/cli/versions
func GetCLIVersions(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	svc := service.NewVersionService(database.GetDB(), versionStorageAccessor())
	versions, total, err := svc.List(page, pageSize, "", "cli")
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}

	frontendList := make([]gin.H, 0, len(versions))
	for _, v := range versions {
		frontendList = append(frontendList, cliVersionToFrontend(v))
	}

	response.Success(c, gin.H{
		"list":     frontendList,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// CreateCLIVersion 创建 CLI 版本
// POST /api/v1/admin/cli/versions
func CreateCLIVersion(c *gin.Context) {
	var req struct {
		Version           string `json:"version" binding:"required"`
		Os                string `json:"os" binding:"required"`
		Arch              string `json:"arch" binding:"required"`
		DownloadUrl       string `json:"downloadUrl"`
		UpdateNotes       string `json:"updateNotes"`
		ForceUpdate       bool   `json:"forceUpdate"`
		Sha256            string `json:"sha256"`
		FileSize          int64  `json:"fileSize"`
		RolloutPercentage *int   `json:"rolloutPercentage"`
		MinVersion        string `json:"minVersion"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	svc := service.NewVersionService(database.GetDB(), versionStorageAccessor())
	version, err := svc.Create(service.CreateVersionInput{
		Version:           req.Version,
		Platform:          req.Os + "-" + req.Arch,
		AppType:           "cli",
		Os:                req.Os,
		Arch:              req.Arch,
		DownloadURL:       req.DownloadUrl,
		Changelog:         req.UpdateNotes,
		ForceUpdate:       req.ForceUpdate,
		RolloutPercentage: req.RolloutPercentage,
		MinVersion:        req.MinVersion,
		Sha256:            req.Sha256,
		FileSize:          req.FileSize,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrVersionExists):
			response.BadRequest(c, "该版本已存在")
		case errors.Is(err, service.ErrMissingDownloadURL), errors.Is(err, service.ErrMissingSha256):
			response.BadRequest(c, err.Error())
		case errors.Is(err, service.ErrInvalidRolloutPercentage):
			response.BadRequest(c, err.Error())
		default:
			response.InternalServerError(c, "创建失败")
		}
		return
	}

	response.Success(c, cliVersionToFrontend(*version))
}

// UpdateCLIVersion 更新 CLI 版本
// PUT /api/v1/admin/cli/versions/:id
func UpdateCLIVersion(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var req struct {
		UpdateNotes       *string `json:"updateNotes"`
		ForceUpdate       *bool   `json:"forceUpdate"`
		Status            *string `json:"status"`
		RolloutPercentage *int    `json:"rolloutPercentage"`
		MinVersion        *string `json:"minVersion"`
		DownloadUrl       *string `json:"downloadUrl"`
		Sha256            *string `json:"sha256"`
		FileSize          *int64  `json:"fileSize"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	svc := service.NewVersionService(database.GetDB(), versionStorageAccessor())
	version, err := svc.Update(uint(id), service.UpdateVersionInput{
		Changelog:         req.UpdateNotes,
		ForceUpdate:       req.ForceUpdate,
		Status:            req.Status,
		RolloutPercentage: req.RolloutPercentage,
		MinVersion:        req.MinVersion,
		DownloadURL:       req.DownloadUrl,
		Sha256:            req.Sha256,
		FileSize:          req.FileSize,
	})
	if err != nil {
		if errors.Is(err, service.ErrVersionNotFound) {
			response.NotFound(c, "版本不存在")
			return
		}
		if errors.Is(err, service.ErrInvalidRolloutPercentage) {
			response.BadRequest(c, err.Error())
			return
		}
		response.InternalServerError(c, "更新失败")
		return
	}

	response.Success(c, cliVersionToFrontend(*version))
}

// DeleteCLIVersion 删除 CLI 版本
// DELETE /api/v1/admin/cli/versions/:id
func DeleteCLIVersion(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	svc := service.NewVersionService(database.GetDB(), versionStorageAccessor())
	if err := svc.Delete(uint(id)); err != nil {
		if errors.Is(err, service.ErrVersionNotFound) {
			response.NotFound(c, "版本不存在")
			return
		}
		response.InternalServerError(c, "删除失败")
		return
	}

	response.Success(c, gin.H{"message": "删除成功"})
}

// ToggleCLIVersionStatus 切换 CLI 版本启用状态
// PATCH /api/v1/admin/cli/versions/:id/toggle
func ToggleCLIVersionStatus(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var req struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	svc := service.NewVersionService(database.GetDB(), versionStorageAccessor())
	version, err := svc.ToggleStatus(uint(id), req.Status == "active")
	if err != nil {
		if errors.Is(err, service.ErrVersionNotFound) {
			response.NotFound(c, "版本不存在")
			return
		}
		response.InternalServerError(c, "更新失败")
		return
	}

	response.Success(c, cliVersionToFrontend(*version))
}
