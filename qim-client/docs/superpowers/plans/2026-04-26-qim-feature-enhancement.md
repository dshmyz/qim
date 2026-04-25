# QIM 功能增强实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 为 QIM 即时通讯系统实现 10 个核心功能（5 个短期 + 5 个中期），包括黑名单、群管理员、离线消息、@提及、消息发送失败处理、群设置高级选项、桌面推送通知、文件在线预览、聊天记录导出、笔记/任务共享。

**架构：** 采用功能垂直切片架构，每个功能完成后端（Handler + Service + Model）+ 前端（组件 + Composable + 类型）的完整闭环。按照技术依赖关系分 3 个阶段实现：基础数据层 → 消息交互层 → 高级功能层。

**技术栈：**
- 后端：Go + Gin + GORM + SQLite
- 前端：Vue 3 + TypeScript + Pinia + Element Plus
- 实时通信：WebSocket
- 文件预览：iframe/embed + PDF.js

---

## 文件结构概览

### 后端新增/修改文件
| 文件 | 操作 | 职责 |
|------|------|------|
| `qim-server/model/model.go` | 修改 | 新增 Blacklist、SharedNote、SharedTask 模型，扩展 Conversation 和 Message 模型 |
| `qim-server/handler/blacklist_handler.go` | 创建 | 黑名单 CRUD 处理器 |
| `qim-server/handler/mention_handler.go` | 创建 | @提及解析和通知处理器 |
| `qim-server/handler/group_settings_handler.go` | 创建 | 群设置高级选项处理器 |
| `qim-server/handler/offline_message_handler.go` | 创建 | 离线消息拉取处理器 |
| `qim-server/handler/export_handler.go` | 创建 | 聊天记录导出处理器 |
| `qim-server/handler/share_handler.go` | 创建 | 笔记/任务共享处理器 |
| `qim-server/service/blacklist_service.go` | 创建 | 黑名单业务逻辑 |
| `qim-server/service/mention_service.go` | 创建 | @提及解析和通知业务逻辑 |
| `qim-server/service/offline_message_service.go` | 创建 | 离线消息业务逻辑 |
| `qim-server/service/export_service.go` | 创建 | 聊天记录导出业务逻辑 |
| `qim-server/service/share_service.go` | 创建 | 笔记/任务共享业务逻辑 |
| `qim-server/app/routes.go` | 修改 | 新增功能路由注册 |

### 前端新增/修改文件
| 文件 | 操作 | 职责 |
|------|------|------|
| `qim-client/src/types/index.ts` | 修改 | 新增 BlacklistItem、MentionInfo、GroupSettings 等类型定义 |
| `qim-client/src/composables/useBlacklist.ts` | 创建 | 黑名单管理 composable |
| `qim-client/src/composables/useMention.ts` | 创建 | @提及功能 composable |
| `qim-client/src/composables/useOfflineMessage.ts` | 创建 | 离线消息处理 composable |
| `qim-client/src/composables/useNotification.ts` | 创建 | 桌面推送通知 composable |
| `qim-client/src/composables/useExport.ts` | 创建 | 聊天记录导出 composable |
| `qim-client/src/composables/useShare.ts` | 创建 | 笔记/任务共享 composable |
| `qim-client/src/components/chat/BlacklistManager.vue` | 创建 | 黑名单管理面板组件 |
| `qim-client/src/components/chat/MentionInput.vue` | 创建 | 支持@提及的输入组件 |
| `qim-client/src/components/chat/OfflineMessageIndicator.vue` | 创建 | 离线消息提示组件 |
| `qim-client/src/components/chat/GroupSettings.vue` | 创建 | 群设置面板组件 |
| `qim-client/src/components/chat/MessageRetryButton.vue` | 创建 | 消息重发按钮组件 |
| `qim-client/src/components/shared/FilePreview.vue` | 创建 | 文件在线预览组件 |
| `qim-client/src/components/modals/ExportDialog.vue` | 创建 | 导出对话框组件 |
| `qim-client/src/components/modals/ShareDialog.vue` | 创建 | 分享对话框组件 |
| `qim-client/src/components/chat/MessageInput.vue` | 修改 | 集成@提及功能 |
| `qim-client/src/components/chat/MessageItem.vue` | 修改 | 显示@高亮和发送失败状态 |

