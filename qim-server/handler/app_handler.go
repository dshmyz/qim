package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/dshmyz/qim/qim-server/di"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/response"
	"github.com/dshmyz/qim/qim-server/service"

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

// appFrontend 是返回给前端的 App 结构，字段使用 camelCase，
// 与 qim-admin 的 App 接口定义保持一致。
type appFrontend struct {
	ID              uint   `json:"id"`
	Name            string `json:"name"`
	Code            string `json:"code"`
	Icon            string `json:"icon"`
	Category        string `json:"category"`
	URL             string `json:"url"`
	OpenType        string `json:"openType"`
	IsGlobal        bool   `json:"isGlobal"`
	ScopeType       string `json:"scopeType"`
	ScopeValue      string `json:"scopeValue"`
	AvailableOrgIDs string `json:"availableOrgIDs"`
	Status          string `json:"status"`
	CreatedAt       string `json:"createdAt"`
	UpdatedAt       string `json:"updatedAt"`
}

// appToFrontend 将 model.App 转换为前端期望的 camelCase 结构。
func appToFrontend(app *model.App) appFrontend {
	result := appFrontend{
		ID:              app.ID,
		Name:            app.Name,
		Code:            app.Code,
		Icon:            app.Icon,
		Category:        app.Category,
		URL:             app.URL,
		OpenType:        app.OpenType,
		IsGlobal:        app.IsGlobal,
		ScopeType:       app.ScopeType,
		ScopeValue:      app.ScopeValue,
		AvailableOrgIDs: app.AvailableOrgIDs,
		Status:          app.Status,
	}
	if !app.CreatedAt.IsZero() {
		result.CreatedAt = app.CreatedAt.Format("2006-01-02 15:04:05")
	}
	if !app.UpdatedAt.IsZero() {
		result.UpdatedAt = app.UpdatedAt.Format("2006-01-02 15:04:05")
	}
	return result
}

// appsToFrontend 批量转换 App 列表为前端结构。
func appsToFrontend(apps []model.App) []appFrontend {
	result := make([]appFrontend, 0, len(apps))
	for i := range apps {
		result = append(result, appToFrontend(&apps[i]))
	}
	return result
}

// appCreateRequest 同时兼容 camelCase 与 snake_case 的创建应用请求体。
// 由于 encoding/json 一个字段只能有一个 json tag，这里通过自定义
// UnmarshalJSON 实现双格式兼容：先按 snake_case 解析，再用 camelCase 覆盖。
type appCreateRequest struct {
	Name            string
	Icon            string
	Category        string
	URL             string
	Status          string
	OpenType        string
	IsGlobal        bool
	ScopeType       string
	ScopeValue      string
	AvailableOrgIDs string
}

func (r *appCreateRequest) UnmarshalJSON(data []byte) error {
	// snake_case 版本
	snake := struct {
		Name            string `json:"name"`
		Icon            string `json:"icon"`
		Category        string `json:"category"`
		URL             string `json:"url"`
		Status          string `json:"status"`
		OpenType        string `json:"open_type"`
		IsGlobal        bool   `json:"is_global"`
		ScopeType       string `json:"scope_type"`
		ScopeValue      string `json:"scope_value"`
		AvailableOrgIDs string `json:"available_org_ids"`
	}{}
	if err := json.Unmarshal(data, &snake); err != nil {
		return err
	}

	// camelCase 版本（仅含命名不一致的字段，使用指针判断是否提供）
	camel := struct {
		OpenType        *string `json:"openType"`
		IsGlobal        *bool   `json:"isGlobal"`
		ScopeType       *string `json:"scopeType"`
		ScopeValue      *string `json:"scopeValue"`
		AvailableOrgIDs *string `json:"availableOrgIDs"`
	}{}
	if err := json.Unmarshal(data, &camel); err != nil {
		return err
	}

	r.Name = snake.Name
	r.Icon = snake.Icon
	r.Category = snake.Category
	r.URL = snake.URL
	r.Status = snake.Status
	r.OpenType = snake.OpenType
	r.IsGlobal = snake.IsGlobal
	r.ScopeType = snake.ScopeType
	r.ScopeValue = snake.ScopeValue
	r.AvailableOrgIDs = snake.AvailableOrgIDs

	// camelCase 覆盖 snake_case（如果同时存在，以前端主流的 camelCase 为准）
	if camel.OpenType != nil {
		r.OpenType = *camel.OpenType
	}
	if camel.IsGlobal != nil {
		r.IsGlobal = *camel.IsGlobal
	}
	if camel.ScopeType != nil {
		r.ScopeType = *camel.ScopeType
	}
	if camel.ScopeValue != nil {
		r.ScopeValue = *camel.ScopeValue
	}
	if camel.AvailableOrgIDs != nil {
		r.AvailableOrgIDs = *camel.AvailableOrgIDs
	}
	return nil
}

// appUpdateRequest 同时兼容 camelCase 与 snake_case 的更新应用请求体。
type appUpdateRequest struct {
	Name            *string
	Icon            *string
	Category        *string
	URL             *string
	Status          *string
	OpenType        *string
	IsGlobal        *bool
	ScopeType       *string
	ScopeValue      *string
	AvailableOrgIDs *string
}

