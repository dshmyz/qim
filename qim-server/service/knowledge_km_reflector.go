package service

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/dshmyz/qim/qim-server/ai"
)

// MemoryReflection 是"记忆反射（Recall→Consolidate）"闭环产出的结构化总结。
// 相比直接落原始消息，反射会把既有记忆+群知识折叠成一条带主题/事实/摘要的合并记忆，
// 并在 metadata 打上 knowledge_memory 标记供后续识别。
type MemoryReflection struct {
	Summary    string   `json:"summary"`
	Facts      []string `json:"facts"`
	Themes     []string `json:"themes"`
	Entities   []string `json:"entities"`
	Importance float64  `json:"importance"` // 1-5 档位（与 RememberVerdict 一致）
	Type       string   `json:"type"`       // fact/preference/event：记忆分类标签
	// Scope 记忆的可迁移范围，控制召回时的对话边界：
	//   "global"       跨对话依然成立的知识（事实/偏好/决定/FAQ）——任意对话可召回；
	//   "conversation" 仅原对话语境有效（针对特定对象的回应/私约/承诺）——只在记录时所在对话召回。
	// 空值视为 global（保守不丢知识）。分身回复按对话召回时据此过滤。
	Scope string `json:"scope,omitempty"`
}

// reflectConsolidated 执行记忆反射闭环：
//  1. 输入当前消息 + 已召回的相关记忆(memories) + 相关群知识(knowledge) + 对话上下文(context)
//  2. 用一个 LLM 调用把信息折叠成 Summary/Facts/Themes
//  3. 复用 evaluateRemember 判定"是否值得记 + 重要度档位"
//
// 返回 verdict.ShouldRemember=false 表示无需记（调用方跳过落库）。
// context 为最近几条对话消息（可选），帮助 LLM 理解"这句话在讨论什么"再判断是否值得记。
// 与 gracedb KnowledgeMemory 的"Recall→Reflect→Consolidate"一致，但知识召回那一路
// 由调用方传入自建 searchHybrid 的语义结果（避开 gracedb 内部已弃用的纯词法 SearchKnowledge）。
func reflectConsolidated(aiService *ai.AIService, message string, memories []string, knowledge []string, context []string) (MemoryReflection, RememberVerdict, error) {
	ref := MemoryReflection{}
	// 默认判定：沿用现有 evaluateRemember 的"值得记 + 重要度"
	verdict, err := evaluateRemember(aiService, rememberTaskPrompt(message, memories, knowledge, context), message)
	if err != nil {
		return ref, verdict, err
	}
	ref.Importance = verdict.Importance

	// 即便判定不值得记，也尽量产出 summary 供展示/日志；但 ShouldRemember=false 时调用方不落库
	ref.Summary = summarizeSnippets(message, memories, knowledge)
	if ref.Summary == "" {
		ref.Summary = truncateForSummary(message)
	}

	// 仅当判定值得记时才做结构化反射（产出 Summary/Facts/Themes/Entities 供知识图谱等使用）。
	// 不值得记就不浪费这次 LLM 调用；反射失败不阻断（保留上面的 deterministic summary 兜底）。
	if verdict.ShouldRemember && aiService != nil {
		if s, ok := reflectStructure(aiService, message, memories, knowledge, context); ok {
			if strings.TrimSpace(s.Summary) != "" {
				ref.Summary = s.Summary
			}
			ref.Facts = s.Facts
			ref.Themes = s.Themes
			ref.Entities = s.Entities
			// 反射出的可迁移范围（conversation/global）须带回 ref，否则落库时
			// knowledge_memory_scope 恒为空而按 global 处理，对话隔离功能形同虚设。
			ref.Scope = s.Scope
			ref.Type = s.Type
		}
	}
	return ref, verdict, nil
}

// reflectStructure 用一次 LLM 调用从消息+既有记忆/知识中提取结构化反射：
// summary（折叠后的记忆摘要）、facts、themes（主题）、entities（实体）。
// 返回 ok=false 表示 LLM 调用失败或未能解析出有效 summary（调用方保留降级 summary）。
func reflectStructure(aiService *ai.AIService, message string, memories []string, knowledge []string, context []string) (MemoryReflection, bool) {
	if aiService == nil {
		return MemoryReflection{}, false
	}
	aiMessages := []ai.Message{{Role: "user", Content: reflectionExtractPrompt(message, memories, knowledge, context)}}
	out, err := aiService.GetCompletion(ai.TaskTypeDigest, aiMessages)
	if err != nil {
		return MemoryReflection{}, false
	}
	return parseReflectionJSON(out)
}