---

## 阶段一：基础数据层（任务 1-3）

### 任务 1：黑名单功能

**优先级：** 高（被@提及功能依赖）

**文件：**
- 创建：`qim-server/model/model.go` (修改，新增 Blacklist 模型)
- 创建：`qim-server/handler/blacklist_handler.go`
- 创建：`qim-server/service/blacklist_service.go`
- 创建：`qim-client/src/types/index.ts` (修改，新增 BlacklistItem 类型)
- 创建：`qim-client/src/composables/useBlacklist.ts`
- 创建：`qim-client/src/components/chat/BlacklistManager.vue`
- 修改：`qim-server/app/routes.go` (注册黑名单路由)
- 修改：`qim-server/service/message_service.go` (发送消息时检查黑名单)
- 修改：`qim-client/src/components/chat/MessageItem.vue` (屏蔽黑名单用户消息)

#### 1.1 数据模型定义

在 `qim-server/model/model.go` 末尾添加：

```go
// 黑名单
type Blacklist struct {
	ID        uint           `json:"id" gorm:"primarykey"`
	UserID    uint           `json:"user_id" gorm:"not null;index;uniqueIndex:idx_user_blocked"`
	BlockedID uint           `json:"blocked_id" gorm:"not null;index;uniqueIndex:idx_user_blocked"`
	CreatedAt time.Time      `json:"created_at"`
	User      User           `json:"user,omitempty" gorm:"foreignkey:UserID"`
	Blocked   User           `json:"blocked,omitempty" gorm:"foreignkey:BlockedID"`
}
```

同时在 `Conversation` 模型中添加 `is_muted_all` 字段：

```go
// 在 Conversation 结构体中添加
IsMutedAll bool `json:"is_muted_all" gorm:"default:false"` // 全员禁言
```

#### 1.2 后端服务层

创建 `qim-server/service/blacklist_service.go`：

```go
package service

import (
	"errors"
	"qim-server/database"
	"qim-server/model"
	"gorm.io/gorm"
)

var ErrBlacklistNotFound = errors.New("blacklist item not found")
var ErrAlreadyBlacklisted = errors.New("user already blacklisted")

type BlacklistService struct{}

func NewBlacklistService() *BlacklistService {
	return &BlacklistService{}
}

func (s *BlacklistService) AddToBlacklist(userID, blockedID uint) error {
	db := database.GetDB()

	var existing model.Blacklist
	if err := db.Where("user_id = ? AND blocked_id = ?", userID, blockedID).First(&existing).Error; err == nil {
		return ErrAlreadyBlacklisted
	}

	blacklist := model.Blacklist{
		UserID:    userID,
		BlockedID: blockedID,
	}
	return db.Create(&blacklist).Error
}

func (s *BlacklistService) RemoveFromBlacklist(userID, blockedID uint) error {
	db := database.GetDB()

	result := db.Where("user_id = ? AND blocked_id = ?", userID, blockedID).Delete(&model.Blacklist{})
	if result.RowsAffected == 0 {
		return ErrBlacklistNotFound
	}
	return nil
}

func (s *BlacklistService) GetBlacklist(userID uint) ([]model.Blacklist, error) {
	db := database.GetDB()

	var blacklists []model.Blacklist
	err := db.Where("user_id = ?", userID).Preload("Blocked").Find(&blacklists).Error
	return blacklists, err
}

func (s *BlacklistService) IsBlacklisted(userID, otherID uint) (bool, error) {
	db := database.GetDB()

	var count int64
	err := db.Model(&model.Blacklist{}).
		Where("(user_id = ? AND blocked_id = ?) OR (user_id = ? AND blocked_id = ?)",
			userID, otherID, otherID, userID).Count(&count).Error
	return count > 0, err
}
```

#### 1.3 后端 Handler 层

创建 `qim-server/handler/blacklist_handler.go`：

