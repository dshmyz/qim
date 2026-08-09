package ai

import "strings"

// StaticCapability 恒定可用的分析/问答能力（与具体工具解耦，系统始终具备）。
// 用于 AI 被问「具备哪些能力」时如实自述，避免只靠模型通用知识瞎猜。
type StaticCapability struct {
	Name string
	Desc string
}

// StaticCapabilities 系统恒定能力清单。name 描述能力，desc 说明用途。
// 注意：此处只放「无需调用工具也能提供的」纯分析/问答能力；
// 需要工具的实时检索/待办/发消息等能力一律由 ToolRegistry.BuildCapabilityPrompt 的动态工具段注入，
// 避免在无工具路径（私人对话）声称系统不真正具备的检索/执行能力。
var StaticCapabilities = []StaticCapability{
	{"智能问答", "基于已注入的上下文与你的知识回答问题"},
	{"对话总结", "总结会话或群聊的讨论要点并提炼结论"},
	{"文本分析", "翻译、改写、润色文本"},
	{"内容整理", "提炼要点、整理待办清单"},
}

// BuildStaticCapabilitiesPrompt 仅返回静态能力自述段（不含工具段）。
// 供无工具（或无法访问 ToolRegistry）的入口（Bot 对话、legacy 提示词降级路径）复用，
// 保证各处静态能力文案单一来源。
func BuildStaticCapabilitiesPrompt() string {
	if len(StaticCapabilities) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("你具备以下能力：\n")
	for _, c := range StaticCapabilities {
		sb.WriteString("- " + c.Name + "：" + c.Desc + "\n")
	}
	return sb.String()
}

// BuildCapabilityPrompt 依据实际 allowed 工具集 + 静态能力，生成一段「能力自述」注入系统提示词。
// 目标：让 AI 能如实列出自己当前真正具备的能力，而非靠通用知识泛泛而谈。
//
//   - 静态能力段恒定注入（系统始终具备）。
//   - 工具段只照实列出 allowed 范围内、且当前 enabled 的工具（name + 一句描述，取自工具 Description()），
//     allowed 为空则工具段为空——保证「AI 说的永远 = 它当时真能用的」。
//
// 返回空串表示既无静态能力也无工具（安全降级，调用方不注入，行为不变）。
func (r *ToolRegistry) BuildCapabilityPrompt(allowed []string) string {
	var sb strings.Builder

	// 静态能力段
	if static := BuildStaticCapabilitiesPrompt(); static != "" {
		sb.WriteString(static)
	}

	// 动态工具段：只列 allowed 范围内且启用中的工具
	if r != nil && len(allowed) > 0 {
		allowedSet := make(map[string]bool, len(allowed))
		for _, a := range allowed {
			allowedSet[a] = true
		}
		var toolLines []string
		for _, tool := range r.ListTools() {
			name, _ := tool["name"].(string)
			if name == "" || !allowedSet[name] {
				continue
			}
			enabled, _ := tool["enabled"].(bool)
			if !enabled {
				continue
			}
			desc, _ := tool["description"].(string)
			if desc == "" {
				desc = "系统工具"
			}
			toolLines = append(toolLines, "- "+name+"："+desc)
		}
		if len(toolLines) > 0 {
			if sb.Len() > 0 {
				sb.WriteString("\n")
			}
			sb.WriteString("此外，你还可以调用以下工具执行对应操作：\n")
			sb.WriteString(strings.Join(toolLines, "\n") + "\n")
		}
	}

	return sb.String()
}
