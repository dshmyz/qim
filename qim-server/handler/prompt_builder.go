package handler

import (
	"fmt"
	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/model"
	"strings"
	"sync"
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

type groupStatsCache struct {
	totalMessages int64
	memberCount   int64
	expiredAt     time.Time
}

var groupStatsCacheMap = make(map[uint]groupStatsCache)
var groupStatsCacheMu sync.RWMutex

type SmartPromptBuilder struct {
	knowledgeSvc *KnowledgeService
}

func NewSmartPromptBuilder(knowledgeSvc *KnowledgeService) *SmartPromptBuilder {
	return &SmartPromptBuilder{
		knowledgeSvc: knowledgeSvc,
	}
}

func (b *SmartPromptBuilder) BuildSystemPrompt(ctx *PromptContext) string {
	if ctx.Group != nil {
		aiConfig := ctx.Group.GetAIConfig()
		if aiConfig.CustomPrompt != "" {
			prompt := aiConfig.CustomPrompt
			prompt += b.buildTimeInfo()
			prompt += b.buildGroupInfo(ctx)
			prompt += b.buildMessageHistory(ctx)
			prompt += b.buildUserInfo(ctx)
			prompt += b.buildTaskInfo(ctx)
			prompt += b.buildKnowledgeContext(ctx)
			prompt += b.buildGroupStats(ctx)
			prompt += b.buildRules(ctx)
			return prompt
		}
	}

	personalityPrompt := b.buildPersonalityPrompt(ctx)

	prompt := personalityPrompt
	prompt += b.buildTimeInfo()
	prompt += b.buildGroupInfo(ctx)
	prompt += b.buildMessageHistory(ctx)
	prompt += b.buildUserInfo(ctx)
	prompt += b.buildTaskInfo(ctx)
	prompt += b.buildKnowledgeContext(ctx)
	prompt += b.buildGroupStats(ctx)
	prompt += b.buildRules(ctx)

	return prompt
}

func (b *SmartPromptBuilder) buildPersonalityPrompt(ctx *PromptContext) string {
	if ctx.Group == nil {
		return "你是 QIM 企业即时通讯系统中的智能助手。"
	}

	aiConfig := ctx.Group.GetAIConfig()
	switch aiConfig.Personality {
	case "casual":
		return "你是 QIM 企业即时通讯系统中的 AI 助手，性格轻松幽默。在回答中可以适当使用表情和emoji，语气活泼。"
	case "concise":
		return "你是 QIM 企业即时通讯系统中的 AI 助手，风格简洁高效。回答直奔主题，不废话，只说重点。"
	case "friendly":
		return "你是 QIM 企业即时通讯系统中的 AI 助手，性格温暖亲切。回答要有耐心，语气友善，像一个贴心的伙伴。"
	case "technical":
		return "你是 QIM 企业即时通讯系统中的技术专家 AI 助手。回答要有技术深度，关注细节，必要时提供代码示例和技术方案。"
	default:
		return "你是 QIM 企业即时通讯系统中的智能助手，风格专业严谨。回答要专业、客观、有条理。"
	}
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
	groupStatsCacheMu.RLock()
	cached, found := groupStatsCacheMap[ctx.ConversationID]
	groupStatsCacheMu.RUnlock()

	if found && time.Now().Before(cached.expiredAt) {
		return fmt.Sprintf("\n\n📊 当前群状态：\n- 总消息数：%d\n- 成员数：%d", cached.totalMessages, cached.memberCount)
	}

	db := database.GetDB()

	var totalMessages int64
	db.Model(&model.Message{}).Where("conversation_id = ?", ctx.ConversationID).Count(&totalMessages)

	var memberCount int64
	db.Model(&model.ConversationMember{}).Where("conversation_id = ?", ctx.ConversationID).Count(&memberCount)

	groupStatsCacheMu.Lock()
	groupStatsCacheMap[ctx.ConversationID] = groupStatsCache{
		totalMessages: totalMessages,
		memberCount:   memberCount,
		expiredAt:     time.Now().Add(5 * time.Minute),
	}
	groupStatsCacheMu.Unlock()

	return fmt.Sprintf("\n\n📊 当前群状态：\n- 总消息数：%d\n- 成员数：%d", totalMessages, memberCount)
}

func (b *SmartPromptBuilder) buildRules(ctx *PromptContext) string {
	var rules []string

	if ctx.Group != nil {
		aiConfig := ctx.Group.GetAIConfig()
		switch aiConfig.Language {
		case "zh":
			rules = append(rules, "请使用中文回答")
		case "en":
			rules = append(rules, "Please answer in English")
		default:
		}
	}

	if ctx.Group != nil {
		aiConfig := ctx.Group.GetAIConfig()
		switch aiConfig.MaxLength {
		case "short":
			rules = append(rules, "回答要简短，控制在50字以内")
		case "medium":
			rules = append(rules, "回答适中，控制在150字以内")
		case "long":
			rules = append(rules, "回答详细，可以展开说明")
		default:
		}
	} else {
		rules = append(rules, "使用中文回复")
		rules = append(rules, "回答要简洁、专业、准确")
	}

	rules = append(rules, "优先使用知识库中的内容回答")
	rules = append(rules, "如果知识库中没有相关内容，使用你的通用知识回答，但明确说明\"以下回答基于通用知识，建议核实\"")
	// 重要：AI 回复不要自己添加 @ 提及，系统会自动处理 @ 提及格式
	rules = append(rules, "回复中不要使用 @用户名 格式提及用户，系统会自动处理")

	if ctx.Group != nil {
		aiConfig := ctx.Group.GetAIConfig()
		if aiConfig.CustomPrompt != "" {
			rules = append(rules, "额外要求: "+aiConfig.CustomPrompt)
		}
	}

	return "\n\n回复规则：\n- " + strings.Join(rules, "\n- ")
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