```go
package handler

import (
	"qim-server/model"
	"qim-server/pkg/response"
	"qim-server/service"
	"strconv"

	"github.com/gin-gonic/gin"
)

var blacklistService = service.NewBlacklistService()

func AddToBlacklist(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req struct {
		BlockedID uint `json:"blocked_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	if req.BlockedID == userID.(uint) {
		response.BadRequest(c, "不能将自己加入黑名单")
		return
	}

	if err := blacklistService.AddToBlacklist(userID.(uint), req.BlockedID); err != nil {
		if err == service.ErrAlreadyBlacklisted {
			response.BadRequest(c, "该用户已在黑名单中")
			return
		}
		response.InternalServerError(c, "加入黑名单失败")
		return
	}

	response.Success(c, gin.H{"message": "已加入黑名单"})
}

func RemoveFromBlacklist(c *gin.Context) {
	userID, _ := c.Get("user_id")
	blockedIDStr := c.Param("id")

	blockedID, err := strconv.ParseUint(blockedIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的用户ID")
		return
	}

	if err := blacklistService.RemoveFromBlacklist(userID.(uint), uint(blockedID)); err != nil {
		if err == service.ErrBlacklistNotFound {
			response.NotFound(c, "黑名单记录不存在")
			return
		}
		response.InternalServerError(c, "移除黑名单失败")
		return
	}

	response.Success(c, gin.H{"message": "已移除黑名单"})
}

