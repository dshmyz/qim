package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"qim-server/ai"
	"qim-server/database"
	"qim-server/model"
	"qim-server/ws"

	"github.com/gin-gonic/gin"
)

// ==========================================
// 用户管理工具
// ==========================================

// UserManagementTool 用户管理工具
type UserManagementTool struct{}

func (t *UserManagementTool) Name() string {
	return "user_management"
}

func (t *UserManagementTool) Description() string {
	return "用户管理工具，用于管理用户账号：启用/禁用用户"
}

func (t *UserManagementTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"action": map[string]interface{}{
			"type":        "string",
			"description": "操作类型: enable(启用), disable(禁用)",
			"enum":        []string{"enable", "disable"},
			"required":    true,
		},
		"user_identifier": map[string]interface{}{
			"type":        "string",
			"description": "用户标识：用户名、昵称或用户ID",
			"required":    true,
		},
	}
}

func (t *UserManagementTool) Execute(params map[string]interface{}) (interface{}, error) {
	return t.ExecuteWithAuth(params, nil)
}

// ExecuteWithAuth 带权限检查的执行
func (t *UserManagementTool) ExecuteWithAuth(params map[string]interface{}, ctx *ai.CallerContext) (interface{}, error) {
	// 权限检查：需要是系统管理员
	if ctx != nil && ctx.UserID > 0 {
		if !isSystemAdmin(ctx.UserID) {
			return nil, fmt.Errorf("权限不足：只有系统管理员才能管理用户账号")
		}
	}

	action, ok := params["action"].(string)
	if !ok {
		return nil, fmt.Errorf("action parameter is required")
	}

	identifier, ok := params["user_identifier"].(string)
	if !ok {
		return nil, fmt.Errorf("user_identifier parameter is required")
	}

	db := database.GetDB()

	var user model.User
	err := db.Where("id = ? OR username = ? OR nickname = ?", identifier, identifier, identifier).First(&user).Error
	if err != nil {
		return nil, fmt.Errorf("用户不存在: %s", identifier)
	}

	switch action {
	case "enable":
		db.Model(&user).Update("status", "active")
		return map[string]interface{}{
			"result": "success",
			"action": "enable",
			"detail": fmt.Sprintf("已启用用户 %s", user.Username),
		}, nil

	case "disable":
		db.Model(&user).Update("status", "disabled")
		return map[string]interface{}{
			"result": "success",
			"action": "disable",
			"detail": fmt.Sprintf("已禁用用户 %s", user.Username),
		}, nil

	default:
		return nil, fmt.Errorf("未知操作: %s", action)
	}
}

// ==========================================
// 群组管理工具
// ==========================================

// GroupManagementTool 群组管理工具
type GroupManagementTool struct{}

func (t *GroupManagementTool) Name() string {
	return "group_management"
}

func (t *GroupManagementTool) Description() string {
	return "群组管理工具，用于管理群组：添加/移除成员、禁言/解除禁言"
}

func (t *GroupManagementTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"action": map[string]interface{}{
			"type":        "string",
			"description": "操作类型: add_member(添加成员), remove_member(移除成员), mute(禁言), unmute(解除禁言)",
			"enum":        []string{"add_member", "remove_member", "mute", "unmute"},
			"required":    true,
		},
		"group_identifier": map[string]interface{}{
			"type":        "string",
			"description": "群组标识：群名或群组ID",
			"required":    true,
		},
		"user_identifier": map[string]interface{}{
			"type":        "string",
			"description": "用户标识：用户名、昵称或用户ID",
			"required":    true,
		},
	}
}

func (t *GroupManagementTool) Execute(params map[string]interface{}) (interface{}, error) {
	return t.ExecuteWithAuth(params, nil)
}

