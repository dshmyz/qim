package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/di"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/logger"
	"github.com/dshmyz/qim/qim-server/service"
	"github.com/dshmyz/qim/qim-server/ws"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ==========================================
// 用户管理工具
// ==========================================

// UserManagementTool 用户管理工具
type UserManagementTool struct{}

func requireAuthenticatedCaller(ctx *ai.CallerContext) error {
	if ctx == nil || ctx.UserID == 0 {
		return fmt.Errorf("需要登录后才能执行管理工具")
	}
	return nil
}

func (t *UserManagementTool) Name() string {
	return "user_management"
}

func (t *UserManagementTool) Description() string {
	return "用户管理工具，用于管理用户账号：启用/禁用用户"
}

func (t *UserManagementTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"action": map[string]interface{}{
				"type":        "string",
				"description": "操作类型: enable(启用), disable(禁用)",
				"enum":        []string{"enable", "disable"},
			},
			"user_identifier": map[string]interface{}{
				"type":        "string",
				"description": "用户标识：用户名、昵称或用户ID",
			},
		},
		"required": []string{"action", "user_identifier"},
	}
}

func (t *UserManagementTool) Execute(params map[string]interface{}, ctx *ai.CallerContext) (interface{}, error) {
	if err := requireAuthenticatedCaller(ctx); err != nil {
		return nil, err
	}
	// 权限检查：需要是系统管理员
	if !isSystemAdmin(ctx.UserID) {
		return nil, fmt.Errorf("权限不足：只有系统管理员才能管理用户账号")
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
		if err := db.Model(&user).Update("account_status", "active").Error; err != nil {
			return nil, fmt.Errorf("启用用户失败: %w", err)
		}
		return map[string]interface{}{
			"result": "success",
			"action": "enable",
			"detail": fmt.Sprintf("已启用用户 %s", user.Username),
		}, nil

	case "disable":
		if err := db.Model(&user).Update("account_status", "disabled").Error; err != nil {
			return nil, fmt.Errorf("禁用用户失败: %w", err)
		}
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
	return "群组管理工具，用于管理群组：添加/移除成员、禁言/解除禁言、设置/取消管理员、转让群主、修改群公告、查看群成员"
}

func (t *GroupManagementTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"action": map[string]interface{}{
				"type":        "string",
				"description": "操作类型: add_member(添加成员), remove_member(移除成员), mute(禁言), unmute(解除禁言), set_role(设置/取消管理员), transfer_owner(转让群主), update_announcement(修改群公告), list_members(查看群成员)",
				"enum":        []string{"add_member", "remove_member", "mute", "unmute", "set_role", "transfer_owner", "update_announcement", "list_members"},
			},
			"group_identifier": map[string]interface{}{
				"type":        "string",
				"description": "群组标识：群名或群组ID",
			},
			"user_identifier": map[string]interface{}{
				"type":        "string",
				"description": "用户标识：用户名、昵称或用户ID。add_member/remove_member/mute/unmute/set_role/transfer_owner 必填，update_announcement/list_members 不需要",
			},
			"role": map[string]interface{}{
				"type":        "string",
				"description": "目标角色，仅 set_role 必填：admin(设为管理员) 或 member(取消管理员)",
				"enum":        []string{"admin", "member"},
			},
			"announcement": map[string]interface{}{
				"type":        "string",
				"description": "群公告内容，仅 update_announcement 必填",
			},
		},
		"required": []string{"action", "group_identifier"},
	}
}

