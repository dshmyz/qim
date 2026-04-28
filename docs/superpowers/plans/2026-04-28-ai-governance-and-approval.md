# AI 管控与审批流实现计划

> **面向 AI 代理的工作者:** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 为 QIM 系统实现完整的 AI 管控体系，包括 Bot 审批流、模板 Bot、用户自建 Bot 限制、管理员审批面板、用户"我的 Bot"页面。

**架构：** 前后端分离方案。后端新增 Bot 审批 handler + 审计日志 handler，扩展现有 Bot model 和 routes；前端 admin 端改造 AIAssistant.vue 增加审批 Tab，client 端改造 AIAssistantApp.vue 增加模板选择和我的 Bot 面板。按文件职责拆分，每个任务产出独立的、可测试的变更。

**技术栈：**
- 前端: Vue 3 + TypeScript + Element Plus (admin) / 原生 CSS (client)
- 后端: Go + Gin + GORM
- 数据库: SQLite (GORM AutoMigrate)
- 状态管理: 前端 localStorage + axios

---

# 文件结构

## 后端（qim-server）

| 文件 | 操作 | 职责 |
|------|------|------|
| `qim-server/model/model.go` | 修改 | 给 Bot 和 AIConfig model 加新字段，新增 AIUsageLog model |
| `qim-server/app/init.go` | 修改 | AutoMigrate 注册新表 |
| `qim-server/handler/misc_handler.go` | 修改 | 扩展 GetBots 支持过滤，新增用户侧 Bot API |
| `qim-server/handler/bot_approval_handler.go` | 新建 | 管理员审批 API（列表/通过/拒绝） |
| `qim-server/handler/bot_creation_handler.go` | 新建 | 用户创建/编辑/删除 Bot，数量限制校验 |
| `qim-server/handler/ai_usage_handler.go` | 新建 | AI 使用审计日志 API |
| `qim-server/app/routes.go` | 修改 | 注册新路由 |

## 前端 Admin（qim-admin）

| 文件 | 操作 | 职责 |
|------|------|------|
| `qim-admin/src/types/index.ts` | 修改 | 新增 Bot 审批相关 TypeScript 类型 |
| `qim-admin/src/api/aiBots.ts` | 修改 | 增加审批相关 API 函数 |
| `qim-admin/src/views/AIAssistant.vue` | 修改 | 增加 Tab 切换（AI 助手管理 / Bot 审批），提取子组件 |
| `qim-admin/src/views/BotApprovalPanel.vue` | 新建 | Bot 审批面板组件 |

## 前端 Client（qim-client）

| 文件 | 操作 | 职责 |
|------|------|------|
| `qim-client/src/components/apps/AIAssistantApp.vue` | 修改 | 增加 Tab 导航、模板选择入口、我的 Bot 入口 |
| `qim-client/src/components/apps/MyBotsPanel.vue` | 新建 | 我的机器人面板 |
| `qim-client/src/composables/useBots.ts` | 新建 | Bot 相关组合式函数（创建、查询、数量检查） |

---

# 角色一：🗄️ 后端开发

## 任务 B1：扩展 Bot Model 和新增 AIUsageLog Model

**文件：**
- 修改：`qim-server/model/model.go` — 给 Bot struct 加字段，给 AIConfig struct 加字段，新增 AIUsageLog struct

**步骤：**

- [ ] **步骤 1：给 Bot struct 增加审批相关字段**

在 `qim-server/model/model.go` 的 `Bot` struct 中找到 `UpdatedAt` 字段，在其后添加新字段：

```go
type Bot struct {
	ID             uint      `json:"id" gorm:"primarykey"`
	Name           string    `json:"name" gorm:"size:100;not null"`
	Avatar         string    `json:"avatar" gorm:"size:500"`
	Description    string    `json:"description" gorm:"type:text"`
	Type           string    `json:"type" gorm:"size:50;not null"` // system, custom, ai
	Config         string    `json:"config" gorm:"type:text"`      // JSON配置
	IsActive       bool      `json:"is_active" gorm:"default:true"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	// 新增字段
	ApprovalStatus string    `json:"approval_status" gorm:"size:20;default:'approved'"` // pending, approved, rejected
	CreatorID      uint      `json:"creator_id" gorm:"default:0"`                       // 0=系统创建
	CreatorName    string    `json:"creator_name" gorm:"size:100;default:''"`
	RejectReason   string    `json:"reject_reason" gorm:"type:text"`
	IsTemplate     bool      `json:"is_template" gorm:"default:false"`
}
```

- [ ] **步骤 2：给 AIConfig struct 增加管控字段**

在 `AIConfig` struct 的 `BytedanceBaseURL` 后添加：

```go
type AIConfig struct {
	// ... 已有字段 ...
	BytedanceBaseURL string    `json:"bytedance_base_url" gorm:"size:500"`
	AnthropicAPIKey  string    `json:"anthropic_api_key" gorm:"size:500"`
	// 新增字段
	AIEnabled  bool `json:"ai_enabled" gorm:"default:true"`   // 是否可以使用 AI
	DailyLimit int  `json:"daily_limit" gorm:"default:0"`    // 每日调用限制，0=不限制
}
```

- [ ] **步骤 3：新增 AIUsageLog model**

在 model.go 中 Bot 定义之后添加：

```go
type AIUsageLog struct {
	ID             uint      `json:"id" gorm:"primarykey"`
	UserID         uint      `json:"user_id" gorm:"not null"`
	BotID          uint      `json:"bot_id" gorm:"not null"`
	MessagePreview string    `json:"message_preview" gorm:"size:100"`
	CallType       string    `json:"call_type" gorm:"size:20"` // chat, ops
	CreatedAt      time.Time `json:"created_at"`
}
```

- [ ] **步骤 4：在 AutoMigrate 中注册新表**

修改 `qim-server/app/init.go`，在 `db.AutoMigrate` 调用中 `&model.Group{}` 后添加：

```go
&model.AIUsageLog{},
```

GORM AutoMigrate 会自动为已有表添加新列，无需手动迁移脚本。

- [ ] **步骤 5：Commit**

```bash
git add qim-server/model/model.go qim-server/app/init.go
git commit -m "feat: 扩展 Bot model 增加审批字段，新增 AIUsageLog model"
```

---

## 任务 B2：实现用户侧 Bot API（创建/编辑/删除/查询/数量限制）

**文件：**
- 创建：`qim-server/handler/bot_creation_handler.go`
- 修改：`qim-server/handler/misc_handler.go` — 改造 GetBots

**步骤：**

- [ ] **步骤 1：创建 bot_creation_handler.go 骨架**

```go
package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"qim-server/database"
	"qim-server/model"
	"github.com/gin-gonic/gin"
	"qim-server/middleware"
)

// CreateBotRequest 创建 Bot 请求体
type CreateBotRequest struct {
	Name          string `json:"name" binding:"required"`
	Description   string `json:"description" binding:"required"`
	Type          string `json:"type" binding:"required"` // ai, custom
	Provider      string `json:"provider"`
	CustomModelURL string `json:"custom_model_url"`
	LobsterURL    string `json:"lobster_url"`
	Avatar        string `json:"avatar"`
	IsTemplate    bool   `json:"is_template"`
	Config        map[string]interface{} `json:"config"`
}

