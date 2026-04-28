package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"qim-server/database"
	"qim-server/model"

	"github.com/gin-gonic/gin"
)

func GetApps(c *gin.Context) {
	userID, _ := c.Get("user_id")

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

	query := db.Model(&model.App{}).Where("user_id = ?", userID)

	if name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}

	if status != "" {
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

func GetAllApps(c *gin.Context) {
	db := database.GetDB()
	var apps []model.App
	db.Order("created_at DESC").Find(&apps)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": apps,
	})
}

func CreateApp(c *gin.Context) {
	userID, _ := c.Get("user_id")
	fmt.Println("创建应用请求，用户ID:", userID)

	var req struct {
		Name     string `json:"name" binding:"required"`
		Icon     string `json:"icon"`
		Category string `json:"category"`
		URL      string `json:"url"`
		Status   string `json:"status"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		fmt.Println("参数错误:", err)
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	fmt.Println("创建应用请求体:", req)

	db := database.GetDB()
	app := model.App{
		UserID:   userID.(uint),
		Name:     req.Name,
		Icon:     req.Icon,
		Category: req.Category,
		URL:      req.URL,
		Status:   req.Status,
	}
	result := db.Create(&app)
	if result.Error != nil {
		fmt.Println("创建应用失败:", result.Error)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建应用失败"})
		return
	}

	fmt.Println("创建应用成功，应用ID:", app.ID)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": app,
	})
}

func UpdateApp(c *gin.Context) {
	userID, _ := c.Get("user_id")
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
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	db := database.GetDB()
	var app model.App
	if err := db.Where("id = ? AND user_id = ?", uint(appID), userID).First(&app).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "应用不存在"})
		return
	}

	app.Name = req.Name
	app.Icon = req.Icon
	app.Category = req.Category
	app.URL = req.URL
	app.Status = req.Status
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

	if status != "" {
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
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		fmt.Println(err)
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	db := database.GetDB()
	note := model.Note{
		UserID:  userID.(uint),
		Title:   req.Title,
		Content: req.Content,
		Color:   req.Color,
		Type:    req.Type,
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
	note.Color = req.Color
	note.Type = req.Type
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