func (t *GroupManagementTool) Execute(params map[string]interface{}, ctx *ai.CallerContext) (interface{}, error) {
	if err := requireAuthenticatedCaller(ctx); err != nil {
		return nil, err
	}
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

	userIDStr, _ := params["user_identifier"].(string)

	// 查找群组
	var conversation model.Conversation
	err := db.Where("id = ?", groupIDStr).First(&conversation).Error
	if err != nil {
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
	var member model.ConversationMember
	err = db.Where("conversation_id = ? AND user_id = ?", conversation.ID, ctx.UserID).First(&member).Error
	if err != nil {
		return nil, fmt.Errorf("您不是群组成员，无法执行操作")
	}
	if member.Role != "owner" && member.Role != "admin" {
		return nil, fmt.Errorf("权限不足：只有群主或管理员才能执行此操作")
	}

	// 查找要操作的用户（list_members / update_announcement 不需要用户）
	var user model.User
	if action != "list_members" && action != "update_announcement" {
		if userIDStr == "" {
			return nil, fmt.Errorf("user_identifier parameter is required for action: %s", action)
		}
		// 先精确匹配 id/username/nickname。精确命中 → 直接执行（后续 switch）。
		exactFound := db.Where("id = ? OR username = ? OR nickname = ?", userIDStr, userIDStr, userIDStr).First(&user).Error == nil
		if !exactFound {
			// 精确未命中，用昵称 LIKE 找出相似用户作为「候选」返回给 LLM 转述反问，
			// 而不是自动模糊匹配执行——操作类动作（拉人/踢人/禁言）不允许系统替用户猜身份。
			// 候选仅呈递给用户确认，绝不自动拉取。只模糊昵称，不模糊 username/id（易串号）。
			candidates := make([]map[string]interface{}, 0)
			fuzzy := "%" + userIDStr + "%"
			var similar []model.User
			if err := db.Where("nickname LIKE ?", fuzzy).Limit(5).Find(&similar).Error; err == nil {
				for _, u := range similar {
					candidates = append(candidates, map[string]interface{}{
						"id":       u.ID,
						"username": u.Username,
						"nickname": u.Nickname,
					})
				}
			}
			if len(candidates) > 0 {
				// 无法唯一确定目标 → 返回候选名单，要求用户确认后再重发，不直接执行。
				return map[string]interface{}{
					"result":          "need_confirmation",
					"action":          action,
					"reason":          fmt.Sprintf("无法唯一确定用户「%s」，请用户明确指定其中的一个并重发指令", userIDStr),
					"suggest_to_user": "我没有找到完全匹配的成员，但找到了几个相似的昵称。请告诉我要操作的是哪一位（提供准确的用户名或 ID 后再发一次指令）。",
					"candidates":      candidates,
					"candidate_count": len(candidates),
				}, nil
			}
			// 既无精确命中也无相似昵称 → 也是结构化结果交 LLM 转述，引导用户补充完整信息。
			return map[string]interface{}{
				"result":          "not_found",
				"action":          action,
				"reason":          fmt.Sprintf("系统里没有名为「%s」的用户", userIDStr),
				"suggest_to_user": fmt.Sprintf("我没有在系统里找到叫「%s」的成员。请确认对方是否已注册，并提供准确的用户名、完整昵称或用户 ID 后再发一次指令。", userIDStr),
			}, nil
		}
		if action == "remove_member" || action == "mute" || action == "unmute" {
			var targetMember model.ConversationMember
			if err := db.Where("conversation_id = ? AND user_id = ?", conversation.ID, user.ID).First(&targetMember).Error; err != nil {
				return nil, fmt.Errorf("目标用户不是群组成员")
			}
			if targetMember.Role == "owner" {
				return nil, fmt.Errorf("不能移除或禁言群主")
			}
		}
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

		if ws.GlobalHub != nil {
			msg := ws.WSMessage{
				Type: "group_member_joined",
				Data: gin.H{
					"conversation_id": conversation.ID,
					"user_id":         user.ID,
					"member": gin.H{
						"id":       user.ID,
						"nickname": user.Nickname,
						"username": user.Username,
						"avatar":   user.Avatar,
						"type":     user.Type,
					},
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

	case "set_role":
		role, ok := params["role"].(string)
		if !ok || (role != "admin" && role != "member") {
			return nil, fmt.Errorf("role parameter required (admin or member)")
		}
		if user.ID == ctx.UserID {
			return nil, fmt.Errorf("不能修改自己的角色")
		}
		// 禁止修改群主角色：set_role 只允许设为 admin/member，
		// 若目标用户是 owner，admin 不得将其降级（只有 owner 能通过 transfer_owner 转让）
		var targetMember model.ConversationMember
		if err := db.Where("conversation_id = ? AND user_id = ?", conversation.ID, user.ID).First(&targetMember).Error; err != nil {
			return nil, fmt.Errorf("目标用户不是群组成员")
		}
		if targetMember.Role == "owner" {
			return nil, fmt.Errorf("不能修改群主的角色")
		}
		if err := db.Model(&model.ConversationMember{}).
			Where("conversation_id = ? AND user_id = ?", conversation.ID, user.ID).
			Update("role", role).Error; err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"result": "success",
			"action": "set_role",
			"detail": fmt.Sprintf("已将 %s 的角色设为 %s（群组 %s）", user.Username, role, group.Name),
		}, nil

	case "transfer_owner":
		// 转让群主仅群主可操作（权限校验已确保 owner/admin，这里再收紧到 owner）
		var caller model.ConversationMember
		if err := db.Where("conversation_id = ? AND user_id = ?", conversation.ID, ctx.UserID).First(&caller).Error; err != nil {
			return nil, fmt.Errorf("您不是群组成员，无法执行操作")
		}
		if caller.Role != "owner" {
			return nil, fmt.Errorf("权限不足：只有群主才能转让群主")
		}
		if user.ID == ctx.UserID {
			return nil, fmt.Errorf("不能把群主转让给自己")
		}
		// 原群主降为成员，新群主升为 owner，更新 group.CreatorID —— 必须原子完成
		if err := db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Model(&model.ConversationMember{}).
				Where("conversation_id = ? AND user_id = ?", conversation.ID, ctx.UserID).
				Update("role", "member").Error; err != nil {
				return err
			}
			res := tx.Model(&model.ConversationMember{}).
				Where("conversation_id = ? AND user_id = ?", conversation.ID, user.ID).
				Update("role", "owner")
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 0 {
				return fmt.Errorf("目标用户不是群组成员，无法转让")
			}
			return tx.Model(&model.Group{}).Where("conversation_id = ?", conversation.ID).Update("creator_id", user.ID).Error
		}); err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"result": "success",
			"action": "transfer_owner",
			"detail": fmt.Sprintf("已将群主转让给 %s（群组 %s）", user.Username, group.Name),
		}, nil

	case "update_announcement":
		announcement, ok := params["announcement"].(string)
		if !ok {
			return nil, fmt.Errorf("announcement parameter required")
		}
		if err := db.Model(&model.Group{}).
			Where("conversation_id = ?", conversation.ID).
			Update("announcement", announcement).Error; err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"result": "success",
			"action": "update_announcement",
			"detail": fmt.Sprintf("已更新群公告（群组 %s）", group.Name),
		}, nil

	case "list_members":
		var members []model.ConversationMember
		if err := db.Preload("User").Where("conversation_id = ?", conversation.ID).Find(&members).Error; err != nil {
			return nil, err
		}
		list := make([]map[string]interface{}, 0, len(members))
		for _, m := range members {
			name := m.User.Nickname
			if name == "" {
				name = m.User.Username
			}
			list = append(list, map[string]interface{}{
				"user_id": m.UserID,
				"name":    name,
				"role":    m.Role,
			})
		}
		return map[string]interface{}{
			"result":  "success",
			"action":  "list_members",
			"members": list,
			"count":   len(list),
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
		"type": "object",
		"properties": map[string]interface{}{
			"title": map[string]interface{}{
				"type":        "string",
				"description": "通知标题",
			},
			"content": map[string]interface{}{
				"type":        "string",
				"description": "通知内容",
			},
			"target_type": map[string]interface{}{
				"type":        "string",
				"description": "目标类型: user(用户), group(群组), all(全体)",
				"enum":        []string{"user", "group", "all"},
			},
			"target_id": map[string]interface{}{
				"type":        "string",
				"description": "目标ID（全体时可选）",
			},
		},
		"required": []string{"title", "content", "target_type"},
	}
}