// GetMyBots 获取我创建的 Bot 列表
func GetMyBots(c *gin.Context) {
	userID := middleware.GetUserID(c)
	db := database.GetDB()

	var bots []model.Bot
	db.Where("creator_id = ?", userID).Order("created_at DESC").Find(&bots)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": bots,
	})
}

// GetMyBotCount 获取我已创建的 Bot 数量
func GetMyBotCount(c *gin.Context) {
	userID := middleware.GetUserID(c)
	db := database.GetDB()

	var count int64
	db.Model(&model.Bot{}).Where("creator_id = ? AND type IN ('custom', 'ai')", userID).Count(&count)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{"count": count},
	})
}

// GetTemplates 获取模板 Bot 列表
func GetTemplates(c *gin.Context) {
	db := database.GetDB()

	var bots []model.Bot
	db.Where("is_template = ? AND is_active = ? AND approval_status = ?", true, true, "approved").Find(&bots)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": bots,
	})
}

// CreateBot 创建 Bot
func CreateBot(c *gin.Context) {
	userID := middleware.GetUserID(c)
	db := database.GetDB()

	var req CreateBotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请求参数错误: " + err.Error()})
		return
	}

	// 检查用户是否已达到创建上限（模板 Bot 不计入限制）
	if !req.IsTemplate {
		var count int64
		db.Model(&model.Bot{}).Where("creator_id = ? AND type IN ('custom', 'ai')", userID).Count(&count)
		if count >= getMaxBotsPerUser(db) {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    "BOT_LIMIT_EXCEEDED",
				"message": "已达到创建上限，请联系管理员",
			})
			return
		}
	}

	// 构建 Config JSON
	configJSON, _ := json.Marshal(req.Config)
	if req.Config == nil {
		configJSON = []byte("{}")
	}

	// 判断审批状态
	approvalStatus := "approved" // 系统 Bot 和模板 Bot 直接通过
	creatorID := userID
	creatorName := ""

	if !req.IsTemplate && creatorID != 0 {
		approvalStatus = "pending" // 用户自建 Bot 需要审批
	}

	bot := model.Bot{
		Name:           req.Name,
		Description:    req.Description,
		Type:           req.Type,
		Avatar:         req.Avatar,
		Config:         string(configJSON),
		IsActive:       true,
		ApprovalStatus: approvalStatus,
		CreatorID:      creatorID,
		CreatorName:    creatorName,
		IsTemplate:     req.IsTemplate,
	}

	db.Create(&bot)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": bot,
	})
}

// UpdateMyBot 更新我的 Bot（仅允许待审批和已拒绝状态）
func UpdateMyBot(c *gin.Context) {
	userID := middleware.GetUserID(c)
	botID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	db := database.GetDB()

	var bot model.Bot
	if err := db.Where("id = ? AND creator_id = ?", botID, userID).First(&bot).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "Bot 不存在或无权操作"})
		return
	}

	// 仅允许待审批和已拒绝状态的 Bot 编辑
	if bot.ApprovalStatus != "pending" && bot.ApprovalStatus != "rejected" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "仅可编辑待审批或已拒绝的 Bot"})
		return
	}

	var req CreateBotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请求参数错误"})
		return
	}

	configJSON, _ := json.Marshal(req.Config)
	if req.Config == nil {
		configJSON = []byte("{}")
	}

	db.Model(&bot).Updates(map[string]interface{}{
		"name":        req.Name,
		"description": req.Description,
		"type":        req.Type,
		"avatar":      req.Avatar,
		"config":      string(configJSON),
	})

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": bot})
}

// DeleteMyBot 删除我的 Bot
func DeleteMyBot(c *gin.Context) {
	userID := middleware.GetUserID(c)
	botID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	db := database.GetDB()

	result := db.Where("id = ? AND creator_id = ?", botID, userID).Delete(&model.Bot{})
	if result.Error != nil || result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "Bot 不存在或无权操作"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// getMaxBotsPerUser 从系统配置获取每个用户的最大 Bot 数量
func getMaxBotsPerUser(db *gorm.DB) int64 {
	// 默认 5，后续从 system_configs 表读取
	return 5
}
```

- [ ] **步骤 2：改造 GetBots 支持过滤**

修改 `qim-server/handler/misc_handler.go` 的 `GetBots` 函数：

```go
func GetBots(c *gin.Context) {
	db := database.GetDB()

	var bots []model.Bot
	// 返回：系统 Bot + 模板 Bot + 已审批通过的用户自建 Bot
	db.Where(
		"(creator_id = 0 AND is_active = ?) OR (is_template = ? AND is_active = ? AND approval_status = ?) OR (approval_status = ? AND is_active = ?)",
		true, true, true, "approved", "approved", true,
	).Find(&bots)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": bots,
	})
}
```

- [ ] **步骤 3：在 routes.go 中注册用户侧 Bot 路由**

修改 `qim-server/app/routes.go`，找到 `authed.GET("/bots", handler.GetBots)` 这行，在其后添加：

```go
authed.GET("/bots", handler.GetBots)
authed.GET("/bots/templates", handler.GetTemplates)
authed.GET("/bots/my", handler.GetMyBots)
authed.GET("/bots/my-count", handler.GetMyBotCount)
authed.POST("/bots", handler.CreateBot)
authed.PUT("/bots/:id", handler.UpdateMyBot)
authed.DELETE("/bots/:id", handler.DeleteMyBot)
```

- [ ] **步骤 4：添加 gorm 依赖导入**

确保 `bot_creation_handler.go` 顶部的 import 包含：

```go
import (
	"encoding/json"
	"net/http"
	"strconv"
	"qim-server/database"
	"qim-server/model"
	"qim-server/middleware"
	"gorm.io/gorm"
)
```

- [ ] **步骤 5：编译验证**

```bash
cd qim-server
go build ./...
```

预期：编译成功，无错误。

- [ ] **步骤 6：Commit**

```bash
git add qim-server/handler/bot_creation_handler.go qim-server/handler/misc_handler.go qim-server/app/routes.go
git commit -m "feat: 实现用户侧 Bot API（创建/编辑/删除/查询/数量限制）"
```

---

## 任务 B3：实现管理员 Bot 审批 API

**文件：**
- 创建：`qim-server/handler/bot_approval_handler.go`

**步骤：**

- [ ] **步骤 1：创建 bot_approval_handler.go**

```go
package handler

import (
	"net/http"
	"strconv"
	"qim-server/database"
	"qim-server/model"
	"github.com/gin-gonic/gin"
)

// BotApprovalItem 审批列表项
type BotApprovalItem struct {
	model.Bot
	CreatorAvatar  string `json:"creator_avatar"`
	CreatorBotCount int64  `json:"creator_bot_count"`
}

