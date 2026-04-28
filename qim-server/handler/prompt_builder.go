package handler

import (
	"fmt"
	"qim-server/database"
	"qim-server/model"
	"strings"
	"time"
)

type PromptContext struct {
	ConversationID uint
	Conversation   *model.Conversation
	UserID         uint
	Group          *model.Group
	User           *model.User
	Messages       []model.Message
	Tasks          []model.Task
	GroupName      string
}

type PromptBuilder interface {
	BuildSystemPrompt(ctx *PromptContext) string
}

type SmartPromptBuilder struct {
	knowledgeSvc *KnowledgeService
}

func NewSmartPromptBuilder(knowledgeSvc *KnowledgeService) *SmartPromptBuilder {
	return &SmartPromptBuilder{
		knowledgeSvc: knowledgeSvc,
	}
}

func (b *SmartPromptBuilder) BuildSystemPrompt(ctx *PromptContext) string {
	prompt := "你是 QIM 企业即时通讯系统中的智能助手。"

	prompt += b.buildTimeInfo()
	prompt += b.buildGroupInfo(ctx)
	prompt += b.buildMessageHistory(ctx)
	prompt += b.buildUserInfo(ctx)
	prompt += b.buildTaskInfo(ctx)
	prompt += b.buildKnowledgeContext(ctx)
	prompt += b.buildGroupStats(ctx)
	prompt += b.buildRules()

	return prompt
}

func (b *SmartPromptBuilder) buildTimeInfo() string {
	now := time.Now()
	return fmt.Sprintf("\n当前时间：%s (%s)", now.Format("2006-01-02 15:04"), now.Weekday().String())
}

func (b *SmartPromptBuilder) buildGroupInfo(ctx *PromptContext) string {
	if ctx.Conversation.Type != "group" && ctx.Conversation.Type != "discussion" {
		return ""
	}

	info := "\n\n📋 群组信息："
	info += fmt.Sprintf("\n- 群名：%s", ctx.GroupName)
	info += fmt.Sprintf("\n- 群ID：%d", ctx.Conversation.ID)
	info += fmt.Sprintf("\n- 成员数：%d", len(ctx.Conversation.Members))
	info += "\n- 群成员："

	memberNames := make([]string, 0, len(ctx.Conversation.Members))
	for _, m := range ctx.Conversation.Members {
		name := m.User.Nickname
		if name == "" {
			name = m.User.Username
		}
		memberNames = append(memberNames, name)
	}
	info += strings.Join(memberNames, "、") + "。"

	return info
}

func (b *SmartPromptBuilder) buildMessageHistory(ctx *PromptContext) string {
	if len(ctx.Messages) == 0 {
		return ""
	}

	prompt := "\n\n📝 最近对话历史（按时间倒序）："

	for i := len(ctx.Messages) - 1; i >= 0; i-- {
		msg := ctx.Messages[i]
		senderName := msg.Sender.Nickname
		if senderName == "" {
			senderName = msg.Sender.Username
		}
		content := msg.Content
		if len(content) > 100 {
			content = content[:100] + "..."
		}
		prompt += fmt.Sprintf("\n[%s] %s: %s", msg.CreatedAt.Format("15:04"), senderName, content)
	}

	return prompt
}

func (b *SmartPromptBuilder) buildUserInfo(ctx *PromptContext) string {
	if ctx.User == nil {
		return ""
	}

	prompt := fmt.Sprintf("\n\n👤 当前提问用户：%s", ctx.User.Nickname)
	if ctx.User.Status == "disabled" {
		prompt += "（账号已禁用）"
	}

	return prompt
}

func (b *SmartPromptBuilder) buildTaskInfo(ctx *PromptContext) string {
	if len(ctx.Tasks) == 0 {
		return ""
	}

	prompt := "\n\n📋 用户待办任务（未完成）："
	for _, task := range ctx.Tasks {
		dueStr := "无截止日期"
		if task.DueDate != nil {
			dueStr = task.DueDate.Format("2006-01-02")
		}
		prompt += fmt.Sprintf("\n- [%s] %s (截止: %s)", strings.ToUpper(task.Priority[:1]), task.Title, dueStr)
	}
	if len(ctx.Tasks) >= 5 {
		prompt += "\n- ... 还有更多未显示"
	}

	return prompt
}

func (b *SmartPromptBuilder) buildKnowledgeContext(ctx *PromptContext) string {
	if b.knowledgeSvc == nil || ctx.GroupName == "" {
		return ""
	}

	knowledgeCtx := b.knowledgeSvc.BuildKnowledgeContext(ctx.GroupName)
	if knowledgeCtx == "" {
		return ""
	}

	return "\n\n📚 群组相关文档：\n" + knowledgeCtx
}

func (b *SmartPromptBuilder) buildGroupStats(ctx *PromptContext) string {
	db := database.GetDB()

	var totalMessages int64
	db.Model(&model.Message{}).Where("conversation_id = ?", ctx.ConversationID).Count(&totalMessages)

	var memberCount int64
	db.Model(&model.ConversationMember{}).Where("conversation_id = ?", ctx.ConversationID).Count(&memberCount)

	return fmt.Sprintf("\n\n📊 当前群状态：\n- 总消息数：%d\n- 成员数：%d", totalMessages, memberCount)
}

func (b *SmartPromptBuilder) buildRules() string {
	return `\n\n回复规则：
- 优先使用知识库中的内容回答
- 如果知识库中有相关内容，基于该内容给出准确答案
- 如果知识库中没有相关内容，使用你的通用知识回答，但明确说明"以下回答基于通用知识，建议核实"
- 回答要简洁、专业、准确
- 使用中文回复，除非用户用其他语言
- 如果是管理员提问，可以提供更详细的管理操作建议`
}

func (b *SmartPromptBuilder) BuildPromptContext(conversationID uint, userID uint) *PromptContext {
	db := database.GetDB()

	var conv model.Conversation
	if err := db.Preload("Members.User").First(&conv, conversationID).Error; err != nil {
		return nil
	}

	var user model.User
	db.First(&user, userID)

	var group *model.Group
	var groupName string
	if conv.Type == "group" || conv.Type == "discussion" {
		var g model.Group
		if err := db.Where("conversation_id = ?", conversationID).First(&g).Error; err == nil {
			group = &g
			groupName = g.Name
		}
	}

	var messages []model.Message
	db.Where("conversation_id = ?", conversationID).
		Preload("Sender").
		Order("created_at DESC").
		Limit(20).
		Find(&messages)

	var tasks []model.Task
	db.Where("user_id = ? AND status = 'todo'", userID).
		Order("due_date ASC").
		Limit(5).
		Find(&tasks)

	return &PromptContext{
		ConversationID: conversationID,
		Conversation:   &conv,
		UserID:         userID,
		Group:          group,
		User:           &user,
		Messages:       messages,
		Tasks:          tasks,
		GroupName:      groupName,
	}
}
