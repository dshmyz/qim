package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"qim-server/database"
	"qim-server/model"

	"github.com/gin-gonic/gin"
)

// containsRole 检查角色列表中是否包含指定角色
func containsRole(roles []string, role string) bool {
	for _, r := range roles {
		if r == role {
			return true
		}
	}
	return false
}

func GetApps(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDUint := userID.(uint)

	db := database.GetDB()

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	name := c.Query("name")
	status := c.Query("status")
	category := c.Query("category")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	// 用户查看：自己的应用 + 全局应用
	query := db.Model(&model.App{}).Where("(user_id = ? OR is_global = ?) AND deleted_at IS NULL", userIDUint, true)

	if name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}

	if category != "" {
		query = query.Where("category = ?", category)
	}

	if status != "" && status != "all" {
		query = query.Where("status = ?", status)
	}

	var total int64
	query.Count(&total)

	var apps []model.App
	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&apps).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取应用列表失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":     apps,
			"total":    total,
			"page":     page,
			"pageSize": pageSize,
		},
	})
}

func ToggleAppStatus(c *gin.Context) {
	appIDStr := c.Param("id")

	appID, err := strconv.ParseUint(appIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的应用ID"})
		return
	}

	db := database.GetDB()
	var app model.App
	if err := db.First(&app, uint(appID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "应用不存在"})
		return
	}

	app.Status = map[string]string{
		"active":   "inactive",
		"inactive": "active",
	}[app.Status]

	db.Save(&app)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": app,
	})
}

func GetAllApps(c *gin.Context) {
	db := database.GetDB()

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	isGlobal := c.Query("is_global")
	status := c.Query("status")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	query := db.Model(&model.App{})

	if isGlobal != "" {
		if isGlobal == "true" {
			query = query.Where("is_global = ?", true)
		} else {
			query = query.Where("is_global = ?", false)
		}
	}

	if status != "" && status != "all" {
		query = query.Where("status = ?", status)
	}

	var total int64
	query.Count(&total)

	var apps []model.App
	offset := (page - 1) * pageSize
	query.Order("is_global DESC, created_at DESC").Offset(offset).Limit(pageSize).Find(&apps)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":     apps,
			"total":    total,
			"page":     page,
			"pageSize": pageSize,
		},
	})
}

func CreateApp(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req struct {
		Name     string `json:"name" binding:"required"`
		Icon     string `json:"icon"`
		Category string `json:"category"`
		URL      string `json:"url"`
		Status   string `json:"status"`
		OpenType string `json:"open_type"`
		IsGlobal bool   `json:"is_global"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	// 检查是否为全局应用，如果是则需要管理员权限
	if req.IsGlobal {
		roles, exists := c.Get("roles")
		if !exists || !containsRole(roles.([]string), "system_admin") {
			c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "只有管理员才能创建全局应用"})
			return
		}
	}

	db := database.GetDB()
	app := model.App{
		UserID:   userID.(uint),
		Name:     req.Name,
		Icon:     req.Icon,
		Category: req.Category,
		URL:      req.URL,
		Status:   req.Status,
		OpenType: req.OpenType,
		IsGlobal: req.IsGlobal,
	}
	result := db.Create(&app)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建应用失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": app,
	})
}

func UpdateApp(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDUint := userID.(uint)
	appIDStr := c.Param("id")

	appID, err := strconv.ParseUint(appIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的应用ID"})
		return
	}

	var req struct {
		Name     string `json:"name" binding:"required"`
		Icon     string `json:"icon"`
		Category string `json:"category"`
		URL      string `json:"url"`
		Status   string `json:"status"`
		OpenType string `json:"open_type"`
		IsGlobal bool   `json:"is_global"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	db := database.GetDB()
	var app model.App

	// 先尝试查找用户自己的应用
	if err := db.Where("id = ? AND user_id = ?", uint(appID), userIDUint).First(&app).Error; err != nil {
		// 如果不是自己的应用，检查是否是全局应用
		if err := db.Where("id = ? AND is_global = ?", uint(appID), true).First(&app).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "应用不存在"})
			return
		}

		// 全局应用需要管理员权限才能更新
		roles, exists := c.Get("roles")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "无权限更新此应用"})
			return
		}

		roleList, ok := roles.([]string)
		if !ok || !containsRole(roleList, "system_admin") {
			c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "无权限更新此应用"})
			return
		}
	}

	// 如果请求修改 is_global 字段，需要检查管理员权限
	if req.IsGlobal != app.IsGlobal {
		roles, exists := c.Get("roles")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "只有管理员才能修改应用的全局状态"})
			return
		}

		roleList, ok := roles.([]string)
		if !ok || !containsRole(roleList, "system_admin") {
			c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "只有管理员才能修改应用的全局状态"})
			return
		}
	}

	app.Name = req.Name
	app.Icon = req.Icon
	app.Category = req.Category
	app.URL = req.URL
	app.Status = req.Status
	if req.OpenType != "" {
		app.OpenType = req.OpenType
	}
	app.IsGlobal = req.IsGlobal

	db.Save(&app)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": app,
	})
}

func DeleteApp(c *gin.Context) {
	userID, _ := c.Get("user_id")
	appIDStr := c.Param("id")

	appID, err := strconv.ParseUint(appIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的应用ID"})
		return
	}

	db := database.GetDB()
	if err := db.Where("id = ? AND user_id = ?", uint(appID), userID).Delete(&model.App{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除应用失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "删除应用成功",
	})
}

func GetMiniApps(c *gin.Context) {
	db := database.GetDB()

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	name := c.Query("name")
	status := c.Query("status")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	query := db.Model(&model.MiniApp{})

	if name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}

	if status != "" && status != "all" {
		query = query.Where("status = ?", status)
	}

	var total int64
	query.Count(&total)

	var miniApps []model.MiniApp
	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&miniApps).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "查询小程序失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":     miniApps,
			"total":    total,
			"page":     page,
			"pageSize": pageSize,
		},
	})
}