// GetBotApprovals 获取 Bot 审批列表
func GetBotApprovals(c *gin.Context) {
	db := database.GetDB()

	status := c.DefaultQuery("status", "pending") // pending, approved, rejected, all

	var bots []model.Bot
	query := db.Model(&model.Bot{}).Where("creator_id != 0") // 只看用户创建的

	if status != "all" {
		query = query.Where("approval_status = ?", status)
	}

	query.Order("created_at DESC").Find(&bots)

	// 组装审批列表项
	items := make([]BotApprovalItem, 0, len(bots))
	for _, bot := range bots {
		item := BotApprovalItem{Bot: bot}
		// 查询创建者头像
		var creator model.User
		db.Where("id = ?", bot.CreatorID).First(&creator)
		item.CreatorAvatar = creator.Avatar
		// 查询创建者已创建 Bot 数量
		var count int64
		db.Model(&model.Bot{}).Where("creator_id = ?", bot.CreatorID).Count(&count)
		item.CreatorBotCount = count
		items = append(items, item)
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": items,
	})
}

// ApproveBot 通过 Bot 申请
func ApproveBot(c *gin.Context) {
	botID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	db := database.GetDB()

	var bot model.Bot
	if err := db.Where("id = ? AND approval_status = ?", botID, "pending").First(&bot).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "Bot 不存在或无需审批"})
		return
	}

	db.Model(&bot).Updates(map[string]interface{}{
		"approval_status": "approved",
		"is_active":       true,
	})

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "审批通过", "data": bot})
}

// RejectBot 拒绝 Bot 申请
type RejectBotRequest struct {
	Reason string `json:"reason"`
}

func RejectBot(c *gin.Context) {
	botID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	db := database.GetDB()

	var bot model.Bot
	if err := db.Where("id = ? AND approval_status = ?", botID, "pending").First(&bot).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "Bot 不存在或无需审批"})
		return
	}

	var req RejectBotRequest
	c.ShouldBindJSON(&req)

	db.Model(&bot).Updates(map[string]interface{}{
		"approval_status": "rejected",
		"is_active":       false,
		"reject_reason":   req.Reason,
	})

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已拒绝", "data": bot})
}
```

- [ ] **步骤 2：在 routes.go 中注册审批路由**

在 `admin := authed.Group("/admin")` 块中，添加：

```go
admin.GET("/bot-approvals", handler.GetBotApprovals)
admin.PATCH("/bot-approvals/:id/approve", handler.ApproveBot)
admin.PATCH("/bot-approvals/:id/reject", handler.RejectBot)
admin.GET("/ai-usage-logs", handler.GetAIUsageLogs)
```

- [ ] **步骤 3：编译验证**

```bash
cd qim-server
go build ./...
```

预期：编译成功。

- [ ] **步骤 4：Commit**

```bash
git add qim-server/handler/bot_approval_handler.go qim-server/app/routes.go
git commit -m "feat: 实现管理员 Bot 审批 API（列表/通过/拒绝）"
```

---

## 任务 B4：实现 AI 使用审计日志 API

**文件：**
- 创建：`qim-server/handler/ai_usage_handler.go`

**步骤：**

- [ ] **步骤 1：创建 ai_usage_handler.go**

```go
package handler

import (
	"net/http"
	"qim-server/database"
	"qim-server/model"
	"github.com/gin-gonic/gin"
)

// LogAIUsage 记录 AI 使用日志（内部调用）
func LogAIUsage(userID uint, botID uint, messagePreview string, callType string) {
	db := database.GetDB()
	log := model.AIUsageLog{
		UserID:         userID,
		BotID:          botID,
		MessagePreview: messagePreview,
		CallType:       callType,
	}
	db.Create(&log)
}

// GetAIUsageLogs 获取 AI 使用审计日志（管理员）
func GetAIUsageLogs(c *gin.Context) {
	db := database.GetDB()

	// 分页
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	offset := (page - 1) * pageSize

	// 筛选
	userID := c.Query("user_id")
	botID := c.Query("bot_id")
	callType := c.Query("call_type")

	query := db.Model(&model.AIUsageLog{})
	if userID != "" {
		query = query.Where("user_id = ?", userID)
	}
	if botID != "" {
		query = query.Where("bot_id = ?", botID)
	}
	if callType != "" {
		query = query.Where("call_type = ?", callType)
	}

	var total int64
	query.Count(&total)

	var logs []model.AIUsageLog
	query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&logs)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":     logs,
			"total":    total,
			"page":     page,
			"pageSize": pageSize,
		},
	})
}
```

- [ ] **步骤 2：添加缺失的 import**

确保 import 包含：

```go
import (
	"net/http"
	"strconv"
	"qim-server/database"
	"qim-server/model"
	"github.com/gin-gonic/gin"
)
```

- [ ] **步骤 3：编译验证**

```bash
cd qim-server
go build ./...
```

- [ ] **步骤 4：Commit**

```bash
git add qim-server/handler/ai_usage_handler.go qim-server/app/routes.go
git commit -m "feat: 实现 AI 使用审计日志 API"
```

---

# 角色二：🎨 前端 Admin 开发

## 任务 A1：扩展类型定义和 API 函数

**文件：**
- 修改：`qim-admin/src/types/index.ts`
- 修改：`qim-admin/src/api/aiBots.ts`

**步骤：**

- [ ] **步骤 1：在 types/index.ts 中新增审批相关类型**

在 `AIBot` 接口后添加：

```typescript
export interface AIBot {
  id: number
  name: string
  avatar: string
  description: string
  systemPrompt: string
  status: 'active' | 'inactive'
  conversationCount: number
  createdAt: string
  // 新增审批相关字段
  approvalStatus?: 'pending' | 'approved' | 'rejected'
  creatorId?: number
  creatorName?: string
  creatorAvatar?: string
  rejectReason?: string
  isTemplate?: boolean
  creatorBotCount?: number
}

export interface BotApprovalItem {
  id: number
  name: string
  avatar: string
  description: string
  type: string
  creatorName: string
  creatorAvatar: string
  creatorBotCount: number
  approvalStatus: 'pending' | 'approved' | 'rejected'
  rejectReason?: string
  createdAt: string
}

export interface AIUsageLog {
  id: number
  userId: number
  botId: number
  messagePreview: string
  callType: string
  createdAt: string
}
```

- [ ] **步骤 2：在 aiBots.ts 中新增审批 API**

在文件末尾添加：

```typescript
// Bot 审批相关
export const getBotApprovals = (params?: { status?: string; page?: number; pageSize?: number }): Promise<AxiosResponse<ApiResponse<PaginatedResponse<AIBot>>>> => {
  return request({
    url: '/v1/admin/bot-approvals',
    method: 'get',
    params,
  })
}

export const approveBot = (id: number): Promise<AxiosResponse<ApiResponse<void>>> => {
  return request({
    url: `/v1/admin/bot-approvals/${id}/approve`,
    method: 'patch',
  })
}

export const rejectBot = (id: number, data: { reason?: string }): Promise<AxiosResponse<ApiResponse<void>>> => {
  return request({
    url: `/v1/admin/bot-approvals/${id}/reject`,
    method: 'patch',
    data,
  })
}

