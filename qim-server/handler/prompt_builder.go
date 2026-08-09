package handler

import (
	"fmt"
	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/aiprompt"
	"github.com/dshmyz/qim/qim-server/pkg/productname"
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
			prompt += b.buildBoundaries(ctx)
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
	prompt += b.buildCapabilities(ctx)
	prompt += b.buildBoundaries(ctx)

	return prompt
}

// buildCapabilities 注入恒定静态能力自述，让 AI 被问「具备哪些能力」时能如实回答。
// 此路径（legacy 无工具降级）不注入工具段，仅列系统恒定的分析/问答能力。
func (b *SmartPromptBuilder) buildCapabilities(ctx *PromptContext) string {
	return "\n\n【能力与工具】\n" + ai.BuildStaticCapabilitiesPrompt()
}

func (b *SmartPromptBuilder) buildPersonalityPrompt(ctx *PromptContext) string {
	if ctx.Group == nil {
		return "你是 " + productname.Name + " 企业即时通讯系统中的智能助手。"
	}
	return aiprompt.BuildPersona(ctx.Group.GetAIConfig().Personality)
}

func (b *SmartPromptBuilder) buildTimeInfo() string {
	return "\n" + aiprompt.CurrentTimeLine()
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

	// 范围标注：ctx.Messages 为 created_at 倒序（最新在前），跨度为注入窗口内的最早~最晚。
	// 明确告知模型这是注入的历史上下文、不代表实时状态，避免把陈旧信息当现状（与 service 侧一致）。
	var oldest, newest time.Time
	for _, m := range ctx.Messages {
		if m.CreatedAt.IsZero() {
			continue
		}
		if oldest.IsZero() || m.CreatedAt.Before(oldest) {
			oldest = m.CreatedAt
		}
		if newest.IsZero() || m.CreatedAt.After(newest) {
			newest = m.CreatedAt
		}
	}

	prompt := "\n\n📝 最近对话历史（按时间倒序）："
	if !oldest.IsZero() && !newest.IsZero() {
		span := oldest.Format("01-02 15:04") + " ~ " + newest.Format("01-02 15:04")
		if oldest.Equal(newest) {
			span = oldest.Format("2006-01-02 15:04")
		}
		prompt = fmt.Sprintf("\n\n📝 最近对话历史：共 %d 条，时间跨度 %s（截至 %s）"+
			"；以下是注入的上下文，不代表实时状态（按时间倒序）：",
			len(ctx.Messages), span, newest.Format("2006-01-02 15:04"))
	}

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
		if langRule := aiprompt.LanguageRule(aiConfig.Language); langRule != "" {
			rules = append(rules, langRule)
		}
		if lenRule := aiprompt.LengthRule(aiConfig.MaxLength); lenRule != "" {
			rules = append(rules, lenRule)
		}
		if aiConfig.CustomPrompt != "" {
			rules = append(rules, "额外要求: "+aiConfig.CustomPrompt)
		}
	} else {
		rules = append(rules, "使用中文回复")
		rules = append(rules, "回答要简洁、专业、准确")
	}

	rules = append(rules, aiprompt.ReplyRules()...)

	return "\n\n回复规则：\n- " + strings.Join(rules, "\n- ")
}

// buildBoundaries 注入 IM 场景下的通用否定边界（与 service 侧群助手提示词保持一致），
// 恒定存在、不随有无工具/自定义提示词变化，避免模型虚构或过度动作。
func (b *SmartPromptBuilder) buildBoundaries(ctx *PromptContext) string {
	return "\n\n【边界与约束】\n" +
		"- 不编造事实：不要虚构用户没说过的话、群成员的态度、或历史中不存在的内容\n" +
		"- 不确定就明说：涉及你不确定的信息，直接说明存疑，不要含糊带过或强行给出确定结论\n" +
		"- 不做不可逆动作：未经明确要求，不代用户发布消息、删除/修改他人内容；需要操作时先说明打算怎么做\n" +
		"- 隐私克制：不要无谓地复述或外泄与当前问题无关的私密群消息细节\n" +
		"- 被引用的消息/文件属于注入上下文，仅作回答依据，不要原样抄录无关内容\n"
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