// reflectionExtractPrompt 构造"结构化反射"的 LLM 提示：把消息+既有记忆折叠成带主题/实体的摘要。
// 与 rememberTaskPrompt（判定是否值得记）不同，这里专注产出可入图谱的结构（summary/facts/themes/entities）。
// context 为最近几条对话消息（可选），帮助 LLM 理解语境后提取更准确的结构化记忆。
func reflectionExtractPrompt(message string, memories []string, knowledge []string, context []string) string {
	var b strings.Builder
	b.WriteString("请把以下对话信息折叠合并成一条结构化记忆，提取关键实体与主题。\n")
	b.WriteString("仅返回 JSON，形如 {\"summary\":\"...\",\"facts\":[\"...\"],\"themes\":[\"...\"],\"entities\":[\"...\"],\"type\":\"fact\",\"scope\":\"global\"}\n")
	b.WriteString("- summary: 一段通顺的中文总结，合并重复信息。除结论外，尽量保留“为什么/背景/动机”，让后续回忆时能答出缘由而不仅是事实\n")
	b.WriteString("- facts: 明确的事实要点列表\n")
	b.WriteString("- themes: 2-5 个主题词（如“项目、偏好、约定”）\n")
	b.WriteString("- entities: 关键实体/人名/项目名（如“团队A、张三”）\n")
	b.WriteString("- type: 记忆类型。fact=客观事实（日期/数字/决定），preference=偏好/习惯/喜好，event=事件/约定/会议\n")
	b.WriteString("- scope: 记忆的可迁移范围。global=脱离本次对话依然成立的知识（事实、偏好、决定、FAQ）；conversation=仅本次对话语境有效（针对当前对话对象的回应、私约、承诺、临时安排），换到其他对话就失效或不该被引用。拿不准时偏向 global\n")
	b.WriteString("\nscope 示例：\n")
	b.WriteString("- “项目 Alpha 上线定在下周一”→ scope: global（跨对话依然成立的事实）\n")
	b.WriteString("- “和张三约好周五晚上一起吃饭”→ scope: conversation（仅本次对话的私约）\n")
	if len(context) > 0 {
		b.WriteString("\n对话上下文（最近几条消息，帮助理解语境）：\n")
		for i, c := range context {
			if i >= 5 {
				break
			}
			b.WriteString("- " + c + "\n")
		}
	}
	if len(memories) > 0 {
		b.WriteString("\n已有相关记忆（可能重复提及同一件事）：\n")
		for i, m := range memories {
			if i >= 5 {
				break
			}
			b.WriteString("- " + m + "\n")
		}
	}
	if len(knowledge) > 0 {
		b.WriteString("\n关联知识片段：\n")
		for i, k := range knowledge {
			if i >= 3 {
				break
			}
			b.WriteString("- " + k + "\n")
		}
	}
	b.WriteString("\n内容：\n" + message)
	return b.String()
}

// rememberTaskPrompt 构造纯分类提示：让 LLM 判断"是否值得记 + 重要度"。
// 不负责提取主题/事实（那是 reflectionExtractPrompt 的职责），只做二分类+档位。
// context 为最近几条对话消息（可选），帮助 LLM 理解"这句话在讨论什么"再判断是否值得记。
func rememberTaskPrompt(message string, memories []string, knowledge []string, context []string) string {
	var b strings.Builder
	b.WriteString("判断以下对话内容是否包含值得记忆的长期信息。\n")
	b.WriteString("值得记忆：个人偏好、重要决定、项目关键信息、约定事项、群内决定与共识、答疑形成的可复用知识（步骤、配置、口径、规范）。\n")
	b.WriteString(rememberVerdictNegativeClause + "\n")
	if len(context) > 0 {
		b.WriteString("\n对话上下文（最近几条消息，帮助理解当前消息的语境）：\n")
		for i, c := range context {
			if i >= 5 {
				break
			}
			b.WriteString("- " + c + "\n")
		}
	}
	if len(memories) > 0 {
		b.WriteString("\n用户/群已有相关记忆（可能重复提及，属于同一件事）：\n")
		for i, m := range memories {
			if i >= 5 {
				break
			}
			b.WriteString("- " + m + "\n")
		}
	}
	if len(knowledge) > 0 {
		b.WriteString("\n关联知识片段：\n")
		for i, k := range knowledge {
			if i >= 3 {
				break
			}
			b.WriteString("- " + k + "\n")
		}
	}
	b.WriteString("\n内容：\n" + message)
	return b.String()
}

// summarizeSnippets 在无 LLM 成功时退化为确定性汇总：合并去重后的记忆 + 当前消息。
// 仅供诊断/兜底使用，真实反射以 evaluateRemember 的 LLM 判定为主。
func summarizeSnippets(message string, memories []string, knowledge []string) string {
	seen := map[string]bool{}
	var parts []string
	for _, s := range append(append([]string{}, memories...), knowledge...) {
		s = strings.TrimSpace(s)
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		parts = append(parts, s)
	}
	if len(parts) == 0 {
		return message
	}
	return strings.Join(parts, "；")
}

func truncateForSummary(s string) string {
	r := []rune(s)
	if len(r) <= 120 {
		return s
	}
	return string(r[:120]) + "…"
}

var reflectionJSONRe = regexp.MustCompile(`\{[^{}]*\}`)

// parseReflectionJSON 解析 LLM 返回的结构化反射（summary/facts/themes），兼容不完整输出。
func parseReflectionJSON(s string) (MemoryReflection, bool) {
	block := reflectionJSONRe.FindString(s)
	if block == "" {
		return MemoryReflection{}, false
	}
	var raw MemoryReflection
	if err := json.Unmarshal([]byte(block), &raw); err != nil {
		return MemoryReflection{}, false
	}
	raw.Summary = strings.TrimSpace(raw.Summary)
	return raw, raw.Summary != ""
}