// AI 使用审计日志
export const getAIUsageLogs = (params?: { userId?: string; botId?: string; callType?: string; page?: number; pageSize?: number }): Promise<AxiosResponse<ApiResponse<PaginatedResponse<AIUsageLog>>>> => {
  return request({
    url: '/v1/admin/ai-usage-logs',
    method: 'get',
    params,
  })
}
```

- [ ] **步骤 3：Commit**

```bash
git add qim-admin/src/types/index.ts qim-admin/src/api/aiBots.ts
git commit -m "feat(admin): 增加 Bot 审批和审计日志类型定义与 API"
```

---

## 任务 A2：重构 AIAssistant.vue 增加 Tab 和 BotApprovalPanel 组件

**文件：**
- 修改：`qim-admin/src/views/AIAssistant.vue`
- 新建：`qim-admin/src/views/BotApprovalPanel.vue`

**步骤：**

- [ ] **步骤 1：创建 BotApprovalPanel.vue**

```vue
<template>
  <div class="bot-approval-panel">
    <!-- 筛选栏 -->
    <div class="filter-bar">
      <el-radio-group v-model="filterStatus" @change="fetchApprovals">
        <el-radio-button value="pending">待审批 ({{ pendingCount }})</el-radio-button>
        <el-radio-button value="approved">已通过</el-radio-button>
        <el-radio-button value="rejected">已拒绝</el-radio-button>
        <el-radio-button value="all">全部</el-radio-button>
      </el-radio-group>
    </div>

    <!-- 审批列表 -->
    <el-table :data="approvals" v-loading="loading">
      <el-table-column label="申请人" min-width="140">
        <template #default="{ row }">
          <div class="applicant-cell">
            <el-avatar :size="28" :src="row.creator_avatar">{{ row.creator_name?.charAt(0) }}</el-avatar>
            <span>{{ row.creator_name || '未知' }}</span>
          </div>
        </template>
      </el-table-column>
      <el-table-column label="Bot 名称" min-width="160">
        <template #default="{ row }">
          <div class="bot-cell">
            <el-avatar :size="28" :src="row.avatar">{{ row.name.charAt(0) }}</el-avatar>
            <span class="bot-name">{{ row.name }}</span>
          </div>
        </template>
      </el-table-column>
      <el-table-column label="类型" width="100">
        <template #default="{ row }">
          <el-tag :type="row.type === 'ai' ? 'primary' : 'info'" size="small">
            {{ row.type === 'ai' ? 'AI' : '自定义' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="description" label="描述" min-width="200" show-overflow-tooltip />
      <el-table-column label="已创建" width="80">
        <template #default="{ row }">
          <el-tag size="small">{{ row.creator_bot_count }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="statusTagType(row.approval_status)" size="small">
            {{ statusLabel(row.approval_status) }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="createdAt" label="创建时间" width="180" />
      <el-table-column label="操作" width="220" fixed="right">
        <template #default="{ row }">
          <el-button size="small" @click="handleViewPrompt(row)">提示词</el-button>
          <template v-if="row.approval_status === 'pending'">
            <el-button size="small" type="success" @click="handleApprove(row)">通过</el-button>
            <el-button size="small" type="danger" @click="handleReject(row)">拒绝</el-button>
          </template>
          <span v-else-if="row.approval_status === 'rejected'" class="reject-reason">
            {{ row.reject_reason || '无' }}
          </span>
        </template>
      </el-table-column>
    </el-table>

    <!-- 分页 -->
    <div class="pagination-container">
      <el-pagination
        v-model:current-page="pagination.page"
        v-model:page-size="pagination.pageSize"
        :total="pagination.total"
        :page-sizes="[10, 20, 50]"
        layout="total, sizes, prev, pager, next"
        @size-change="fetchApprovals"
        @current-change="fetchApprovals"
      />
    </div>

    <!-- 查看提示词对话框 -->
    <el-dialog v-model="promptDialogVisible" title="Bot 配置详情" width="600px">
      <div v-if="selectedBot" class="prompt-content">
        <p><strong>名称：</strong>{{ selectedBot.name }}</p>
        <p><strong>描述：</strong>{{ selectedBot.description }}</p>
        <p><strong>类型：</strong>{{ selectedBot.type }}</p>
        <p><strong>系统提示词：</strong></p>
        <pre>{{ selectedBot.config }}</pre>
      </div>
    </el-dialog>

    <!-- 拒绝原因输入对话框 -->
    <el-dialog v-model="rejectDialogVisible" title="拒绝原因" width="400px">
      <el-input v-model="rejectReason" type="textarea" :rows="4" placeholder="请输入拒绝原因（可选）" />
      <template #footer>
        <el-button @click="rejectDialogVisible = false">取消</el-button>
        <el-button type="danger" @click="confirmReject">确认拒绝</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getBotApprovals, approveBot, rejectBot } from '@/api/aiBots'
import type { AIBot } from '@/types'

const filterStatus = ref('pending')
const loading = ref(false)
const approvals = ref<AIBot[]>([])
const pendingCount = ref(0)
const selectedBot = ref<AIBot | null>(null)
const promptDialogVisible = ref(false)
const rejectDialogVisible = ref(false)
const rejectReason = ref('')
const rejectingBotId = ref(0)

const pagination = reactive({ page: 1, pageSize: 10, total: 0 })

const fetchApprovals = async () => {
  loading.value = true
  try {
    const { data } = await getBotApprovals({
      status: filterStatus.value,
      page: pagination.page,
      pageSize: pagination.pageSize,
    })
    approvals.value = data.data.list
    pagination.total = data.data.total

    // 更新待审批计数
    if (filterStatus.value !== 'pending') {
      const { data: pendingData } = await getBotApprovals({ status: 'pending', page: 1, pageSize: 1 })
      pendingCount.value = pendingData.data.total
    } else {
      pendingCount.value = pagination.total
    }
  } catch {
    // 错误已在请求拦截器中处理
  } finally {
    loading.value = false
  }
}

const handleApprove = async (row: AIBot) => {
  try {
    await ElMessageBox.confirm(`确定通过「${row.name}」的申请吗？`, '确认通过')
    await approveBot(row.id)
    ElMessage.success('审批通过')
    fetchApprovals()
  } catch {
    // 用户取消或请求失败
  }
}

const handleReject = (row: AIBot) => {
  rejectingBotId.value = row.id
  rejectReason.value = ''
  rejectDialogVisible.value = true
}

const confirmReject = async () => {
  try {
    await rejectBot(rejectingBotId.value, { reason: rejectReason.value })
    ElMessage.success('已拒绝')
    rejectDialogVisible.value = false
    fetchApprovals()
  } catch {
    // 错误已在请求拦截器中处理
  }
}

const handleViewPrompt = (row: AIBot) => {
  selectedBot.value = row
  promptDialogVisible.value = true
}

const statusTagType = (status: string) => {
  const map: Record<string, string> = { pending: 'warning', approved: 'success', rejected: 'danger' }
  return map[status] || 'info'
}

const statusLabel = (status: string) => {
  const map: Record<string, string> = { pending: '待审批', approved: '已通过', rejected: '已拒绝' }
  return map[status] || status
}

onMounted(fetchApprovals)
</script>

<style scoped>
.bot-approval-panel {
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
}

.filter-bar {
  display: flex;
  justify-content: flex-end;
}

.applicant-cell, .bot-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.bot-name {
  font-weight: 600;
  color: var(--color-text-primary);
}

.reject-reason {
  font-size: 12px;
  color: var(--color-text-secondary);
}

.pagination-container {
  display: flex;
  justify-content: flex-end;
  padding-top: var(--space-4);
}

.prompt-content pre {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  padding: var(--space-4);
  max-height: 300px;
  overflow-y: auto;
  white-space: pre-wrap;
  word-break: break-word;
  font-size: 13px;
  line-height: 1.6;
}
</style>
```

- [ ] **步骤 2：重构 AIAssistant.vue 增加 Tab**

替换现有 template 和 script，引入 Tab 切换：

```vue
<template>
  <div class="ai-assistant-page">
    <el-tabs v-model="activeTab">
      <el-tab-pane label="AI 助手管理" name="management">
        <AIAssistantList />
      </el-tab-pane>
      <el-tab-pane label="Bot 审批" name="approval">
        <BotApprovalPanel />
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import AIAssistantList from './AIAssistantList.vue'
import BotApprovalPanel from './BotApprovalPanel.vue'

const activeTab = ref('management')
</script>

<style scoped>
.ai-assistant-page {
  padding: var(--space-4);
}
</style>
```

- [ ] **步骤 3：提取现有 AIAssistant.vue 内容为 AIAssistantList.vue**

将当前 `qim-admin/src/views/AIAssistant.vue` 的完整内容复制到新文件 `qim-admin/src/views/AIAssistantList.vue`，保持完全一致（只是文件名变化）。

- [ ] **步骤 4：Commit**

```bash
git add qim-admin/src/views/AIAssistant.vue qim-admin/src/views/AIAssistantList.vue qim-admin/src/views/BotApprovalPanel.vue
git commit -m "feat(admin): AI 助手页面增加 Tab 和 Bot 审批面板"
```

---

# 角色三：🎨 前端 Client 开发

## 任务 C1：创建 useBots 组合式函数

**文件：**
- 新建：`qim-client/src/composables/useBots.ts`

**步骤：**

- [ ] **步骤 1：创建 useBots.ts**

```typescript
import { ref } from 'vue'
import axios from 'axios'

const serverUrl = ref(localStorage.getItem('serverUrl') || '')

function getToken() {
  return localStorage.getItem('token')
}

export function useBots() {
  const loading = ref(false)
  const error = ref('')
  const botCount = ref(0)

  const fetchBots = async () => {
    loading.value = true
    try {
      const response = await axios.get(`${serverUrl.value}/api/v1/bots`, {
        headers: { Authorization: `Bearer ${getToken()}` }
      })
      return response.data.data
    } catch (e: any) {
      error.value = e.message
      return []
    } finally {
      loading.value = false
    }
  }

  const fetchTemplates = async () => {
    loading.value = true
    try {
      const response = await axios.get(`${serverUrl.value}/api/v1/bots/templates`, {
        headers: { Authorization: `Bearer ${getToken()}` }
      })
      return response.data.data
    } catch (e: any) {
      error.value = e.message
      return []
    } finally {
      loading.value = false
    }
  }

  const fetchMyBots = async () => {
    loading.value = true
    try {
      const response = await axios.get(`${serverUrl.value}/api/v1/bots/my`, {
        headers: { Authorization: `Bearer ${getToken()}` }
      })
      return response.data.data
    } catch (e: any) {
      error.value = e.message
      return []
    } finally {
      loading.value = false
    }
  }

  const fetchMyBotCount = async () => {
    try {
      const response = await axios.get(`${serverUrl.value}/api/v1/bots/my-count`, {
        headers: { Authorization: `Bearer ${getToken()}` }
      })
      botCount.value = response.data.data.count
      return botCount.value
    } catch {
      return 0
    }
  }

  const createBot = async (data: Record<string, unknown>) => {
    loading.value = true
    try {
      const response = await axios.post(`${serverUrl.value}/api/v1/bots`, data, {
        headers: { Authorization: `Bearer ${getToken()}` }
      })
      return response.data
    } catch (e: any) {
      error.value = e.message
      throw e
    } finally {
      loading.value = false
    }
  }

  const updateBot = async (id: number, data: Record<string, unknown>) => {
    loading.value = true
    try {
      const response = await axios.put(`${serverUrl.value}/api/v1/bots/${id}`, data, {
        headers: { Authorization: `Bearer ${getToken()}` }
      })
      return response.data
    } catch (e: any) {
      error.value = e.message
      throw e
    } finally {
      loading.value = false
    }
  }

  const deleteBot = async (id: number) => {
    loading.value = true
    try {
      const response = await axios.delete(`${serverUrl.value}/api/v1/bots/${id}`, {
        headers: { Authorization: `Bearer ${getToken()}` }
      })
      return response.data
    } catch (e: any) {
      error.value = e.message
      throw e
    } finally {
      loading.value = false
    }
  }

  return {
    loading,
    error,
    botCount,
    fetchBots,
    fetchTemplates,
    fetchMyBots,
    fetchMyBotCount,
    createBot,
    updateBot,
    deleteBot,
  }
}
```

- [ ] **步骤 2：Commit**

```bash
git add qim-client/src/composables/useBots.ts
git commit -m "feat(client): 创建 useBots 组合式函数"
```

---

## 任务 C2：改造 AIAssistantApp.vue 增加 Tab 导航和模板选择

**文件：**
- 修改：`qim-client/src/components/apps/AIAssistantApp.vue`

**步骤：**

- [ ] **步骤 1：增加 Tab 导航**

在 `AIAssistantApp.vue` 的 bot-selection 区域顶部，替换现有模式/机器人选择结构，增加三个 Tab：

找到 `<div v-if="!showBotChat" class="bot-selection">` 内部结构，替换为：

```vue
<div v-if="!showBotChat" class="bot-selection">
  <!-- Tab 导航 -->
  <div class="bot-tabs">
    <button 
      :class="['bot-tab', { active: activeTab === 'available' }]" 
      @click="activeTab = 'available'"
    >可用机器人</button>
    <button 
      :class="['bot-tab', { active: activeTab === 'my-bots' }]" 
      @click="activeTab = 'my-bots'"
    >我的机器人</button>
    <button 
      :class="['bot-tab', { active: activeTab === 'create' }]" 
      @click="openCreateModal"
    >创建机器人</button>
  </div>

  <!-- 可用机器人 Tab -->
  <div v-show="activeTab === 'available'" class="tab-content">
    <!-- 模式选择保持不变 -->
    <h3>选择模式</h3>
    <div class="mode-list">
      <div class="mode-item" @click="selectMode('chat')">
        <div class="mode-icon"><i class="fas fa-comments"></i></div>
        <div class="mode-info">
          <h4>聊天模式</h4>
          <p>与AI进行日常对话，获取信息和建议</p>
        </div>
      </div>
      <div class="mode-item" @click="selectMode('ops')">
        <div class="mode-icon ops"><i class="fas fa-server"></i></div>
        <div class="mode-info">
          <h4>运维模式</h4>
          <p>智能故障排查、命令生成、日志分析等</p>
        </div>
      </div>
    </div>

    <div class="bot-selection-header">
      <h3>选择机器人</h3>
      <button class="create-bot-btn" @click="openCreateModal">
        <i class="fas fa-plus"></i>
        <span>创建机器人</span>
      </button>
    </div>
    <div class="bot-list">
      <div v-for="bot in bots" :key="bot.id" class="bot-item" @click="selectBot(bot.id)">
        <div class="bot-avatar">
          <img :src="bot.avatar" :alt="bot.name" v-if="bot.avatar">
          <i class="fas fa-robot" v-else></i>
        </div>
        <div class="bot-info">
          <h4>{{ bot.name }}</h4>
          <p>{{ bot.description }}</p>
          <span class="bot-type" :class="bot.type">{{ bot.type === 'ai' ? 'AI 机器人' : '系统机器人' }}</span>
        </div>
      </div>
      <div v-if="bots.length === 0" class="empty-bots">
        <i class="fas fa-robot"></i>
        <p>暂无可用的机器人</p>
      </div>
    </div>
  </div>

  <!-- 我的机器人 Tab -->
  <div v-show="activeTab === 'my-bots'" class="tab-content">
    <MyBotsPanel />
  </div>

  <!-- 创建机器人模态框（通过 openCreateModal 打开） -->
  <div v-if="showCreateBotModal" class="tool-modal">
    <div class="modal-content">
      <!-- 创建方式选择 -->
      <div v-if="!showCreateForm" class="create-method-selector">
        <h3>选择创建方式</h3>
        <div class="method-options">
          <div class="method-option recommended" @click="selectCreateMethod('template')">
            <div class="method-icon"><i class="fas fa-th-large"></i></div>
            <h4>使用模板（推荐）</h4>
            <p>从管理员配置的模板中选择，直接可用</p>
          </div>
          <div class="method-option" @click="selectCreateMethod('custom')">
            <div class="method-icon"><i class="fas fa-cog"></i></div>
            <h4>自定义机器人</h4>
            <p>配置自己的 Prompt 和模型，需审批</p>
          </div>
        </div>
        <button class="close-button full-width" @click="showCreateBotModal = false">取消</button>
      </div>

      <!-- 模板选择 -->
      <div v-else-if="createMethod === 'template' && !showCreateForm" class="template-selector">
        <div class="modal-header">
          <h3>选择模板</h3>
          <button class="close-button" @click="selectCreateMethod(null)"><i class="fas fa-arrow-left"></i></button>
        </div>
        <div class="template-list">
          <div v-for="tpl in templates" :key="tpl.id" class="template-item" @click="createFromTemplate(tpl)">
            <div class="template-avatar">
              <img :src="tpl.avatar" :alt="tpl.name" v-if="tpl.avatar">
              <i class="fas fa-robot" v-else></i>
            </div>
            <div class="template-info">
              <h4>{{ tpl.name }}</h4>
              <p>{{ tpl.description }}</p>
            </div>
          </div>
          <div v-if="templates.length === 0" class="empty-templates">
            <p>暂无可用模板</p>
          </div>
        </div>
      </div>

      <!-- 自定义创建表单（保持现有表单内容） -->
      <div v-else>
        <div class="modal-header">
          <h3>{{ createMethod === 'template' ? '模板创建' : '创建自定义机器人' }}</h3>
          <button class="close-button" @click="showCreateBotModal = false"><i class="fas fa-times"></i></button>
        </div>
        <div class="modal-body">
          <!-- 现有表单字段保持不变 -->
          <div class="form-group">
            <label>机器人名称</label>
            <input v-model="createBotForm.name" type="text" placeholder="请输入机器人名称">
          </div>
          <div class="form-group">
            <label>描述</label>
            <textarea v-model="createBotForm.description" placeholder="请输入机器人描述" rows="3"></textarea>
          </div>
          <div class="form-group" v-if="createMethod === 'custom'">
            <label>机器人类型</label>
            <select v-model="createBotForm.type">
              <option value="ai">AI 机器人</option>
              <option value="custom">自定义机器人</option>
            </select>
          </div>
          <div class="form-group" v-if="createMethod === 'custom' && createBotForm.type === 'ai'">
            <label>AI 提供商</label>
            <select v-model="createBotForm.provider">
              <option value="openai">OpenAI</option>
              <option value="baidu">百度文心一言</option>
              <option value="alibaba">阿里通义千问</option>
              <option value="tencent">腾讯混元大模型</option>
              <option value="bytedance">字节跳动豆包</option>
              <option value="anthropic">Anthropic Claude</option>
              <option value="custom">自定义模型</option>
            </select>
          </div>
          <div class="form-group" v-if="createMethod === 'custom' && createBotForm.provider === 'custom'">
            <label>自定义模型地址</label>
            <input v-model="createBotForm.custom_model_url" type="text" placeholder="请输入模型 API 地址">
          </div>
          <div class="form-group" v-if="createMethod === 'custom' && createBotForm.type === 'custom'">
            <label>龙虾地址</label>
            <input v-model="createBotForm.lobster_url" type="text" placeholder="请输入龙虾地址">
          </div>
          <div class="form-group">
            <label>头像 URL</label>
            <input v-model="createBotForm.avatar" type="text" placeholder="请输入头像 URL（可选）">
          </div>
        </div>
        <div class="modal-footer">
          <button class="cancel-button" @click="showCreateBotModal = false">取消</button>
          <button class="submit-button" @click="createBot" :disabled="creatingBot">
            {{ creatingBot ? '创建中...' : (createMethod === 'template' ? '创建' : '提交审批') }}
          </button>
        </div>
      </div>
    </div>
  </div>
</div>
```

- [ ] **步骤 2：更新 script 部分**

在 script setup 中增加新的响应式变量和函数：

```typescript
// Tab 相关
const activeTab = ref<'available' | 'my-bots' | 'create'>('available')

// 创建方式
const createMethod = ref<'template' | 'custom' | null>(null)
const showCreateForm = ref(false)
const templates = ref<any[]>([])

// 使用 useBots composable
const { fetchBots, fetchTemplates, fetchMyBotCount, createBot: submitCreateBot } = useBots()

// 打开创建模态框
const openCreateModal = async () => {
  showCreateBotModal.value = true
  createMethod.value = null
  showCreateForm.value = false
  // 预加载模板列表
  templates.value = await fetchTemplates()
  // 检查数量限制
  const count = await fetchMyBotCount()
  if (count >= 5) {
    alert('已达到创建上限（5个），如需更多请联系管理员')
    showCreateBotModal.value = false
  }
}

// 选择创建方式
const selectCreateMethod = (method: 'template' | 'custom' | null) => {
  createMethod.value = method
  if (method === 'template') {
    showCreateForm.value = false
  }
}

// 从模板创建
const createFromTemplate = async (tpl: any) => {
  createBotForm.value = {
    name: tpl.name,
    description: tpl.description,
    type: tpl.type,
    provider: 'openai',
    custom_model_url: '',
    lobster_url: '',
    avatar: tpl.avatar,
    config: JSON.parse(tpl.config || '{}'),
    is_template: true,
  }
  showCreateForm.value = true
}

// 创建 Bot（修改现有 createBot 函数）
const createBot = async () => {
  if (!createBotForm.value.name.trim()) {
    alert('请输入机器人名称')
    return
  }

  creatingBot.value = true
  try {
    const isTemplate = createMethod.value === 'template'
    const response = await submitCreateBot({
      ...createBotForm.value,
      is_template: isTemplate,
    })

    if (response.code === 0) {
      await loadBots()
      showCreateBotModal.value = false
      // 重置
      Object.keys(createBotForm.value).forEach(key => {
        createBotForm.value[key as keyof typeof createBotForm.value] = ''
      })
      createBotForm.value.type = 'ai'
      createBotForm.value.provider = 'openai'
      createMethod.value = null
      showCreateForm.value = false
      alert(isTemplate ? '机器人创建成功' : '已提交审批，等待管理员审核')
    }
  } catch (error: any) {
    if (error.response?.data?.code === 'BOT_LIMIT_EXCEEDED') {
      alert('已达到创建上限，请联系管理员')
    } else {
      alert('创建失败，请稍后再试')
    }
  } finally {
    creatingBot.value = false
  }
}

// 组件挂载时加载
onMounted(async () => {
  await loadBots()
})
```

- [ ] **步骤 3：增加 Tab 和模板选择器样式**

在 `<style scoped>` 中添加：

```css
/* Tab 导航 */
.bot-tabs {
  display: flex;
  gap: 4px;
  margin-bottom: 24px;
  border-bottom: 2px solid var(--border-color);
  padding-bottom: 0;
}

.bot-tab {
  padding: 10px 20px;
  border: none;
  background: transparent;
  color: var(--text-secondary);
  cursor: pointer;
  font-size: 14px;
  font-weight: 500;
  border-bottom: 2px solid transparent;
  margin-bottom: -2px;
  transition: all 0.2s;
}

.bot-tab:hover {
  color: var(--text-primary);
}

.bot-tab.active {
  color: var(--primary-color);
  border-bottom-color: var(--primary-color);
}

/* 创建方式选择器 */
.create-method-selector {
  padding: 30px;
  text-align: center;
}

.create-method-selector h3 {
  margin-bottom: 24px;
  color: var(--text-primary);
}

.method-options {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 20px;
  margin-bottom: 24px;
}

.method-option {
  padding: 24px;
  border: 2px solid var(--border-color);
  border-radius: 12px;
  cursor: pointer;
  transition: all 0.2s;
}

.method-option:hover {
  border-color: var(--primary-color);
  transform: translateY(-2px);
}

.method-option.recommended {
  border-color: var(--primary-color);
  background: rgba(var(--primary-rgb, 59, 130, 246), 0.05);
}

.method-icon {
  width: 50px;
  height: 50px;
  border-radius: 50%;
  background: var(--bg-color);
  display: flex;
  align-items: center;
  justify-content: center;
  margin: 0 auto 12px;
  font-size: 22px;
  color: var(--primary-color);
}

.method-option h4 {
  margin: 0 0 8px;
  color: var(--text-primary);
}

.method-option p {
  margin: 0;
  font-size: 13px;
  color: var(--text-secondary);
}

.close-button.full-width {
  width: 100%;
  padding: 10px;
}

/* 模板选择器 */
.template-selector {
  padding: 20px;
}

.template-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.template-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px;
  border: 1px solid var(--border-color);
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s;
}

.template-item:hover {
  background: var(--hover-color);
  border-color: var(--primary-color);
}

.template-avatar {
  width: 44px;
  height: 44px;
  border-radius: 50%;
  background: var(--bg-color);
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
}

.template-avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.template-info h4 {
  margin: 0 0 4px;
  font-size: 15px;
  color: var(--text-primary);
}

.template-info p {
  margin: 0;
  font-size: 13px;
  color: var(--text-secondary);
}

.empty-templates {
  text-align: center;
  padding: 40px;
  color: var(--text-secondary);
}

/* 响应式 */
@media (max-width: 768px) {
  .method-options {
    grid-template-columns: 1fr;
  }
}
```

- [ ] **步骤 4：Commit**

```bash
git add qim-client/src/components/apps/AIAssistantApp.vue
git commit -m "feat(client): AI 助手增加 Tab 导航和模板选择功能"
```

---

## 任务 C3：创建 MyBotsPanel 组件

**文件：**
- 新建：`qim-client/src/components/apps/MyBotsPanel.vue`

**步骤：**

- [ ] **步骤 1：创建 MyBotsPanel.vue**

```vue
<template>
  <div class="my-bots-panel">
    <div v-if="loading" class="loading">
      <i class="fas fa-spinner fa-spin"></i>
      <span>加载中...</span>
    </div>

    <div v-else-if="myBots.length === 0" class="empty-state">
      <i class="fas fa-robot"></i>
      <p>你还没有创建过机器人</p>
      <button class="create-btn" @click="$emit('open-create')">
        <i class="fas fa-plus"></i> 创建第一个机器人
      </button>
    </div>

    <div v-else class="bot-grid">
      <div v-for="bot in myBots" :key="bot.id" class="bot-card" :class="bot.approval_status">
        <div class="bot-header">
          <div class="bot-avatar">
            <img :src="bot.avatar" :alt="bot.name" v-if="bot.avatar">
            <i class="fas fa-robot" v-else></i>
          </div>
          <span class="status-badge" :class="bot.approval_status">
            {{ statusLabel(bot.approval_status) }}
          </span>
        </div>
        <div class="bot-body">
          <h4>{{ bot.name }}</h4>
          <p>{{ bot.description }}</p>
          <p v-if="bot.approval_status === 'rejected' && bot.reject_reason" class="reject-reason">
            拒绝原因：{{ bot.reject_reason }}
          </p>
        </div>
        <div class="bot-actions">
          <button v-if="bot.approval_status === 'approved'" class="action-btn primary" @click="useBot(bot)">
            <i class="fas fa-comment"></i> 使用
          </button>
          <button v-if="['pending', 'rejected'].includes(bot.approval_status)" class="action-btn" @click="$emit('edit-bot', bot)">
            <i class="fas fa-edit"></i> 编辑
          </button>
          <button class="action-btn danger" @click="confirmDelete(bot)">
            <i class="fas fa-trash"></i> 删除
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useBots } from '../../composables/useBots'

const emit = defineEmits(['open-create', 'edit-bot', 'use-bot'])

const { loading, fetchMyBots, deleteBot } = useBots()
const myBots = ref<any[]>([])

const loadMyBots = async () => {
  myBots.value = await fetchMyBots()
}

const statusLabel = (status: string) => {
  const map: Record<string, string> = { approved: '可用', pending: '待审批', rejected: '已拒绝' }
  return map[status] || status
}

const useBot = (bot: any) => {
  emit('use-bot', bot)
}

const confirmDelete = async (bot: any) => {
  if (confirm(`确定删除「${bot.name}」吗？删除后将释放一个创建配额。`)) {
    await deleteBot(bot.id)
    await loadMyBots()
  }
}

onMounted(loadMyBots)
</script>

<style scoped>
.my-bots-panel {
  padding: 10px 0;
}

.loading {
  text-align: center;
  padding: 40px;
  color: var(--text-secondary);
}

.loading i {
  font-size: 24px;
  margin-right: 8px;
}

.empty-state {
  text-align: center;
  padding: 60px 20px;
  color: var(--text-secondary);
}

.empty-state i {
  font-size: 48px;
  margin-bottom: 12px;
  color: var(--text-tertiary);
}

.create-btn {
  margin-top: 20px;
  padding: 10px 24px;
  border: 1px solid var(--primary-color);
  border-radius: 6px;
  background: var(--primary-color);
  color: white;
  cursor: pointer;
  font-size: 14px;
}

.bot-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 16px;
}