func (t *SystemNotificationTool) Execute(params map[string]interface{}, ctx *ai.CallerContext) (interface{}, error) {
	if err := requireAuthenticatedCaller(ctx); err != nil {
		return nil, err
	}
	// 权限检查：需要是系统管理员
	if !isSystemAdmin(ctx.UserID) {
		return nil, fmt.Errorf("权限不足：只有系统管理员才能发送系统通知")
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

	logger.WithModule("SystemNotification").Info("已发送通知", "title", title, "targetType", targetType)

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

// RegisterAdminTools 注册管理操作工具到 AI 工具注册表（ai.ToolRegistry）
// resolveConversationID 把群标识（群组ID 或 群名）解析为 conversation_id。
func resolveConversationID(groupIDStr string) (uint, error) {
	db := database.GetDB()
	var conversation model.Conversation
	if err := db.Where("id = ?", groupIDStr).First(&conversation).Error; err == nil {
		return conversation.ID, nil
	}
	var groupByName model.Group
	if err := db.Where("name = ?", groupIDStr).First(&groupByName).Error; err == nil {
		return groupByName.ConversationID, nil
	}
	return 0, fmt.Errorf("群组不存在: %s", groupIDStr)
}

// requireGroupMember 校验调用者是群组成员且已登录，否则返回 error。
// 用于群待办/搜索/总结等普通成员可用的工具，不要求管理员权限。
func requireGroupMember(convID uint, ctx *ai.CallerContext) error {
	if ctx == nil || ctx.UserID == 0 {
		return fmt.Errorf("需要登录后才能执行群组工具")
	}
	db := database.GetDB()
	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", convID, ctx.UserID).First(&member).Error; err != nil {
		return fmt.Errorf("您不是群组成员，无法执行操作")
	}
	return nil
}

// ==========================================
// 群待办工具
// ==========================================

// CreateGroupTaskTool 群待办工具，在群里创建一条待办任务。
type CreateGroupTaskTool struct{}

func (t *CreateGroupTaskTool) Name() string { return "create_group_task" }
func (t *CreateGroupTaskTool) Description() string {
	return "在【群聊】场景下创建群待办，可指派给群成员，待办归属到群会话（群成员可见）。必需参数 group_identifier。若用户未提到群组或只想建个人待办，请改用 create_user_task。"
}
func (t *CreateGroupTaskTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"group_identifier":    map[string]interface{}{"type": "string", "description": "群组标识：群名或群组ID"},
			"title":               map[string]interface{}{"type": "string", "description": "待办标题"},
			"assignee_identifier": map[string]interface{}{"type": "string", "description": "指派给谁：用户名/昵称/用户ID，可选"},
			"due_date":            map[string]interface{}{"type": "string", "description": "截止日期，可选，格式 2026-07-22"},
			"remind_minutes":      map[string]interface{}{"type": "integer", "description": "提前提醒分钟数，可选，如 30 表示到期前30分钟提醒"},
		},
		"required": []string{"group_identifier", "title"},
	}
}
func (t *CreateGroupTaskTool) Execute(params map[string]interface{}, ctx *ai.CallerContext) (interface{}, error) {
	if err := requireAuthenticatedCaller(ctx); err != nil {
		return nil, err
	}
	db := database.GetDB()
	title, _ := params["title"].(string)
	if title == "" {
		return nil, fmt.Errorf("title parameter required")
	}
	groupIDStr, _ := params["group_identifier"].(string)
	convID, err := resolveConversationID(groupIDStr)
	if err != nil {
		return nil, err
	}
	if err := requireGroupMember(convID, ctx); err != nil {
		return nil, err
	}
	assigneeID := ctx.UserID
	if assigneeStr, _ := params["assignee_identifier"].(string); assigneeStr != "" {
		var u model.User
		if err := db.Where("id = ? OR username = ? OR nickname = ?", assigneeStr, assigneeStr, assigneeStr).First(&u).Error; err == nil {
			assigneeID = u.ID
		}
	}
	task := model.Task{UserID: assigneeID, Title: title, Status: "todo", ConversationID: convID}
	if dueStr, _ := params["due_date"].(string); dueStr != "" {
		// 与 CreateUserTaskTool/TodoExtractor 统一用 time.Parse（UTC）落库，
		// 读出后按绝对时刻比较，不受时区影响。详见 TaskService.ProcessTaskReminders 注释。
		if due, err := time.Parse("2006-01-02", dueStr); err == nil {
			task.DueDate = &due
		}
	}
	if remindMin, ok := params["remind_minutes"].(float64); ok && remindMin > 0 && task.DueDate != nil {
		task.Reminder = int(remindMin)
	}
	if err := db.Create(&task).Error; err != nil {
		return nil, err
	}

	// 通知被指派人
	if assigneeID != ctx.UserID {
		notification := model.Notification{
			UserID:        assigneeID,
			Type:          "todo_assigned",
			Title:         "新的待办事项",
			Content:       title,
			Read:          false,
			Priority:      "important",
			ActionType:    "confirm_reschedule",
			ActionPayload: fmt.Sprintf(`{"task_id":%d}`, task.ID),
		}
		if err := db.Create(&notification).Error; err != nil {
			logger.WithModule("AdminTools").Warn("待办指派通知落库失败",
				"taskID", task.ID, "assigneeID", assigneeID, "error", err)
		}
		if ws.GlobalHub != nil {
			notifMsg, _ := json.Marshal(ws.WSMessage{Type: "new_notification", Data: notification})
			ws.GlobalHub.SendToUser(assigneeID, notifMsg)
		}
	}

	return map[string]interface{}{
		"result":  "success",
		"action":  "create_group_task",
		"detail":  fmt.Sprintf("已创建待办「%s」", title),
		"task_id": task.ID,
	}, nil
}