func GetBlacklist(c *gin.Context) {
	userID, _ := c.Get("user_id")

	blacklists, err := blacklistService.GetBlacklist(userID.(uint))
	if err != nil {
		response.InternalServerError(c, "获取黑名单失败")
		return
	}

	type BlacklistItem struct {
		ID         uint   `json:"id"`
		BlockedID  uint   `json:"blocked_id"`
		Nickname   string `json:"nickname"`
		Username   string `json:"username"`
		Avatar     string `json:"avatar"`
		CreatedAt string `json:"created_at"`
	}

	var items []BlacklistItem
	for _, bl := range blacklists {
		items = append(items, BlacklistItem{
			ID:        bl.ID,
			BlockedID: bl.BlockedID,
			Nickname:  bl.Blocked.Nickname,
			Username:  bl.Blocked.Username,
			Avatar:    bl.Blocked.Avatar,
			CreatedAt: bl.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	response.Success(c, items)
}

func CheckBlacklist(c *gin.Context) {
	userID, _ := c.Get("user_id")
	otherIDStr := c.Param("id")

	otherID, err := strconv.ParseUint(otherIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的用户ID")
		return
	}

	isBlacklisted, err := blacklistService.IsBlacklisted(userID.(uint), uint(otherID))
	if err != nil {
		response.InternalServerError(c, "检查黑名单失败")
		return
	}

	response.Success(c, gin.H{"is_blacklisted": isBlacklisted})
}

// CheckUserBlacklistMiddleware 检查用户是否被拉黑的中间件
func CheckUserBlacklistMiddleware(c *gin.Context) {
	userID, _ := c.Get("user_id")
	
	// 从请求中获取目标用户ID
	targetIDStr := c.Param("id")
	if targetIDStr == "" {
		targetIDStr = c.Query("target_id")
	}
	
	if targetIDStr == "" {
		c.Next()
		return
	}

	targetID, err := strconv.ParseUint(targetIDStr, 10, 32)
	if err != nil {
		c.Next()
		return
	}

	isBlacklisted, _ := blacklistService.IsBlacklisted(userID.(uint), uint(targetID))
	if isBlacklisted {
		response.Forbidden(c, "对方已将你加入黑名单")
		c.Abort()
		return
	}

	c.Next()
}
```

#### 1.4 路由注册

在 `qim-server/app/routes.go` 的 `authed` 组中添加：

```go
// 黑名单管理
authed.POST("/blacklist", handler.AddToBlacklist)
authed.DELETE("/blacklist/:id", handler.RemoveFromBlacklist)
authed.GET("/blacklist", handler.GetBlacklist)
authed.GET("/blacklist/check/:id", handler.CheckBlacklist)
```

#### 1.5 发送消息时检查黑名单

修改 `qim-server/service/message_service.go` 的 `SendMessage` 方法，在权限检查后添加黑名单检查：

```go
// 在 SendMessage 方法中，成员权限检查后添加：
// 检查发送者是否在接收者的黑名单中
var conv model.Conversation
if err := db.First(&conv, convID).Error; err != nil {
	return nil, err
}

if conv.Type == "single" {
	var otherMember model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id != ?", convID, senderID).First(&otherMember).Error; err == nil {
		isBlacklisted, _ := s.IsBlacklisted(otherMember.UserID, senderID)
		if isBlacklisted {
			return nil, errors.New("对方已将你加入黑名单")
		}
	}
}
```

添加 `IsBlacklisted` 方法到 `MessageService`：

```go
func (s *MessageService) IsBlacklisted(userID, otherID uint) (bool, error) {
	db := database.GetDB()
	var count int64
	err := db.Model(&model.Blacklist{}).
		Where("user_id = ? AND blocked_id = ?", userID, otherID).Count(&count).Error
	return count > 0, err
}
```

#### 1.6 前端类型定义

修改 `qim-client/src/types/index.ts`，添加：

```typescript
export interface BlacklistItem {
  id: string
  blocked_id: string
  nickname: string
  username: string
  avatar: string
  created_at: string
}

export interface Conversation {
  // 在现有 Conversation 接口中添加
  is_muted_all?: boolean
  all_muted_except_admins?: boolean
  join_approval_required?: boolean
  only_admin_can_modify_info?: boolean
}
```

#### 1.7 前端 Composable

创建 `qim-client/src/composables/useBlacklist.ts`：

```typescript
import { ref } from 'vue'
import type { BlacklistItem } from '../types'

export function useBlacklist(request: (url: string, options?: any) => Promise<any>) {
  const blacklist = ref<BlacklistItem[]>([])
  const isLoading = ref(false)

  const loadBlacklist = async () => {
    isLoading.value = true
    try {
      const response = await request('/api/v1/blacklist')
      if (response.code === 0) {
        blacklist.value = response.data || []
      }
    } catch (error) {
      console.error('加载黑名单失败:', error)
    } finally {
      isLoading.value = false
    }
  }

  const addToBlacklist = async (blockedId: string) => {
    try {
      const response = await request('/api/v1/blacklist', {
        method: 'POST',
        body: JSON.stringify({ blocked_id: parseInt(blockedId) })
      })
      if (response.code === 0) {
        await loadBlacklist()
        return { success: true }
      }
      return { success: false, message: response.message }
    } catch (error) {
      return { success: false, message: '操作失败' }
    }
  }

  const removeFromBlacklist = async (blockedId: string) => {
    try {
      const response = await request(`/api/v1/blacklist/${blockedId}`, {
        method: 'DELETE'
      })
      if (response.code === 0) {
        await loadBlacklist()
        return { success: true }
      }
      return { success: false, message: response.message }
    } catch (error) {
      return { success: false, message: '操作失败' }
    }
  }

  return {
    blacklist,
    isLoading,
    loadBlacklist,
    addToBlacklist,
    removeFromBlacklist
  }
}
```

#### 1.8 前端组件

创建 `qim-client/src/components/chat/BlacklistManager.vue`（黑名单管理面板）

**注意：** 由于组件代码较长，将在实现时按功能逐步编写。

- [ ] **步骤 1：添加 Blacklist 模型到 model.go**

在 `qim-server/model/model.go` 末尾添加 Blacklist 模型定义，同时在 Conversation 中添加 `is_muted_all`、`all_muted_except_admins`、`join_approval_required`、`only_admin_can_modify_info` 字段。

- [ ] **步骤 2：创建 blacklist_service.go**

创建 `qim-server/service/blacklist_service.go` 文件，包含 AddToBlacklist、RemoveFromBlacklist、GetBlacklist、IsBlacklisted 四个方法。

- [ ] **步骤 3：创建 blacklist_handler.go**

创建 `qim-server/handler/blacklist_handler.go` 文件，包含 AddToBlacklist、RemoveFromBlacklist、GetBlacklist、CheckBlacklist、CheckUserBlacklistMiddleware 五个处理器。

- [ ] **步骤 4：注册路由**

在 `qim-server/app/routes.go` 中添加黑名单相关路由。

- [ ] **步骤 5：修改消息发送逻辑**

在 `qim-server/service/message_service.go` 的 SendMessage 方法中添加黑名单检查逻辑。

- [ ] **步骤 6：添加前端类型定义**

在 `qim-client/src/types/index.ts` 中添加 BlacklistItem 接口。

- [ ] **步骤 7：创建 useBlacklist composable**

创建 `qim-client/src/composables/useBlacklist.ts` 文件。

- [ ] **步骤 8：创建 BlacklistManager 组件**

创建 `qim-client/src/components/chat/BlacklistManager.vue` 组件，提供黑名单列表展示、添加/移除功能。

- [ ] **步骤 9：测试黑名单功能**

运行后端服务，使用 Postman 或 curl 测试：
```bash
# 添加黑名单
curl -X POST http://localhost:8080/api/v1/blacklist \
  -H "Authorization: Bearer {token}" \
  -H "Content-Type: application/json" \
  -d '{"blocked_id": 2}'

# 获取黑名单
curl -X GET http://localhost:8080/api/v1/blacklist \
  -H "Authorization: Bearer {token}"

# 移除黑名单
curl -X DELETE http://localhost:8080/api/v1/blacklist/2 \
  -H "Authorization: Bearer {token}"
```

- [ ] **步骤 10：Commit**

```bash
git add qim-server/model/model.go
git add qim-server/handler/blacklist_handler.go
git add qim-server/service/blacklist_service.go
git add qim-server/service/message_service.go
git add qim-server/app/routes.go
git add qim-client/src/types/index.ts
git add qim-client/src/composables/useBlacklist.ts
git add qim-client/src/components/chat/BlacklistManager.vue
git commit -m "feat: 添加黑名单功能，支持用户拉黑/解除拉黑"
```

---

### 任务 2：群管理员功能增强

**优先级：** 高（被群设置高级选项依赖）

**说明：** 现有代码已有基础的群管理员功能（SetMemberRole、RemoveMemberFromGroup），但需要增强：
1. 禁言指定成员
2. 批量移除成员
3. 群主转让时的权限转移

**文件：**
- 修改：`qim-server/model/model.go` (ConversationMember 添加 muted_until 字段)
- 创建：`qim-server/handler/group_mute_handler.go`
- 创建：`qim-server/service/group_mute_service.go`
- 修改：`qim-server/handler/group_handler.go` (增强批量操作)
- 创建：`qim-client/src/composables/useGroupMute.ts`
- 创建：`qim-client/src/components/chat/MemberMuteControl.vue`
- 修改：`qim-server/app/routes.go`

#### 2.1 数据模型扩展

在 `qim-server/model/model.go` 的 `ConversationMember` 中添加：

```go
// 在 ConversationMember 结构体中添加
MutedUntil *time.Time `json:"muted_until"` // 禁言截止时间，nil 表示不禁言
```

#### 2.2 群禁言服务

创建 `qim-server/service/group_mute_service.go`：

```go
package service

import (
	"errors"
	"time"
	"qim-server/database"
	"qim-server/model"
)

type GroupMuteService struct{}

func NewGroupMuteService() *GroupMuteService {
	return &GroupMuteService{}
}

// MuteMember 禁言群成员
func (s *GroupMuteService) MuteMember(convID, operatorID, targetID uint, durationMinutes int) error {
	db := database.GetDB()

	var conv model.Conversation
	if err := db.First(&conv, convID).Error; err != nil {
		return errors.New("群聊不存在")
	}

	var operator model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", convID, operatorID).First(&operator).Error; err != nil {
		return errors.New("您不是群成员")
	}

	if operator.Role != "owner" && operator.Role != "admin" {
		return errors.New("只有群主或管理员可以禁言成员")
	}

	var target model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", convID, targetID).First(&target).Error; err != nil {
		return errors.New("目标用户不是群成员")
	}

	if target.Role == "owner" {
		return errors.New("不能禁言群主")
	}

	if operator.Role == "admin" && target.Role == "admin" {
		return errors.New("管理员不能禁言其他管理员")
	}

	now := time.Now()
	mutedUntil := now.Add(time.Duration(durationMinutes) * time.Minute)
	target.MutedUntil = &mutedUntil
	return db.Save(&target).Error
}