.bot-card {
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 12px;
  padding: 16px;
  transition: all 0.2s;
}

.bot-card.pending {
  opacity: 0.6;
}

.bot-card.rejected {
  border-color: #f44336;
}

.bot-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.bot-avatar {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  background: var(--bg-color);
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
}

.bot-avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.status-badge {
  padding: 4px 10px;
  border-radius: 12px;
  font-size: 12px;
  font-weight: 500;
}

.status-badge.approved {
  background: #E8F5E8;
  color: #388E3C;
}

.status-badge.pending {
  background: #FFF8E1;
  color: #FF9800;
}

.status-badge.rejected {
  background: #FFEBEE;
  color: #F44336;
}

.bot-body h4 {
  margin: 0 0 6px;
  font-size: 15px;
  color: var(--text-primary);
}

.bot-body p {
  margin: 0;
  font-size: 13px;
  color: var(--text-secondary);
  line-height: 1.4;
}

.reject-reason {
  margin-top: 8px !important;
  color: #F44336 !important;
  font-size: 12px !important;
}

.bot-actions {
  display: flex;
  gap: 8px;
  margin-top: 16px;
}

.action-btn {
  flex: 1;
  padding: 8px;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  background: var(--card-bg);
  cursor: pointer;
  font-size: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  transition: all 0.2s;
}

