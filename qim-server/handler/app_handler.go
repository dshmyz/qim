package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"qim-server/di"
	"qim-server/model"
	"qim-server/pkg/response"
	"qim-server/service"

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

	appSvc := di.GlobalContainer.AppService

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

	result, err := appSvc.GetUserApps(service.AppQuery{
		UserID:   userIDUint,
		Page:     page,
		PageSize: pageSize,
		Name:     name,
		Status:   status,
		Category: category,
	})
	if err != nil {
		response.InternalServerError(c, "获取应用列表失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":     result.List,
			"total":    result.Total,
			"page":     result.Page,
			"pageSize": result.PageSize,
		},
	})
}

func ToggleAppStatus(c *gin.Context) {
	appIDStr := c.Param("id")

	appID, err := strconv.ParseUint(appIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的应用ID")
		return
	}

	appSvc := di.GlobalContainer.AppService
	app, err := appSvc.GetApp(uint(appID))
	if err != nil {
		response.NotFound(c, "应用不存在")
		return
	}

	app.Status = map[string]string{
		"active":   "inactive",
		"inactive": "active",
	}[app.Status]

	appSvc.UpdateApp(app)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": app,
	})
}

func GetAllApps(c *gin.Context) {
	appSvc := di.GlobalContainer.AppService

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

	result, err := appSvc.GetAllApps(service.AppQuery{
		Page:     page,
		PageSize: pageSize,
		IsGlobal: isGlobal,
		Status:   status,
	})
	if err != nil {
		response.InternalServerError(c, "获取应用列表失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":     result.List,
			"total":    result.Total,
			"page":     result.Page,
			"pageSize": result.PageSize,
		},
	})
}

