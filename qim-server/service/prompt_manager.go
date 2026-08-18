package service

import (
	"fmt"
	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/aiprompt"
	"github.com/dshmyz/qim/qim-server/pkg/productname"
	"strings"
	"time"
)

type PromptScene string

const (
	// 仅保留实际被调用的 scene：bot 1:1 与文本处理（翻译/改写/润色）。
	// 其余 scene（分身/群助手/意图/摘要/搜索/待办/笔记/知识/运维）均已迁到独立
	// graph 或工具实现（avatar_reply_graph / smart_reply_graph / UnifiedSearchGraph /
	// ai.IntentDetector / 各类 AI Tool 等），PromptManager 不再承载，勿再为它们注册 builder。
	SceneBotChat   PromptScene = "bot_chat"
	SceneTranslate PromptScene = "translate"
	SceneRewrite   PromptScene = "rewrite"
	ScenePolish    PromptScene = "polish"
)

type PromptContext struct {
	Time         time.Time
	UserID       uint
	User         *model.User
	Group        *model.Group
	Conversation *model.Conversation
	Messages     []model.Message
	Bot          *model.Bot
	AvatarConfig *model.AvatarConfig

	Intent       string
	SourceLang   string
	TargetLang   string
	Style        string
	Tone         string
	Language     string
	CustomPrompt string

	KnowledgeContext string
	MemoryContext    string
	NoteContext      string
	GroupKnowledge   string
	History          string

	AdditionalData map[string]interface{}
}

type ScenePromptBuilder interface {
	BuildSystemPrompt(ctx *PromptContext) string
}

type PromptManager struct {
	builders map[PromptScene]ScenePromptBuilder
}

func NewPromptManager() *PromptManager {
	pm := &PromptManager{
		builders: make(map[PromptScene]ScenePromptBuilder),
	}

	pm.registerBuilders()

	return pm
}

func (pm *PromptManager) registerBuilders() {
	pm.builders[SceneBotChat] = &BotChatPromptBuilder{}
	pm.builders[SceneTranslate] = &TranslatePromptBuilder{}
	pm.builders[SceneRewrite] = &RewritePromptBuilder{}
	pm.builders[ScenePolish] = &PolishPromptBuilder{}
}

func (pm *PromptManager) BuildSystemPrompt(scene PromptScene, ctx *PromptContext) string {
	if ctx.Time.IsZero() {
		ctx.Time = time.Now()
	}

	builder, ok := pm.builders[scene]
	if !ok {
		return pm.buildDefaultPrompt(ctx)
	}

	return builder.BuildSystemPrompt(ctx)
}

func (pm *PromptManager) buildDefaultPrompt(ctx *PromptContext) string {
	var sb strings.Builder
	sb.WriteString(BuildBaseInfo(ctx))
	sb.WriteString("你是一个智能助手，帮助用户解决问题。")
	return sb.String()
}

func BuildBaseInfo(ctx *PromptContext) string {
	return aiprompt.FormatTimeLine(ctx.Time) + "\n\n"
}

func BuildUserInfo(ctx *PromptContext) string {
	if ctx.User == nil {
		return ""
	}
	return fmt.Sprintf("【当前用户】%s\n\n", ctx.User.Nickname)
}

func BuildGroupInfo(ctx *PromptContext) string {
	if ctx.Group == nil {
		return ""
	}
	return fmt.Sprintf("【群聊信息】群名：%s\n\n", ctx.Group.Name)
}

func BuildHistoryInfo(ctx *PromptContext) string {
	if ctx.History == "" {
		return ""
	}
	return fmt.Sprintf("【对话历史】\n%s\n\n", ctx.History)
}

type BotChatPromptBuilder struct{}

func (b *BotChatPromptBuilder) BuildSystemPrompt(ctx *PromptContext) string {
	var sb strings.Builder

	sb.WriteString(BuildBaseInfo(ctx))

	if ctx.CustomPrompt != "" {
		sb.WriteString(ctx.CustomPrompt)
	} else {
		sb.WriteString("你是一个智能助手，帮助用户解决问题。")
	}

	// 能力自述：注入静态能力，让 Bot 被问「具备哪些能力」时能如实回答（Bot 无工具，故不注入工具段）。
	if capPrompt := ai.BuildStaticCapabilitiesPrompt(); capPrompt != "" {
		if !strings.HasSuffix(sb.String(), "\n") {
			sb.WriteString("\n")
		}
		sb.WriteString("\n【能力与工具】\n" + capPrompt)
	}

	return sb.String()
}

type TranslatePromptBuilder struct{}

func (b *TranslatePromptBuilder) BuildSystemPrompt(ctx *PromptContext) string {
	var sb strings.Builder

	sb.WriteString(BuildBaseInfo(ctx))

	sb.WriteString("你是 " + productname.Name + " 企业即时通讯系统的翻译助手。你的任务是准确、流畅地翻译文本。\n\n")
	sb.WriteString("【翻译规则】\n")
	sb.WriteString("1. 保持原文的语义和语气\n")
	sb.WriteString("2. 使用目标语言的自然表达方式\n")
	sb.WriteString("3. 保留专业术语和专有名词\n")
	sb.WriteString("4. 保持原文的格式和结构\n")
	sb.WriteString("5. 只输出翻译结果，不要额外解释\n")

	if ctx.SourceLang != "" {
		sb.WriteString(fmt.Sprintf("\n【源语言】%s\n", ctx.SourceLang))
	}
	if ctx.TargetLang != "" {
		sb.WriteString(fmt.Sprintf("【目标语言】%s\n", ctx.TargetLang))
	}

	return sb.String()
}

type RewritePromptBuilder struct{}

func (b *RewritePromptBuilder) BuildSystemPrompt(ctx *PromptContext) string {
	var sb strings.Builder

	sb.WriteString(BuildBaseInfo(ctx))

	sb.WriteString("你是 " + productname.Name + " 企业即时通讯系统的改写助手。你的任务是改写文本使其更符合特定风格和语气。\n\n")
	sb.WriteString("【改写规则】\n")
	sb.WriteString("1. 保持原文的核心意思\n")
	sb.WriteString("2. 调整表达方式以符合指定风格\n")
	sb.WriteString("3. 确保改写后的文本流畅自然\n")
	sb.WriteString("4. 保持原文的格式和结构\n")
	sb.WriteString("5. 只输出改写结果，不要额外解释\n")

	if ctx.Style != "" {
		sb.WriteString(fmt.Sprintf("\n【风格】%s\n", ctx.Style))
	}
	if ctx.Tone != "" {
		sb.WriteString(fmt.Sprintf("【语气】%s\n", ctx.Tone))
	}

	return sb.String()
}

type PolishPromptBuilder struct{}

func (b *PolishPromptBuilder) BuildSystemPrompt(ctx *PromptContext) string {
	var sb strings.Builder

	sb.WriteString(BuildBaseInfo(ctx))

	sb.WriteString("你是 " + productname.Name + " 企业即时通讯系统的润色助手。你的任务是润色文本使其更加专业和流畅。\n\n")
	sb.WriteString("【润色规则】\n")
	sb.WriteString("1. 修正语法和拼写错误\n")
	sb.WriteString("2. 优化句子结构和表达\n")
	sb.WriteString("3. 保持原文的语义和语气\n")
	sb.WriteString("4. 使文本更加简洁和专业\n")
	sb.WriteString("5. 只输出润色结果，不要额外解释\n")

	if ctx.Language != "" {
		sb.WriteString(fmt.Sprintf("\n【语言】%s\n", ctx.Language))
	}

	return sb.String()
}