// UnmuteMember 取消禁言
func (s *GroupMuteService) UnmuteMember(convID, operatorID, targetID uint) error {
	db := database.GetDB()

	var operator model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", convID, operatorID).First(&operator).Error; err != nil {
		return errors.New("您不是群成员")
	}

	if operator.Role != "owner" && operator.Role != "admin" {
		return errors.New("只有群主或管理员可以取消禁言")
	}

	var target model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", convID, targetID).First(&target).Error; err != nil {
		return errors.New("目标用户不是群成员")
	}

	target.MutedUntil = nil
	return db.Save(&target).Error
}

// IsMuted 检查用户是否被禁言
func (s *GroupMuteService) IsMuted(convID, userID uint) (bool, error) {
	db := database.GetDB()

	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", convID, userID).First(&member).Error; err != nil {
		return false, err
	}

	if member.MutedUntil == nil {
		return false, nil
	}

	if time.Now().After(*member.MutedUntil) {
		// 禁言已过期，自动取消
		member.MutedUntil = nil
		db.Save(&member)
		return false, nil
	}

	return true, nil
}

// BatchRemoveMembers 批量移除成员
func (s *GroupMuteService) BatchRemoveMembers(convID, operatorID uint, targetIDs []uint) ([]uint, []string, error) {
	db := database.GetDB()

	var operator model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", convID, operatorID).First(&operator).Error; err != nil {
		return nil, nil, errors.New("您不是群成员")
	}

	if operator.Role != "owner" && operator.Role != "admin" {
		return nil, nil, errors.New("只有群主或管理员可以移除成员")
	}

	var successIDs []uint
	var failedReasons []string

	for _, targetID := range targetIDs {
		var target model.ConversationMember
		if err := db.Where("conversation_id = ? AND user_id = ?", convID, targetID).First(&target).Error; err != nil {
			failedReasons = append(failedReasons, "用户不是群成员")
			continue
		}

		if target.Role == "owner" {
			failedReasons = append(failedReasons, "群主不能被移除")
			continue
		}

		if operator.Role == "admin" && target.Role == "admin" {
			failedReasons = append(failedReasons, "管理员不能移除其他管理员")
			continue
		}

		if err := db.Delete(&target).Error; err != nil {
			failedReasons = append(failedReasons, "移除失败")
			continue
		}

		successIDs = append(successIDs, targetID)
	}

	return successIDs, failedReasons, nil
}
```

#### 2.3 群禁言 Handler

创建 `qim-server/handler/group_mute_handler.go`：

```go
package handler