// ==========================================
// 群消息搜索工具
// ==========================================

// SearchMessagesTool 群消息搜索工具，按关键词搜索群历史消息。
type SearchMessagesTool struct{}

func (t *SearchMessagesTool) Name() string { return "search_messages" }
func (t *SearchMessagesTool) Description() string {
	return "群消息搜索工具，在群里按关键词搜索历史消息"
}
func (t *SearchMessagesTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"group_identifier": map[string]interface{}{"type": "string", "description": "群组标识：群名或群组ID"},
			"keyword":          map[string]interface{}{"type": "string", "description": "搜索关键词"},
			"limit":            map[string]interface{}{"type": "integer", "description": "返回条数上限，默认 10，最大 50"},
		},
		"required": []string{"group_identifier", "keyword"},
	}
}
func (t *SearchMessagesTool) Execute(params map[string]interface{}, ctx *ai.CallerContext) (interface{}, error) {
	if err := requireAuthenticatedCaller(ctx); err != nil {
		return nil, err
	}
	db := database.GetDB()
	keyword, _ := params["keyword"].(string)
	if keyword == "" {
		return nil, fmt.Errorf("keyword parameter required")
	}
	groupIDStr, _ := params["group_identifier"].(string)
	convID, err := resolveConversationID(groupIDStr)
	if err != nil {
		return nil, err
	}
	if err := requireGroupMember(convID, ctx); err != nil {
		return nil, err
	}
	limit := 10
	if l, ok := params["limit"].(float64); ok && l > 0 {
		limit = int(l)
	}
	if limit > 50 {
		limit = 50
	}
	var messages []model.Message
	if err := db.Where("conversation_id = ? AND content LIKE ?", convID, "%"+keyword+"%").
		Order("created_at DESC").Limit(limit).Find(&messages).Error; err != nil {
		return nil, err
	}
	results := make([]map[string]interface{}, 0, len(messages))
	for _, m := range messages {
		results = append(results, map[string]interface{}{
			"message_id": m.ID,
			"sender_id":  m.SenderID,
			"content":    m.Content,
			"created_at": m.CreatedAt,
		})
	}
	return map[string]interface{}{
		"result":   "success",
		"action":   "search_messages",
		"keyword":  keyword,
		"count":    len(results),
		"messages": results,
	}, nil
}