func (r *appUpdateRequest) UnmarshalJSON(data []byte) error {
	// snake_case 版本
	snake := struct {
		Name            *string `json:"name"`
		Icon            *string `json:"icon"`
		Category        *string `json:"category"`
		URL             *string `json:"url"`
		Status          *string `json:"status"`
		OpenType        *string `json:"open_type"`
		IsGlobal        *bool   `json:"is_global"`
		ScopeType       *string `json:"scope_type"`
		ScopeValue      *string `json:"scope_value"`
		AvailableOrgIDs *string `json:"available_org_ids"`
	}{}
	if err := json.Unmarshal(data, &snake); err != nil {
		return err
	}

	// camelCase 版本
	camel := struct {
		OpenType        *string `json:"openType"`
		IsGlobal        *bool   `json:"isGlobal"`
		ScopeType       *string `json:"scopeType"`
		ScopeValue      *string `json:"scopeValue"`
		AvailableOrgIDs *string `json:"availableOrgIDs"`
	}{}
	if err := json.Unmarshal(data, &camel); err != nil {
		return err
	}

	r.Name = snake.Name
	r.Icon = snake.Icon
	r.Category = snake.Category
	r.URL = snake.URL
	r.Status = snake.Status
	r.OpenType = snake.OpenType
	r.IsGlobal = snake.IsGlobal
	r.ScopeType = snake.ScopeType
	r.ScopeValue = snake.ScopeValue
	r.AvailableOrgIDs = snake.AvailableOrgIDs

	// camelCase 覆盖 snake_case
	if camel.OpenType != nil {
		r.OpenType = camel.OpenType
	}
	if camel.IsGlobal != nil {
		r.IsGlobal = camel.IsGlobal
	}
	if camel.ScopeType != nil {
		r.ScopeType = camel.ScopeType
	}
	if camel.ScopeValue != nil {
		r.ScopeValue = camel.ScopeValue
	}
	if camel.AvailableOrgIDs != nil {
		r.AvailableOrgIDs = camel.AvailableOrgIDs
	}
	return nil
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
			"list":     appsToFrontend(result.List),
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
		"data": appToFrontend(app),
	})
}

func GetAllApps(c *gin.Context) {
	appSvc := di.GlobalContainer.AppService

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	isGlobal := c.Query("is_global")
	status := c.Query("status")
	name := c.Query("name")

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
		Name:     name,
	})
	if err != nil {
		response.InternalServerError(c, "获取应用列表失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":     appsToFrontend(result.List),
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
		"data": appsToFrontend(result),
	})
}

func CreateApp(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req appCreateRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	if req.Name == "" {
		response.BadRequest(c, "应用名称不能为空")
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
		UserID:          userID.(uint),
		Name:            req.Name,
		Icon:            req.Icon,
		Category:        req.Category,
		URL:             req.URL,
		Status:          req.Status,
		OpenType:        req.OpenType,
		IsGlobal:        req.IsGlobal,
		ScopeType:       req.ScopeType,
		ScopeValue:      req.ScopeValue,
		AvailableOrgIDs: req.AvailableOrgIDs,
	}
	if err := appSvc.CreateApp(&app); err != nil {
		response.InternalServerError(c, "创建应用失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": appToFrontend(&app),
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

	var req appUpdateRequest

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

	if req.IsGlobal != nil && *req.IsGlobal != app.IsGlobal {
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

	if req.Name != nil {
		app.Name = *req.Name
	}
	if req.Icon != nil {
		app.Icon = *req.Icon
	}
	if req.Category != nil {
		app.Category = *req.Category
	}
	if req.URL != nil {
		app.URL = *req.URL
	}
	if req.Status != nil {
		app.Status = *req.Status
	}
	if req.OpenType != nil {
		app.OpenType = *req.OpenType
	}
	if req.IsGlobal != nil {
		app.IsGlobal = *req.IsGlobal
	}

	// 如果请求修改权限范围字段，需要检查管理员权限
	if req.ScopeType != nil || req.ScopeValue != nil || req.AvailableOrgIDs != nil {
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
		if req.ScopeType != nil {
			app.ScopeType = *req.ScopeType
		}
		if req.ScopeValue != nil {
			app.ScopeValue = *req.ScopeValue
		}
		if req.AvailableOrgIDs != nil {
			app.AvailableOrgIDs = *req.AvailableOrgIDs
		}
	}

	appSvc.UpdateApp(app)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": appToFrontend(app),
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

	list := make([]gin.H, 0, len(result.List))
	for _, ma := range result.List {
		list = append(list, miniAppToFrontend(ma))
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":     list,
			"total":    result.Total,
			"page":     result.Page,
			"pageSize": result.PageSize,
		},
	})
}

// miniAppToFrontend 将 MiniApp 模型转换为前端期望的 camelCase 格式
func miniAppToFrontend(ma model.MiniApp) gin.H {
	return gin.H{
		"id":          ma.ID,
		"appID":       ma.AppID,
		"name":        ma.Name,
		"description": ma.Description,
		"icon":        ma.Icon,
		"path":        ma.Path,
		"status":      ma.Status,
		"permissions": ma.Permissions,
		"createdAt":   ma.CreatedAt.Format("2006-01-02 15:04:05"),
		"updatedAt":   ma.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
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
		AppID       string `json:"app_id"`
		AppIDCamel  string `json:"appID"`
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

	// 兼容 camelCase 和 snake_case
	appID := req.AppID
	if appID == "" {
		appID = req.AppIDCamel
	}
	if appID == "" {
		response.BadRequest(c, "appID 不能为空")
		return
	}

	miniAppSvc := di.GlobalContainer.MiniAppService

	exists, _ := miniAppSvc.IsAppIDExists(appID)
	if exists {
		response.BadRequest(c, "AppID已存在")
		return
	}

	miniApp := model.MiniApp{
		AppID:       appID,
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
		"data": miniAppToFrontend(miniApp),
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
		"data": miniAppToFrontend(*miniApp),
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