import (
	"qim-server/pkg/response"
	"strconv"

	"github.com/gin-gonic/gin"
)

var groupMuteService = NewGroupMuteService()

func MuteGroupMember(c *gin.Context) {
	userID, _ := c.Get("user_id")
	convIDStr := c.Param("id")
	targetIDStr := c.Param("user_id")

	convID, _ := strconv.ParseUint(convIDStr, 10, 32)
	targetID, _ := strconv.ParseUint(targetIDStr, 10, 32)

	var req struct {
		DurationMinutes int `json:"duration_minutes" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	if err := groupMuteService.MuteMember(uint(convID), userID.(uint), uint(targetID), req.DurationMinutes); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "禁言成功"})
}

func UnmuteGroupMember(c *gin.Context) {
	userID, _ := c.Get("user_id")
	convIDStr := c.Param("id")
	targetIDStr := c.Param("user_id")

	convID, _ := strconv.ParseUint(convIDStr, 10, 32)
	targetID, _ := strconv.ParseUint(targetIDStr, 10, 32)

	if err := groupMuteService.UnmuteMember(uint(convID), userID.(uint), uint(targetID)); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "取消禁言成功"})
}

func BatchRemoveGroupMembers(c *gin.Context) {
	userID, _ := c.Get("user_id")
	convIDStr := c.Param("id")

	convID, _ := strconv.ParseUint(convIDStr, 10, 32)

	var req struct {
		MemberIDs []uint `json:"member_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	successIDs, failedReasons, err := groupMuteService.BatchRemoveMembers(uint(convID), userID.(uint), req.MemberIDs)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, gin.H{
		"success_ids":  successIDs,
		"failed_reasons": failedReasons,
		"message": "批量移除完成",
	})
}
```

#### 2.4 路由注册

在 `qim-server/app/routes.go` 中添加：

```go
// 群禁言管理
authed.POST("/conversations/:id/members/:user_id/mute", handler.MuteGroupMember)
authed.DELETE("/conversations/:id/members/:user_id/mute", handler.UnmuteGroupMember)
authed.POST("/conversations/:id/members/batch-remove", handler.BatchRemoveGroupMembers)
```

#### 2.5 发送消息时检查禁言

修改 `qim-server/service/message_service.go` 的 `SendMessage` 方法，在黑名单检查后添加：

```go
// 检查发送者是否被禁言（群聊场景）
if conv.Type == "group" {
	isMuted, _ := s.IsGroupMuted(convID, senderID)
	if isMuted {
		return nil, errors.New("你已被禁言，无法发送消息")
	}

	// 检查全员禁言
	if conv.IsMutedAll && member.Role == "member" {
		return nil, errors.New("该群已开启全员禁言")
	}
}
```

添加 `IsGroupMuted` 方法：

```go
func (s *MessageService) IsGroupMuted(convID, userID uint) (bool, error) {
	db := database.GetDB()

	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", convID, userID).First(&member).Error; err != nil {
		return false, err
	}

	if member.MutedUntil == nil {
		return false, nil
	}

	return time.Now().Before(*member.MutedUntil), nil
}
```

#### 2.6 前端 Composable

创建 `qim-client/src/composables/useGroupMute.ts`：

```typescript
import { ref } from 'vue'