func GetMiniApp(c *gin.Context) {
	appID := c.Param("id")

	db := database.GetDB()
	var miniApp model.MiniApp
	if err := db.Where("app_id = ?", appID).First(&miniApp).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "小程序不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": miniApp,
	})
}

func CreateMiniApp(c *gin.Context) {
	var req struct {
		AppID       string `json:"app_id" binding:"required"`
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		Icon        string `json:"icon"`
		Path        string `json:"path"`
		Permissions string `json:"permissions"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	db := database.GetDB()

	var count int64
	db.Model(&model.MiniApp{}).Where("app_id = ?", req.AppID).Count(&count)
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "AppID已存在"})
		return
	}

	miniApp := model.MiniApp{
		AppID:       req.AppID,
		Name:        req.Name,
		Description: req.Description,
		Icon:        req.Icon,
		Path:        req.Path,
		Status:      "inactive",
		Permissions: req.Permissions,
	}

	if err := db.Create(&miniApp).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建小程序失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": miniApp,
	})
}

func UpdateMiniApp(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Icon        string `json:"icon"`
		Path        string `json:"path"`
		Status      string `json:"status"`
		Permissions string `json:"permissions"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	db := database.GetDB()
	var miniApp model.MiniApp
	if err := db.Where("id = ?", id).First(&miniApp).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "小程序不存在"})
		return
	}

	if req.Name != "" {
		miniApp.Name = req.Name
	}
	if req.Description != "" {
		miniApp.Description = req.Description
	}
	if req.Icon != "" {
		miniApp.Icon = req.Icon
	}
	if req.Path != "" {
		miniApp.Path = req.Path
	}
	if req.Status != "" {
		miniApp.Status = req.Status
	}
	if req.Permissions != "" {
		miniApp.Permissions = req.Permissions
	}

	if err := db.Save(&miniApp).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新小程序失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": miniApp,
	})
}

func DeleteMiniApp(c *gin.Context) {
	id := c.Param("id")

	db := database.GetDB()
	if err := db.Where("id = ?", id).Delete(&model.MiniApp{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除小程序失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "删除小程序成功",
	})
}

func GetNotes(c *gin.Context) {
	userID, _ := c.Get("user_id")

	db := database.GetDB()
	var notes []model.Note
	db.Where("user_id = ?", userID).Order("updated_at DESC").Find(&notes)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": notes,
	})
}

func GetNote(c *gin.Context) {
	userID, _ := c.Get("user_id")
	noteIDStr := c.Param("id")

	noteID, err := strconv.ParseUint(noteIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的笔记ID"})
		return
	}

	db := database.GetDB()
	var note model.Note
	if err := db.Where("id = ? AND user_id = ?", uint(noteID), userID).First(&note).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "笔记不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": note,
	})
}

func CreateNote(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req struct {
		Title   string `json:"title" binding:"required"`
		Content string `json:"content" binding:"omitempty"`
		Color   string `json:"color"`
		Type    string `json:"type"`
		Style   string `json:"style"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		fmt.Println(err)
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	style := req.Style
	if style == "" {
		style = "{}"
	}
	// If color is provided, merge it into style JSON
	if req.Color != "" {
		var styleMap map[string]interface{}
		if err := json.Unmarshal([]byte(style), &styleMap); err == nil {
			styleMap["color"] = req.Color
			if styleBytes, err := json.Marshal(styleMap); err == nil {
				style = string(styleBytes)
			}
		}
	}

	db := database.GetDB()
	note := model.Note{
		UserID:  userID.(uint),
		Title:   req.Title,
		Content: req.Content,
		Type:    req.Type,
		Style:   style,
	}
	db.Create(&note)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": note,
	})
}

func UpdateNote(c *gin.Context) {
	userID, _ := c.Get("user_id")
	noteIDStr := c.Param("id")

	noteID, err := strconv.ParseUint(noteIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的笔记ID"})
		return
	}

	var req struct {
		Title   string `json:"title" binding:"required"`
		Content string `json:"content" binding:"omitempty"`
		Color   string `json:"color"`
		Type    string `json:"type"`
		Style   string `json:"style"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	db := database.GetDB()
	var note model.Note
	if err := db.Where("id = ? AND user_id = ?", uint(noteID), userID).First(&note).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "笔记不存在"})
		return
	}

	note.Title = req.Title
	note.Content = req.Content
	note.Type = req.Type

	// Merge color and style into style JSON
	style := note.Style
	if style == "" {
		style = "{}"
	}
	var styleMap map[string]interface{}
	if err := json.Unmarshal([]byte(style), &styleMap); err == nil {
		if req.Color != "" {
			styleMap["color"] = req.Color
		}
		if req.Style != "" {
			var newStyleMap map[string]interface{}
			if err := json.Unmarshal([]byte(req.Style), &newStyleMap); err == nil {
				for k, v := range newStyleMap {
					styleMap[k] = v
				}
			}
		}
		if styleBytes, err := json.Marshal(styleMap); err == nil {
			note.Style = string(styleBytes)
		}
	}
	db.Save(&note)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": note,
	})
}

func DeleteNote(c *gin.Context) {
	userID, _ := c.Get("user_id")
	noteIDStr := c.Param("id")

	noteID, err := strconv.ParseUint(noteIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的笔记ID"})
		return
	}

	db := database.GetDB()
	if err := db.Where("id = ? AND user_id = ?", uint(noteID), userID).Delete(&model.Note{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除笔记失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "删除笔记成功",
	})
}