.action-btn:hover {
  background: var(--hover-color);
}

.action-btn.primary {
  background: var(--primary-color);
  border-color: var(--primary-color);
  color: white;
}

.action-btn.danger {
  color: #F44336;
  border-color: #F44336;
}

.action-btn.danger:hover {
  background: #FFEBEE;
}

@media (max-width: 768px) {
  .bot-grid {
    grid-template-columns: 1fr;
  }
}
</style>
```

- [ ] **步骤 2：在 AIAssistantApp.vue 中引入 MyBotsPanel 事件处理**

在 AIAssistantApp.vue 的 script 中添加事件处理函数：

```typescript
const handleEditBot = (bot: any) => {
  // 打开编辑表单（复用 createBotForm）
  createBotForm.value = {
    name: bot.name,
    description: bot.description,
    type: bot.type,
    provider: bot.provider || 'openai',
    custom_model_url: '',
    lobster_url: '',
    avatar: bot.avatar,
  }
  showCreateBotModal.value = true
  createMethod.value = 'custom'
  showCreateForm.value = true
}

const handleUseBot = async (bot: any) => {
  // 切换到可用机器人 Tab 并选中该 Bot
  activeTab.value = 'available'
  await selectBot(bot.id)
}
```

在 template 中修改 MyBotsPanel 使用处：

```vue
<MyBotsPanel @use-bot="handleUseBot" @edit-bot="handleEditBot" />
```

- [ ] **步骤 3：Commit**

```bash
git add qim-client/src/components/apps/MyBotsPanel.vue qim-client/src/components/apps/AIAssistantApp.vue
git commit -m "feat(client): 创建我的机器人面板组件"
```

---

# 角色四：🔗 前后端联调与验证

## 任务 I1：端到端验证 Bot 创建与审批流程

**步骤：**

- [ ] **步骤 1：后端编译并启动服务**

```bash
cd qim-server
go build ./...
go run main.go
```

预期：服务启动成功，无编译错误。

- [ ] **步骤 2：验证数据库迁移**

启动后检查数据库，确认：
- `bots` 表新增了 `approval_status`, `creator_id`, `creator_name`, `reject_reason`, `is_template` 列
- `ai_configs` 表新增了 `ai_enabled`, `daily_limit` 列
- `ai_usage_logs` 表已创建

- [ ] **步骤 3：前端启动**

```bash
cd qim-client
npm install
npm run dev
```

- [ ] **步骤 4：验证用户创建 Bot 流程**

1. 登录用户账号
2. 进入 AI 助手页面
3. 点击"创建机器人" → 选择"使用模板" → 确认模板列表展示
4. 选择"自定义机器人" → 填写表单 → 提交
5. 确认 Bot 状态显示为"待审批"

- [ ] **步骤 5：验证管理员审批流程**

1. 登录管理员账号
2. 进入 AI 助手管理 → 切换到"Bot 审批" Tab
3. 确认待审批列表显示刚才创建的 Bot
4. 点击"通过" → 确认状态变为"已通过"
5. 切回用户端 → 确认 Bot 出现在"可用机器人"列表

- [ ] **步骤 6：验证拒绝流程**

1. 再创建一个自定义 Bot
2. 管理员点击"拒绝"，填写拒绝原因
3. 用户端"我的机器人"中显示"已拒绝"标签 + 拒绝原因

- [ ] **步骤 7：验证数量限制**

1. 连续创建 Bot 直到达到上限（5 个）
2. 确认第 6 次创建时提示"已达到创建上限"
3. 删除一个 Bot 后确认可再次创建

- [ ] **步骤 8：Commit**

```bash
git add .
git commit -m "chore: 端到端验证 Bot 创建与审批流程"
```

---

# 规格覆盖度自检

| 规格需求 | 对应任务 |
|----------|----------|
| 管理员审批面板 | A2, B3 |
| Bot 列表审批/通过/拒绝 | B3, A2 |
| 用户创建 Bot 增强（模板/自定义） | C2, B2 |
| 创建数量限制 | B2, C2 |
| 用户"我的 Bot"页面 | C3, B2 |
| 数据库变更（bots/ai_configs/ai_usage_logs） | B1 |
| API 设计（用户侧 + 管理员侧） | B2, B3, B4 |
| 前端组件结构 | A1, A2, C1, C2, C3 |
| 错误处理 | B2, B3, C2 |

**无遗漏需求，无占位符，类型定义一致。**

---

**计划已完成并保存到 `docs/superpowers/plans/2026-04-28-ai-governance-and-approval.md`。两种执行方式：**

**1. 子代理驱动（推荐）** — 每个任务调度一个新的子代理，任务间进行审查，快速迭代

**2. 内联执行** — 在当前会话中使用 executing-plans 执行任务，批量执行并设有检查点

**选哪种方式？**