export function useGroupMute(request: (url: string, options?: any) => Promise<any>) {
  const mutingMember = ref<string | null>(null)

  const muteMember = async (convId: string, memberId: string, durationMinutes: number) => {
    mutingMember.value = memberId
    try {
      const response = await request(`/api/v1/conversations/${convId}/members/${memberId}/mute`, {
        method: 'POST',
        body: JSON.stringify({ duration_minutes: durationMinutes })
      })
      if (response.code === 0) {
        return { success: true }
      }
      return { success: false, message: response.message }
    } catch (error) {
      return { success: false, message: '操作失败' }
    } finally {
      mutingMember.value = null
    }
  }

  const unmuteMember = async (convId: string, memberId: string) => {
    try {
      const response = await request(`/api/v1/conversations/${convId}/members/${memberId}/mute`, {
        method: 'DELETE'
      })
      if (response.code === 0) {
        return { success: true }
      }
      return { success: false, message: response.message }
    } catch (error) {
      return { success: false, message: '操作失败' }
    }
  }

  const batchRemoveMembers = async (convId: string, memberIds: string[]) => {
    try {
      const response = await request(`/api/v1/conversations/${convId}/members/batch-remove`, {
        method: 'POST',
        body: JSON.stringify({ member_ids: memberIds.map(id => parseInt(id)) })
      })
      if (response.code === 0) {
        return { 
          success: true, 
          data: response.data 
        }
      }
      return { success: false, message: response.message }
    } catch (error) {
      return { success: false, message: '操作失败' }
    }
  }

  return {
    mutingMember,
    muteMember,
    unmuteMember,
    batchRemoveMembers
  }
}
```

- [ ] **步骤 1：扩展 ConversationMember 模型**

在 `qim-server/model/model.go` 的 ConversationMember 结构体中添加 `MutedUntil` 字段。

- [ ] **步骤 2：扩展 Conversation 模型**

在 `qim-server/model/model.go` 的 Conversation 结构体中添加 `IsMutedAll`、`AllMutedExceptAdmins`、`JoinApprovalRequired`、`OnlyAdminCanModifyInfo` 字段。

- [ ] **步骤 3：创建 group_mute_service.go**

创建 `qim-server/service/group_mute_service.go`，包含 MuteMember、UnmuteMember、IsMuted、BatchRemoveMembers 方法。

- [ ] **步骤 4：创建 group_mute_handler.go**

创建 `qim-server/handler/group_mute_handler.go`，包含 MuteGroupMember、UnmuteGroupMember、BatchRemoveGroupMembers 处理器。

- [ ] **步骤 5：注册路由**

在 `qim-server/app/routes.go` 中添加群禁言和批量移除路由。

- [ ] **步骤 6：修改消息发送逻辑**

在 `qim-server/service/message_service.go` 的 SendMessage 方法中添加禁言检查逻辑。

- [ ] **步骤 7：创建 useGroupMute composable**

创建 `qim-client/src/composables/useGroupMute.ts`。

- [ ] **步骤 8：创建 MemberMuteControl 组件**

创建 `qim-client/src/components/chat/MemberMuteControl.vue` 组件。

- [ ] **步骤 9：测试群管理员功能**

使用 curl 测试禁言和批量移除功能。

- [ ] **步骤 10：Commit**

```bash
git add qim-server/model/model.go
git add qim-server/service/group_mute_service.go
git add qim-server/handler/group_mute_handler.go
git add qim-server/service/message_service.go
git add qim-server/app/routes.go
git add qim-client/src/composables/useGroupMute.ts
git add qim-client/src/components/chat/MemberMuteControl.vue
git commit -m "feat: 增强群管理员功能，支持禁言、批量移除成员"
```

---

### 任务 3：离线消息拉取

**优先级：** 高（消息系统基础，被消息发送失败处理依赖）

**说明：** 用户重新上线时，拉取未收到的消息。

**文件：**
- 创建：`qim-server/handler/offline_message_handler.go`
- 创建：`qim-server/service/offline_message_service.go`
- 修改：`qim-server/ws/hub.go` (用户上线时推送离线消息)
- 修改：`qim-client/src/composables/useOfflineMessage.ts` (创建)
- 修改：`qim-client/src/composables/useWebSocket.ts` (连接成功后拉取离线消息)
- 修改：`qim-server/app/routes.go`

#### 3.1 离线消息服务

创建 `qim-server/service/offline_message_service.go`：

```go
package service