// ==========================================
// 群聊总结工具
// ==========================================

// GroupSummaryTool 群聊总结工具，总结指定时间范围的群聊消息。
type GroupSummaryTool struct{}

func (t *GroupSummaryTool) Name() string { return "group_summary" }
func (t *GroupSummaryTool) Description() string {
	return "群聊总结工具，总结指定时间范围的群聊消息（如\"总结今天聊天\"）"
}
func (t *GroupSummaryTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"group_identifier": map[string]interface{}{"type": "string", "description": "群组标识：群名或群组ID"},
			"time_range":       map[string]interface{}{"type": "string", "description": "时间范围：today(今天) 或 week(本周)，默认 today", "enum": []string{"today", "week"}},
		},
		"required": []string{"group_identifier"},
	}
}
func (t *GroupSummaryTool) Execute(params map[string]interface{}, ctx *ai.CallerContext) (interface{}, error) {
	if err := requireAuthenticatedCaller(ctx); err != nil {
		return nil, err
	}
	groupIDStr, _ := params["group_identifier"].(string)
	convID, err := resolveConversationID(groupIDStr)
	if err != nil {
		return nil, err
	}
	if err := requireGroupMember(convID, ctx); err != nil {
		return nil, err
	}
	timeRange, _ := params["time_range"].(string)
	if timeRange == "" {
		timeRange = "today"
	}
	if di.GlobalContainer == nil || di.GlobalContainer.AIService == nil {
		return nil, fmt.Errorf("AI 服务不可用")
	}
	aiSvc := di.GlobalContainer.AIService
	sg := service.NewSummaryGraph(aiSvc, service.NewAICache())
	if err := sg.Build(); err != nil {
		return nil, fmt.Errorf("构建总结 Graph 失败: %v", err)
	}
	userID := uint(0)
	if ctx != nil {
		userID = ctx.UserID
	}
	out, err := sg.Execute(context.Background(), &service.SummaryInput{
		ConversationID: convID,
		TimeRange:      timeRange,
		UserID:         userID,
	})
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"result":         "success",
		"action":         "group_summary",
		"summary":        out.Summary,
		"messages_count": out.MessagesCount,
		"time_range":     out.TimeRange,
	}, nil
}

func RegisterAdminTools(toolRegistry *ai.ToolRegistry) {
	toolRegistry.RegisterTool(&UserManagementTool{})
	toolRegistry.RegisterTool(&GroupManagementTool{})
	toolRegistry.RegisterTool(&SystemNotificationTool{})
	toolRegistry.RegisterTool(&CreateGroupTaskTool{})
	toolRegistry.RegisterTool(&SearchMessagesTool{})
	toolRegistry.RegisterTool(&GroupSummaryTool{})
	logger.WithModule("AdminTools").Info("已注册管理工具", "tools", "user_management, group_management, system_notification, create_group_task, search_messages, group_summary")
}