// ExecuteWithAuth 带权限检查的执行
func (t *GroupManagementTool) ExecuteWithAuth(params map[string]interface{}, ctx *ai.CallerContext) (interface{}, error) {
	db := database.GetDB()

	// 解析参数
	action, ok := params["action"].(string)
	if !ok {
		return nil, fmt.Errorf("action parameter is required")
	}

	groupIDStr, ok := params["group_identifier"].(string)
	if !ok {
		return nil, fmt.Errorf("group_identifier parameter is required")
	}

	userIDStr, ok := params["user_identifier"].(string)
	if !ok {
		return nil, fmt.Errorf("user_identifier parameter is required")
	}

	// 查找群组
	var conversation model.Conversation
	err := db.Where("id = ?", groupIDStr).First(&conversation).Error
	if err != nil {
		// 尝试通过名称查找群组
		var groupByName model.Group
		if err := db.Where("name = ?", groupIDStr).First(&groupByName).Error; err == nil {
			conversation.ID = groupByName.ConversationID
		} else {
			return nil, fmt.Errorf("群组不存在: %s", groupIDStr)
		}
	}

	// 获取群聊详细信息
	var group model.Group
	if err := db.Where("conversation_id = ?", conversation.ID).First(&group).Error; err != nil {
		return nil, fmt.Errorf("获取群聊信息失败: %v", err)
	}

	// 权限检查：检查调用者是否是群主或管理员
	if ctx != nil && ctx.UserID > 0 {
		var member model.ConversationMember
		err := db.Where("conversation_id = ? AND user_id = ?", conversation.ID, ctx.UserID).First(&member).Error
		if err != nil {
			return nil, fmt.Errorf("您不是群组成员，无法执行操作")
		}
		if member.Role != "owner" && member.Role != "admin" {
			return nil, fmt.Errorf("权限不足：只有群主或管理员才能执行此操作")
		}
	} else if ctx == nil || ctx.UserID == 0 {
		// 没有上下文时，允许执行（兼容旧接口）
		log.Printf("[GroupManagementTool] 警告：无调用者上下文，跳过权限检查")
	}

	// 查找要操作的用户
	var user model.User
	err = db.Where("id = ? OR username = ? OR nickname = ?", userIDStr, userIDStr, userIDStr).First(&user).Error
	if err != nil {
		return nil, fmt.Errorf("用户不存在: %s", userIDStr)
	}

	// 执行操作
	switch action {
	case "add_member":
		member := model.ConversationMember{
			ConversationID: conversation.ID,
			UserID:         user.ID,
			JoinedAt:       time.Now(),
		}
		db.FirstOrCreate(&member, model.ConversationMember{ConversationID: conversation.ID, UserID: user.ID})

		// 发送 WebSocket 通知（与 API 保持一致）
		if ws.GlobalHub != nil {
			msg := ws.WSMessage{
				Type: "group_member_joined",
				Data: gin.H{
					"conversation_id": conversation.ID,
					"user_id":         user.ID,
				},
			}
			jsonMsg, _ := json.Marshal(msg)
			ws.GlobalHub.SendToConversation(conversation.ID, 0, jsonMsg)
		}

		return map[string]interface{}{
			"result": "success",
			"action": "add_member",
			"detail": fmt.Sprintf("已将 %s 添加到群组 %s", user.Username, group.Name),
		}, nil

	case "remove_member":
		db.Where("conversation_id = ? AND user_id = ?", conversation.ID, user.ID).Delete(&model.ConversationMember{})

		// 发送 WebSocket 通知（与 API 保持一致）
		if ws.GlobalHub != nil {
			msg := ws.WSMessage{
				Type: "group_member_left",
				Data: gin.H{
					"conversation_id": conversation.ID,
					"user_id":         user.ID,
				},
			}
			jsonMsg, _ := json.Marshal(msg)
			ws.GlobalHub.SendToConversation(conversation.ID, 0, jsonMsg)
		}

		return map[string]interface{}{
			"result": "success",
			"action": "remove_member",
			"detail": fmt.Sprintf("已将 %s 从群组 %s 移除", user.Username, group.Name),
		}, nil

	case "mute":
		db.Model(&model.ConversationMember{}).
			Where("conversation_id = ? AND user_id = ?", conversation.ID, user.ID).
			Update("muted_until", time.Now().Add(24*time.Hour))

		// 发送 WebSocket 通知
		if ws.GlobalHub != nil {
			msg := ws.WSMessage{
				Type: "group_member_muted",
				Data: gin.H{
					"conversation_id": conversation.ID,
					"user_id":         user.ID,
				},
			}
			jsonMsg, _ := json.Marshal(msg)
			ws.GlobalHub.SendToConversation(conversation.ID, 0, jsonMsg)
		}

		return map[string]interface{}{
			"result": "success",
			"action": "mute",
			"detail": fmt.Sprintf("已禁言 %s（群组 %s），时长 24 小时", user.Username, group.Name),
		}, nil

	case "unmute":
		db.Model(&model.ConversationMember{}).
			Where("conversation_id = ? AND user_id = ?", conversation.ID, user.ID).
			Update("muted_until", nil)

		// 发送 WebSocket 通知
		if ws.GlobalHub != nil {
			msg := ws.WSMessage{
				Type: "group_member_unmuted",
				Data: gin.H{
					"conversation_id": conversation.ID,
					"user_id":         user.ID,
				},
			}
			jsonMsg, _ := json.Marshal(msg)
			ws.GlobalHub.SendToConversation(conversation.ID, 0, jsonMsg)
		}

		return map[string]interface{}{
			"result": "success",
			"action": "unmute",
			"detail": fmt.Sprintf("已解除 %s 的禁言（群组 %s）", user.Username, group.Name),
		}, nil

	default:
		return nil, fmt.Errorf("未知操作: %s", action)
	}
}