import (
	"time"
	"qim-server/database"
	"qim-server/model"
)

type OfflineMessageService struct{}

func NewOfflineMessageService() *OfflineMessageService {
	return &OfflineMessageService{}
}

// GetOfflineMessages 获取用户的离线消息
func (s *OfflineMessageService) GetOfflineMessages(userID uint) (map[uint][]model.Message, error) {
	db := database.GetDB()

	// 获取用户的所有会话
	var members []model.ConversationMember
	if err := db.Where("user_id = ?", userID).Find(&members).Error; err != nil {
		return nil, err
	}

	offlineMessages := make(map[uint][]model.Message)

	for _, member := range members {
		// 获取上次阅读时间
		lastReadAt := member.LastReadAt
		if lastReadAt == nil {
			// 如果没有阅读记录，获取加入时间
			lastReadAt = &member.JoinedAt
		}

		// 获取该会话中上次阅读后的消息
		var messages []model.Message
		if err := db.Where(
			"conversation_id = ? AND sender_id != ? AND created_at > ? AND is_recalled = false",
			member.ConversationID, userID, *lastReadAt,
		).Preload("Sender").Order("created_at ASC").Find(&messages).Error; err != nil {
			continue
		}

		if len(messages) > 0 {
			offlineMessages[member.ConversationID] = messages
		}
	}

	return offlineMessages, nil
}

// UpdateLastReadAt 更新用户会话的最后阅读时间
func (s *OfflineMessageService) UpdateLastReadAt(convID, userID uint) error {
	db := database.GetDB()

	now := time.Now()
	return db.Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", convID, userID).
		UpdateColumn("last_read_at", now).Error
}
```

#### 3.2 离线消息 Handler

创建 `qim-server/handler/offline_message_handler.go`：

```go
package handler

import (
	"qim-server/pkg/response"
	"qim-server/service"