// GetBuiltInApps 获取内置应用列表
func GetBuiltInApps(c *gin.Context) {
	appSvc := di.GlobalContainer.AppService

	userID, _ := c.Get("user_id")

	result, err := appSvc.GetBuiltInAppsForUser(userID.(uint))
	if err != nil {
		response.InternalServerError(c, "获取内置应用列表失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": result,
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
		// 权限范围控制字段（需要管理员权限）
		ScopeType       string `json:"scope_type"`
		ScopeValue      string `json:"scope_value"`
		AvailableOrgIDs string `json:"available_org_ids"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	if req.IsGlobal {
		roles, exists := c.Get("roles")
		if !exists || !containsRole(roles.([]string), "system_admin") {
			response.Forbidden(c, "只有管理员才能创建全局应用")
			return
		}
	}

	appSvc := di.GlobalContainer.AppService
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
	if err := appSvc.CreateApp(&app); err != nil {
		response.InternalServerError(c, "创建应用失败")
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
		response.BadRequest(c, "无效的应用ID")
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
		// 权限范围控制字段（需要管理员权限）
		ScopeType       string `json:"scope_type"`
		ScopeValue      string `json:"scope_value"`
		AvailableOrgIDs string `json:"available_org_ids"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	appSvc := di.GlobalContainer.AppService
	var app *model.App

	userApp, err1 := appSvc.GetUserApp(uint(appID), userIDUint)
	if err1 == nil {
		app = userApp
	} else {
		globalApp, err2 := appSvc.GetGlobalApp(uint(appID))
		if err2 != nil {
			response.NotFound(c, "应用不存在")
			return
		}
		app = globalApp

		roles, exists := c.Get("roles")
		if !exists {
			response.Forbidden(c, "无权限更新此应用")
			return
		}

		roleList, ok := roles.([]string)
		if !ok || !containsRole(roleList, "system_admin") {
			response.Forbidden(c, "无权限更新此应用")
			return
		}
	}

	// 如果请求修改 is_global 字段，需要检查管理员权限
	if req.IsGlobal != app.IsGlobal {
		roles, exists := c.Get("roles")
		if !exists {
			response.Forbidden(c, "只有管理员才能修改应用的全局状态")
			return
		}

		roleList, ok := roles.([]string)
		if !ok || !containsRole(roleList, "system_admin") {
			response.Forbidden(c, "只有管理员才能修改应用的全局状态")
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

	// 如果请求修改权限范围字段，需要检查管理员权限
	if req.ScopeType != "" || req.ScopeValue != "" || req.AvailableOrgIDs != "" {
		roles, exists := c.Get("roles")
		if !exists {
			response.Forbidden(c, "只有管理员才能修改应用的权限范围")
			return
		}
		roleList, ok := roles.([]string)
		if !ok || !containsRole(roleList, "system_admin") {
			response.Forbidden(c, "只有管理员才能修改应用的权限范围")
			return
		}
		app.ScopeType = req.ScopeType
		app.ScopeValue = req.ScopeValue
		app.AvailableOrgIDs = req.AvailableOrgIDs
	}

	appSvc.UpdateApp(app)

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
		response.BadRequest(c, "无效的应用ID")
		return
	}

	appSvc := di.GlobalContainer.AppService
	if err := appSvc.DeleteApp(uint(appID), userID.(uint)); err != nil {
		response.InternalServerError(c, "删除应用失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "删除应用成功",
	})
}

func GetMiniApps(c *gin.Context) {
	miniAppSvc := di.GlobalContainer.MiniAppService

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

	result, err := miniAppSvc.GetMiniApps(service.MiniAppQuery{
		Page:     page,
		PageSize: pageSize,
		Name:     name,
		Status:   status,
	})
	if err != nil {
		response.InternalServerError(c, "查询小程序失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":     result.List,
			"total":    result.Total,
			"page":     result.Page,
			"pageSize": result.PageSize,
		},
	})
}

func GetMiniApp(c *gin.Context) {
	appID := c.Param("id")

	miniAppSvc := di.GlobalContainer.MiniAppService
	miniApp, err := miniAppSvc.GetMiniAppByAppID(appID)
	if err != nil {
		response.NotFound(c, "小程序不存在")
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
		response.BadRequest(c, "参数错误")
		return
	}

	miniAppSvc := di.GlobalContainer.MiniAppService

	exists, _ := miniAppSvc.IsAppIDExists(req.AppID)
	if exists {
		response.BadRequest(c, "AppID已存在")
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

	if err := miniAppSvc.CreateMiniApp(&miniApp); err != nil {
		response.InternalServerError(c, "创建小程序失败")
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
		response.BadRequest(c, "参数错误")
		return
	}

	miniAppSvc := di.GlobalContainer.MiniAppService
	miniApp, err := miniAppSvc.GetMiniApp(id)
	if err != nil {
		response.NotFound(c, "小程序不存在")
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

	if err := miniAppSvc.UpdateMiniApp(miniApp); err != nil {
		response.InternalServerError(c, "更新小程序失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": miniApp,
	})
}

func DeleteMiniApp(c *gin.Context) {
	id := c.Param("id")

	miniAppSvc := di.GlobalContainer.MiniAppService
	if err := miniAppSvc.DeleteMiniApp(id); err != nil {
		response.InternalServerError(c, "删除小程序失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "删除小程序成功",
	})
}

func GetNotes(c *gin.Context) {
	userID, _ := c.Get("user_id")

	noteSvc := di.GlobalContainer.NoteService
	notes, err := noteSvc.GetNotes(userID.(uint))
	if err != nil {
		response.InternalServerError(c, "获取笔记列表失败")
		return
	}

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
		response.BadRequest(c, "无效的笔记ID")
		return
	}

	noteSvc := di.GlobalContainer.NoteService
	note, err := noteSvc.GetNote(uint(noteID), userID.(uint))
	if err != nil {
		response.NotFound(c, "笔记不存在")
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
		response.BadRequest(c, "参数错误")
		return
	}

	style := req.Style
	if style == "" {
		style = "{}"
	}
	if req.Color != "" {
		var styleMap map[string]interface{}
		if err := json.Unmarshal([]byte(style), &styleMap); err == nil {
			styleMap["color"] = req.Color
			if styleBytes, err := json.Marshal(styleMap); err == nil {
				style = string(styleBytes)
			}
		}
	}

	noteSvc := di.GlobalContainer.NoteService
	note := model.Note{
		UserID:  userID.(uint),
		Title:   req.Title,
		Content: req.Content,
		Type:    req.Type,
		Style:   style,
	}
	noteSvc.CreateNote(&note)

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
		response.BadRequest(c, "无效的笔记ID")
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
		response.BadRequest(c, "参数错误")
		return
	}

	noteSvc := di.GlobalContainer.NoteService
	note, err := noteSvc.GetNote(uint(noteID), userID.(uint))
	if err != nil {
		response.NotFound(c, "笔记不存在")
		return
	}

	note.Title = req.Title
	note.Content = req.Content
	note.Type = req.Type

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
	noteSvc.UpdateNote(note)

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
		response.BadRequest(c, "无效的笔记ID")
		return
	}

	noteSvc := di.GlobalContainer.NoteService
	if err := noteSvc.DeleteNote(uint(noteID), userID.(uint)); err != nil {
		response.InternalServerError(c, "删除笔记失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "删除笔记成功",
	})
}