// ==========================================
// 系统通知工具
// ==========================================

// SystemNotificationTool 系统通知工具
type SystemNotificationTool struct{}

func (t *SystemNotificationTool) Name() string {
	return "system_notification"
}

func (t *SystemNotificationTool) Description() string {
	return "系统通知工具，用于向用户或群组发送系统通知"
}

func (t *SystemNotificationTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"title": map[string]interface{}{
			"type":        "string",
			"description": "通知标题",
			"required":    true,
		},
		"content": map[string]interface{}{
			"type":        "string",
			"description": "通知内容",
			"required":    true,
		},
		"target_type": map[string]interface{}{
			"type":        "string",
			"description": "目标类型: user(用户), group(群组), all(全体)",
			"enum":        []string{"user", "group", "all"},
			"required":    true,
		},
		"target_id": map[string]interface{}{
			"type":        "string",
			"description": "目标ID（全体时可选）",
			"required":    false,
		},
	}
}

func (t *SystemNotificationTool) Execute(params map[string]interface{}) (interface{}, error) {
	return t.ExecuteWithAuth(params, nil)
}

// ExecuteWithAuth 带权限检查的执行
func (t *SystemNotificationTool) ExecuteWithAuth(params map[string]interface{}, ctx *ai.CallerContext) (interface{}, error) {
	// 权限检查：需要是系统管理员
	if ctx != nil && ctx.UserID > 0 {
		if !isSystemAdmin(ctx.UserID) {
			return nil, fmt.Errorf("权限不足：只有系统管理员才能发送系统通知")
		}
	}

	title, ok := params["title"].(string)
	if !ok {
		return nil, fmt.Errorf("title parameter is required")
	}

	content, ok := params["content"].(string)
	if !ok {
		return nil, fmt.Errorf("content parameter is required")
	}

	targetType, ok := params["target_type"].(string)
	if !ok {
		return nil, fmt.Errorf("target_type parameter is required")
	}

	db := database.GetDB()

	sysMsg := model.SystemMessage{
		Title:      title,
		Content:    content,
		SenderID:   0,
		Status:     "active",
		TargetType: targetType,
		CreatedAt:  time.Now(),
	}

	if targetIDStr, ok := params["target_id"].(string); ok && targetIDStr != "" {
		var targetID uint
		fmt.Sscanf(targetIDStr, "%d", &targetID)
		sysMsg.TargetID = &targetID
	}

	db.Create(&sysMsg)

	log.Printf("[SystemNotification] 已发送通知: %s -> %s", title, targetType)

	return map[string]interface{}{
		"result": "success",
		"detail": fmt.Sprintf("已发送系统通知: %s", title),
	}, nil
}

// isSystemAdmin 检查用户是否是系统管理员
func isSystemAdmin(userID uint) bool {
	db := database.GetDB()
	var count int64
	db.Model(&model.UserRole{}).Where("user_id = ? AND role = ?", userID, "system_admin").Count(&count)
	return count > 0
}

// RegisterAdminTools 注册管理操作工具到 MCP 服务器
func RegisterAdminTools(mcpServer *ai.MCPServer) {
	mcpServer.RegisterTool(&UserManagementTool{})
	mcpServer.RegisterTool(&GroupManagementTool{})
	mcpServer.RegisterTool(&SystemNotificationTool{})
	log.Printf("已注册管理工具: user_management, group_management, system_notification")
}
