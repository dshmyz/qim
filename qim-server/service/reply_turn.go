package service

import (
	"unicode/utf8"

	"github.com/dshmyz/qim/qim-server/ai"
)

// reply_turn.go —— 群助手「回应回合」的唯一决策点。
//
// 一次回应 = 触发 → 计划 → 执行 → 投递 → 复盘。本文件只负责「计划」：把分散在
// handler / graph / ai 三层的布尔（HasTools、AllowedTools、SkipKnowledge、
// ExternalToolGuide、visionOnly、liveSend）收敛成一个纯函数的输出，供执行与投递消费。
// 以后调整"什么情况下带工具 / 带图怎么处理 / 核验挡不挡发送"，只改这里一处。

type TurnMode int

const (
	// TurnDirectChat 无工具直接回答：小闲聊不挂工具，省掉 ReAct 一轮往返。
	TurnDirectChat TurnMode = iota
	// TurnToolReAct 带工具的 ReAct：查任务/日历/文件、群管理指令、外部 MCP。
	TurnToolReAct
	// TurnVisionOnly 带图的纯视觉单轮：图片与 tools 永不同请求。
	TurnVisionOnly
	// TurnVisionThenAct 先识图、再按工具执行：两段式，兼顾看图与办事。
	TurnVisionThenAct
)

func (m TurnMode) String() string {
	switch m {
	case TurnDirectChat:
		return "direct_chat"
	case TurnToolReAct:
		return "tool_react"
	case TurnVisionOnly:
		return "vision_only"
	case TurnVisionThenAct:
		return "vision_then_act"
	}
	return "unknown"
}

// TurnDelivery 投递策略：正文边流边发，还是先缓冲等核验。
type TurnDelivery int

const (
	DeliveryLive TurnDelivery = iota
	DeliveryBuffered
)

// TurnGate 核验策略：关闭 / 事后异步记录（不阻挡）/ 发送前阻断。
type TurnGate int

const (
	GateOff TurnGate = iota
	GateAsync
	GateBlock
)

// ReplyTurnPlan 一次回应的执行计划，消费方不得再自行判断模式。
type ReplyTurnPlan struct {
	Mode TurnMode
	// Tools 非空表示挂工具（ToolReAct / VisionThenAct 第二段）。空 = 不声明任何工具能力。
	Tools []string
	// ExternalGuide 是否注入外部工具的触发/输出组织指引（仅外部 MCP 启用且确有工具时）。
	ExternalGuide bool
	// SkipKnowledge 是否跳过知识/记忆注入（管理指令不需要知识上下文）。
	SkipKnowledge bool
	Delivery      TurnDelivery
	Gate          TurnGate
	// Status 依次下发的可见状态文案（首字前安抚用）：第一段、第二段各一条。
	Status []string
}

// TurnInput 计划函数的全部输入（均为调用方已读到的既有事实，不做 IO）。
type TurnInput struct {
	Intent   *ai.MessageIntent
	Question string
	// HasImage 触发消息或所引用的消息确为图片且已成功读取。
	HasImage bool
	// GroupTools 本次群作用域可用的工具（内置 ∪ 外部）。
	GroupTools []string
	// ExternalToolsEnabled 外部 MCP 是否对群启用（决定是否注入外部工具指引）。
	ExternalToolsEnabled bool
	// VisionDisableTools ai.vision_disable_tools：带图时是否禁用工具（默认 true）。
	VisionDisableTools bool
	// QualityGateEnabled / MentionGateBlock 对应 ai.reply_quality_gate*。
	QualityGateEnabled bool
	MentionGateBlock   bool
	// Auto 是否 auto 自动回复（无流式气泡，核验必须能阻断）。
	Auto bool
}

// smallTalkMaxRunes 小闲聊判定阈值：意图为 chat 且提问不超过该长度时直答、不挂工具。
// 只收窄明确的寒暄（"你好""谢谢""在吗"），长句即使被判 chat 仍保留工具，宁可多一轮
// 也不让"帮我总结下这个群"这类能力因为意图误判而丢失。
const smallTalkMaxRunes = 12

// PlanGroupTurn 计算一次群助手回应的计划。纯函数：相同输入必然相同输出。
func PlanGroupTurn(in TurnInput) ReplyTurnPlan {
	plan := ReplyTurnPlan{Mode: TurnDirectChat, Delivery: DeliveryLive}

	// 核验策略：auto 恒阻断（无流式气泡，发出去就收不回）；@AI 由配置决定。
	switch {
	case in.Auto:
		plan.Gate = GateBlock
	case !in.QualityGateEnabled:
		plan.Gate = GateOff
	case in.MentionGateBlock:
		plan.Gate = GateBlock
	default:
		plan.Gate = GateAsync
	}
	// 只有核验具备阻断能力时才需要缓冲；旁路/关闭时缓冲换不来任何东西，纯增加首字延迟。
	if plan.Gate == GateBlock {
		plan.Delivery = DeliveryBuffered
	}

	intentType := ""
	if in.Intent != nil {
		intentType = in.Intent.Type
	}
	needsAction := intentType == "command" || intentType == "query" || intentType == "todo"
	// 管理指令不需要知识/记忆上下文（与既有 SkipKnowledge 语义一致）。
	plan.SkipKnowledge = intentType == "command"

	switch {
	case in.HasImage && in.VisionDisableTools:
		// R1：图片与 tools 永不同请求。需要行动时分两段，否则只识图。
		if needsAction {
			plan.Mode = TurnVisionThenAct
		} else {
			plan.Mode = TurnVisionOnly
		}
	case in.HasImage:
		// 显式关掉 ai.vision_disable_tools：回到"带图也挂工具"的旧行为。
		plan.Mode = TurnToolReAct
	case needsAction:
		plan.Mode = TurnToolReAct
	case intentType == "chat" && utf8.RuneCountInString(in.Question) <= smallTalkMaxRunes:
		// R2：小闲聊直答，不挂工具。
		plan.Mode = TurnDirectChat
	default:
		plan.Mode = TurnToolReAct
	}

	// 群作用域一个工具都没有时不得进带工具模式：ai.isToolAllowed 把"空白名单"解释为
	// "全部放行"，会让提示词声称无工具、实际却把整个工具注册表交给模型。
	if len(in.GroupTools) == 0 {
		switch plan.Mode {
		case TurnToolReAct:
			plan.Mode = TurnDirectChat
		case TurnVisionThenAct:
			plan.Mode = TurnVisionOnly
		}
	}

	if plan.Mode == TurnToolReAct || plan.Mode == TurnVisionThenAct {
		plan.Tools = append([]string(nil), in.GroupTools...)
		plan.ExternalGuide = in.ExternalToolsEnabled && len(plan.Tools) > 0
	}

	// R3：首字前先给可见状态。只对视觉相关阶段下发——普通带工具已有统一的「思考中」
	// 占位，再追加一句「正在处理」是噪音。
	if plan.Mode == TurnVisionOnly || plan.Mode == TurnVisionThenAct {
		plan.Status = append(plan.Status, "正在看图片…")
	}
	if plan.Mode == TurnVisionThenAct {
		plan.Status = append(plan.Status, "正在处理…")
	}
	return plan
}

// WantsTools 报告该计划是否需要工具。
func (p ReplyTurnPlan) WantsTools() bool { return len(p.Tools) > 0 }
