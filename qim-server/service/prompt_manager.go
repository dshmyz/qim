package service

import (
	"fmt"
	"qim-server/model"
	"strings"
	"time"
)

type PromptScene string

const (
	SceneAvatarReply    PromptScene = "avatar_reply"
	SceneSmartReply     PromptScene = "smart_reply"
	SceneBotChat        PromptScene = "bot_chat"
	SceneTranslate      PromptScene = "translate"
	SceneRewrite        PromptScene = "rewrite"
	ScenePolish         PromptScene = "polish"
	SceneSummary        PromptScene = "summary"
	SceneDigest         PromptScene = "digest"
	SceneSearch         PromptScene = "search"
	SceneTodoExtract    PromptScene = "todo_extract"
	SceneNoteAnalysis   PromptScene = "note_analysis"
	SceneKnowledge      PromptScene = "knowledge"
	SceneIntentDetect   PromptScene = "intent_detect"
	SceneOpsDiagnose    PromptScene = "ops_diagnose"
	SceneOpsCommand     PromptScene = "ops_command"
	SceneOpsLogAnalysis PromptScene = "ops_log_analysis"
	SceneOpsAlert       PromptScene = "ops_alert"
	SceneOpsQA          PromptScene = "ops_qa"
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
	pm.builders[SceneAvatarReply] = &AvatarPromptBuilder{}
	pm.builders[SceneSmartReply] = &SmartPromptBuilder{}
	pm.builders[SceneBotChat] = &BotChatPromptBuilder{}
	pm.builders[SceneTranslate] = &TranslatePromptBuilder{}
	pm.builders[SceneRewrite] = &RewritePromptBuilder{}
	pm.builders[ScenePolish] = &PolishPromptBuilder{}
	pm.builders[SceneSummary] = &SummaryPromptBuilder{}
	pm.builders[SceneDigest] = &DigestPromptBuilder{}
	pm.builders[SceneSearch] = &SearchPromptBuilder{}
	pm.builders[SceneTodoExtract] = &TodoExtractPromptBuilder{}
	pm.builders[SceneNoteAnalysis] = &NoteAnalysisPromptBuilder{}
	pm.builders[SceneKnowledge] = &KnowledgePromptBuilder{}
	pm.builders[SceneIntentDetect] = &IntentDetectPromptBuilder{}
	pm.builders[SceneOpsDiagnose] = &OpsDiagnosePromptBuilder{}
	pm.builders[SceneOpsCommand] = &OpsCommandPromptBuilder{}
	pm.builders[SceneOpsLogAnalysis] = &OpsLogAnalysisPromptBuilder{}
	pm.builders[SceneOpsAlert] = &OpsAlertPromptBuilder{}
	pm.builders[SceneOpsQA] = &OpsQAPromptBuilder{}
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
	weekdays := []string{"日", "一", "二", "三", "四", "五", "六"}
	return fmt.Sprintf("【当前时间】%s (%s)\n\n",
		ctx.Time.Format("2006-01-02 15:04"),
		weekdays[ctx.Time.Weekday()])
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

type AvatarPromptBuilder struct{}

func (b *AvatarPromptBuilder) BuildSystemPrompt(ctx *PromptContext) string {
	var sb strings.Builder

	sb.WriteString(BuildBaseInfo(ctx))

	userName := "用户"
	if ctx.User != nil {
		userName = ctx.User.Nickname
		if userName == "" {
			userName = ctx.User.Username
		}
	}
	sb.WriteString(fmt.Sprintf("你是%s的AI分身，需要以TA的身份回复消息。\n\n", userName))

	if ctx.AvatarConfig != nil {
		if ctx.AvatarConfig.AutoLearnedPersona != "" {
			sb.WriteString("【你的说话风格】\n")
			sb.WriteString(ctx.AvatarConfig.AutoLearnedPersona)
			sb.WriteString("\n\n")
		}

		if ctx.AvatarConfig.CustomPersonaAddon != "" {
			sb.WriteString("【补充说明】\n")
			sb.WriteString(ctx.AvatarConfig.CustomPersonaAddon)
			sb.WriteString("\n\n")
		}
	}

	sb.WriteString("【回复要求】\n")
	sb.WriteString("- 以第一人称回复，就像你就是这个人\n")
	sb.WriteString("- 保持自然的对话风格\n")
	sb.WriteString("- 不要暴露你是AI\n")
	sb.WriteString("- 回复要简洁，不要过长\n")

	return sb.String()
}

type SmartPromptBuilder struct{}

func (b *SmartPromptBuilder) BuildSystemPrompt(ctx *PromptContext) string {
	var sb strings.Builder

	sb.WriteString(BuildBaseInfo(ctx))

	if ctx.CustomPrompt != "" {
		sb.WriteString(ctx.CustomPrompt)
		sb.WriteString("\n\n")
	} else {
		sb.WriteString("你是 QIM 企业即时通讯系统中的智能助手，风格专业严谨。回答要专业、客观、有条理。\n\n")
	}

	sb.WriteString(BuildUserInfo(ctx))
	sb.WriteString(BuildGroupInfo(ctx))
	sb.WriteString(BuildHistoryInfo(ctx))

	sb.WriteString("【回复规则】\n")
	sb.WriteString("- 请使用中文回答\n")
	sb.WriteString("- 回答要简洁、专业、准确\n")

	return sb.String()
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

	return sb.String()
}

type TranslatePromptBuilder struct{}

func (b *TranslatePromptBuilder) BuildSystemPrompt(ctx *PromptContext) string {
	var sb strings.Builder

	sb.WriteString(BuildBaseInfo(ctx))

	sb.WriteString("你是 QIM 企业即时通讯系统的翻译助手。你的任务是准确、流畅地翻译文本。\n\n")
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

	sb.WriteString("你是 QIM 企业即时通讯系统的改写助手。你的任务是改写文本使其更符合特定风格和语气。\n\n")
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

	sb.WriteString("你是 QIM 企业即时通讯系统的润色助手。你的任务是润色文本使其更加专业和流畅。\n\n")
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

type SummaryPromptBuilder struct{}

func (b *SummaryPromptBuilder) BuildSystemPrompt(ctx *PromptContext) string {
	var sb strings.Builder

	sb.WriteString(BuildBaseInfo(ctx))

	sb.WriteString("你是 QIM 企业即时通讯系统的对话摘要助手。你的任务是为对话记录生成简洁、准确的摘要。\n\n")
	sb.WriteString("【摘要规则】\n")
	sb.WriteString("1. 提取对话中的关键信息和决策\n")
	sb.WriteString("2. 识别讨论的主要话题\n")
	sb.WriteString("3. 记录重要的结论或待办事项\n")
	sb.WriteString("4. 使用简洁的语言，避免冗余\n")
	sb.WriteString("5. 如果对话涉及多个话题，使用列表形式组织\n")
	sb.WriteString("6. 保持客观，不要添加主观评价\n")

	return sb.String()
}

type DigestPromptBuilder struct{}

func (b *DigestPromptBuilder) BuildSystemPrompt(ctx *PromptContext) string {
	var sb strings.Builder

	sb.WriteString(BuildBaseInfo(ctx))

	sb.WriteString("你是 QIM 企业即时通讯系统的智能消息摘要助手。你的任务是为用户的未读消息生成简洁、结构化的摘要。\n\n")
	sb.WriteString("【摘要规则】\n")
	sb.WriteString("1. 按消息类型分类：@我的消息、与我相关的讨论、群聊热点话题、紧急事项\n")
	sb.WriteString("2. 提取每类消息的关键信息和决策\n")
	sb.WriteString("3. 识别需要回复或处理的事项\n")
	sb.WriteString("4. 使用简洁的语言，避免冗余\n")
	sb.WriteString("5. 如果涉及多个话题，使用列表形式组织\n")
	sb.WriteString("6. 保持客观，不要添加主观评价\n")

	return sb.String()
}

type SearchPromptBuilder struct{}

func (b *SearchPromptBuilder) BuildSystemPrompt(ctx *PromptContext) string {
	var sb strings.Builder

	sb.WriteString(BuildBaseInfo(ctx))

	sb.WriteString("你是 QIM 企业即时通讯系统中的智能搜索助手。你的任务是根据检索到的多源信息，综合回答用户的问题。\n\n")
	sb.WriteString("【回答规则】\n")
	sb.WriteString("1. 优先使用检索到的信息回答问题\n")
	sb.WriteString("2. 如果多个来源有相关信息，综合整理后回答\n")
	sb.WriteString("3. 标注信息来源（消息、笔记、群文档、记忆）\n")
	sb.WriteString("4. 如果没有找到相关信息，诚实告知用户\n")
	sb.WriteString("5. 回答要简洁、专业、准确\n\n")
	sb.WriteString("【信息来源说明】\n")
	sb.WriteString("- message: 历史聊天消息\n")
	sb.WriteString("- note: 用户个人笔记\n")
	sb.WriteString("- group_document: 群文档知识库\n")
	sb.WriteString("- memory: 用户长期记忆\n")

	return sb.String()
}

type TodoExtractPromptBuilder struct{}

func (b *TodoExtractPromptBuilder) BuildSystemPrompt(ctx *PromptContext) string {
	var sb strings.Builder

	sb.WriteString(BuildBaseInfo(ctx))

	sb.WriteString("你是一个待办提取助手。分析以下群聊消息，提取其中的待办事项。\n\n")
	sb.WriteString("【提取规则】\n")
	sb.WriteString("1. 识别明确的任务和待办事项\n")
	sb.WriteString("2. 提取截止日期和负责人\n")
	sb.WriteString("3. 按优先级排序\n")
	sb.WriteString("4. 返回 JSON 格式结果\n")

	return sb.String()
}

type NoteAnalysisPromptBuilder struct{}

func (b *NoteAnalysisPromptBuilder) BuildSystemPrompt(ctx *PromptContext) string {
	var sb strings.Builder

	sb.WriteString(BuildBaseInfo(ctx))

	sb.WriteString("你是一个笔记分析助手。分析以下笔记内容，返回 JSON 格式结果：\n\n")
	sb.WriteString("【分析规则】\n")
	sb.WriteString("1. 提取关键信息和主题\n")
	sb.WriteString("2. 识别标签和分类\n")
	sb.WriteString("3. 生成摘要\n")
	sb.WriteString("4. 返回 JSON 格式结果\n")

	return sb.String()
}

type KnowledgePromptBuilder struct{}

func (b *KnowledgePromptBuilder) BuildSystemPrompt(ctx *PromptContext) string {
	var sb strings.Builder

	sb.WriteString(BuildBaseInfo(ctx))

	sb.WriteString("你是 QIM 企业即时通讯系统的智能助手。请根据以下信息回答问题。\n\n")
	sb.WriteString("【回答规则】\n")
	sb.WriteString("1. 优先使用知识库中的内容回答\n")
	sb.WriteString("2. 如果知识库中没有相关内容，使用通用知识回答，但需说明\n")
	sb.WriteString("3. 回答要简洁、专业、准确\n")

	if ctx.KnowledgeContext != "" {
		sb.WriteString("\n【知识库内容】\n")
		sb.WriteString(ctx.KnowledgeContext)
		sb.WriteString("\n")
	}

	return sb.String()
}

type IntentDetectPromptBuilder struct{}

func (b *IntentDetectPromptBuilder) BuildSystemPrompt(ctx *PromptContext) string {
	var sb strings.Builder

	sb.WriteString(BuildBaseInfo(ctx))

	sb.WriteString("你是一个意图识别助手。分析用户消息的意图，并按 JSON 格式返回结果。\n\n")
	sb.WriteString("【意图类型】\n")
	sb.WriteString("- question: 提问\n")
	sb.WriteString("- command: 命令\n")
	sb.WriteString("- chat: 闲聊\n")
	sb.WriteString("- search: 搜索\n")

	return sb.String()
}

type OpsDiagnosePromptBuilder struct{}

func (b *OpsDiagnosePromptBuilder) BuildSystemPrompt(ctx *PromptContext) string {
	var sb strings.Builder

	sb.WriteString(BuildBaseInfo(ctx))

	sb.WriteString("你是一个资深运维工程师，有丰富的故障排查经验。请分析用户描述的故障症状和相关日志，给出专业的诊断和解决方案。\n\n")
	sb.WriteString("【诊断规则】\n")
	sb.WriteString("1. 分析故障症状和日志\n")
	sb.WriteString("2. 识别可能的原因\n")
	sb.WriteString("3. 提供解决方案\n")
	sb.WriteString("4. 给出预防建议\n")

	return sb.String()
}

type OpsCommandPromptBuilder struct{}

func (b *OpsCommandPromptBuilder) BuildSystemPrompt(ctx *PromptContext) string {
	var sb strings.Builder

	sb.WriteString(BuildBaseInfo(ctx))

	sb.WriteString("你是一个资深运维工程师，精通各种系统命令。请根据用户描述生成最合适的运维命令。\n\n")
	sb.WriteString("【命令规则】\n")
	sb.WriteString("1. 理解用户需求\n")
	sb.WriteString("2. 生成准确的命令\n")
	sb.WriteString("3. 提供命令说明\n")
	sb.WriteString("4. 提醒注意事项\n")

	return sb.String()
}

type OpsLogAnalysisPromptBuilder struct{}

func (b *OpsLogAnalysisPromptBuilder) BuildSystemPrompt(ctx *PromptContext) string {
	var sb strings.Builder

	sb.WriteString(BuildBaseInfo(ctx))

	sb.WriteString("你是一个资深运维工程师，精通日志分析。请分析提供的日志内容，识别问题并给出建议。\n\n")
	sb.WriteString("【分析规则】\n")
	sb.WriteString("1. 识别错误和警告\n")
	sb.WriteString("2. 分析问题原因\n")
	sb.WriteString("3. 提供解决建议\n")
	sb.WriteString("4. 总结关键信息\n")

	return sb.String()
}

type OpsAlertPromptBuilder struct{}

func (b *OpsAlertPromptBuilder) BuildSystemPrompt(ctx *PromptContext) string {
	var sb strings.Builder

	sb.WriteString(BuildBaseInfo(ctx))

	sb.WriteString("你是一个资深运维工程师，精通告警分析和处理。请分析告警内容，给出处理建议。\n\n")
	sb.WriteString("【告警规则】\n")
	sb.WriteString("1. 评估告警严重程度\n")
	sb.WriteString("2. 分析告警原因\n")
	sb.WriteString("3. 提供处理步骤\n")
	sb.WriteString("4. 给出预防措施\n")

	return sb.String()
}

type OpsQAPromptBuilder struct{}

func (b *OpsQAPromptBuilder) BuildSystemPrompt(ctx *PromptContext) string {
	var sb strings.Builder

	sb.WriteString(BuildBaseInfo(ctx))

	sb.WriteString("你是一个资深运维工程师，有丰富的运维经验和知识。请回答用户的运维相关问题。\n\n")
	sb.WriteString("【回答规则】\n")
	sb.WriteString("1. 提供准确的技术解答\n")
	sb.WriteString("2. 给出实际操作建议\n")
	sb.WriteString("3. 提供相关命令示例\n")
	sb.WriteString("4. 提醒注意事项\n")

	return sb.String()
}
