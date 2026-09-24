package ai

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/dshmyz/qim/qim-server/pkg/aiprompt"
	"github.com/dshmyz/qim/qim-server/pkg/logger"
)

// MessageIntent 表示消息的意图分类
type MessageIntent struct {
	Type        string                 `json:"type"` // "chat", "command", "query", "alert", "todo"
	Confidence  float32                `json:"confidence"`
	Action      string                 `json:"action,omitempty"`
	Entities    map[string]interface{} `json:"entities,omitempty"`
	ShouldReply *bool                  `json:"should_reply,omitempty"`
	Target      string                 `json:"target,omitempty"`
	Reason      string                 `json:"reason,omitempty"`
}

// IntentDetector 意图检测器
type IntentDetector struct {
	aiService *AIService

	// 预定义的模式匹配规则。strong=锚定/报告式措辞，命中即高置信触发；
	// weak=裸词/子串命中（如陈述句里的 "added"/"错误"），仅低置信提示，
	// 不直接触发自动回复——AI 可用时走语义判断，不可用则 fail-closed 不回。
	patterns map[string][]regexTier
}

type regexTier struct {
	pattern *regexp.Regexp
	strong  bool
}

// NewIntentDetector 创建意图检测器
func NewIntentDetector(aiService *AIService) *IntentDetector {
	detector := &IntentDetector{
		aiService: aiService,
		patterns:  make(map[string][]regexTier),
	}

	detector.initPatterns()
	return detector
}

// initPatterns 初始化规则匹配模式
func (d *IntentDetector) initPatterns() {
	// 管理命令模式
	d.patterns["command"] = []regexTier{
		// strong：中文锚定的祈使结构
		{regexp.MustCompile(`(?i)^\s*(请|帮我|麻烦|请你).*?(移除|移出|删除|踢出|添加|邀请|禁言|解封|设置管理员|取消管理员)`), true},
		{regexp.MustCompile(`(?i)^\s*(把|将)\s*.+?(移除|移出|删除|踢出|添加|邀请|禁言|解封|设置管理员|取消管理员)`), true},
		{regexp.MustCompile(`(?i)^\s*(移除|移出|删除|踢出|添加|邀请|禁言|解封|设置管理员|取消管理员)`), true},
		// strong：英文锚定的祈使动词（可带 please）
		{regexp.MustCompile(`(?i)^\s*(please\s+)?(create|delete|remove|add|ban|unban|promote|demote)\b`), true},
		// weak：动词在句中任意位置出现（陈述句 "I added a comment"）
		{regexp.MustCompile(`(?i)\b(add|adds|added|adding|ban|bans|banned|banning|create|creates|created|creating|delete|deletes|deleted|deleting|remove|removes|removed|removing|unban|promote|promotes|promoted|demote|demotes|demoted)\b`), false},
	}

	// 问题咨询模式
	d.patterns["query"] = []regexTier{
		// strong：疑问词（「什么」单列 weak：「没什么意见」式陈述常见）
		{regexp.MustCompile(`(?i)(怎么|如何|为什么|哪里|哪儿|谁|什么时候|为何|多少|几号|能不能|可不可以)`), true},
		{regexp.MustCompile(`什么`), false},
		{regexp.MustCompile(`(?i)\b(how|why|where|when|who|whom|whose|which)\b|\bcan\s+you\b|\bcould\s+you\b|\bwhat\b`), true},
	}

	// 告警/异常模式
	d.patterns["alert"] = []regexTier{
		// strong：本身即告警报告的谓词/措辞
		{regexp.MustCompile(`(?i)(挂了|宕机|连不上|无法访问|打不开|崩了|崩溃了|出故障了|报错了|出错了|出问题了)`), true},
		{regexp.MustCompile(`(?i)\b(is|are|was|were)\s+(down|offline|unreachable|broken)\b|\bcrashed\b|\burgent\b|\bcritical\b|\bunreachable\b`), true},
		// weak：裸名词（"错误已经改完"）
		{regexp.MustCompile(`(?i)(错误|异常|故障|报错|出错)`), false},
		{regexp.MustCompile(`(?i)\b(down|crash|error|fail|failed|failure|broken|bug)\b`), false},
	}

	// 待办提取模式
	d.patterns["todo"] = []regexTier{
		// strong：锚定的祈使/提醒措辞
		{regexp.MustCompile(`(?i)(请帮我|帮我|提醒我|记得提醒|设置.*提醒|创建.*任务)`), true},
		{regexp.MustCompile(`(?i)\b(remind me|schedule a|create a task)\b`), true},
		// weak：句中任意位置的安排/计划词
		{regexp.MustCompile(`(?i)(安排|截止日期)`), false},
		{regexp.MustCompile(`(?i)\b(remember to|need to|schedule|plan|todo|task|deadline)\b`), false},
	}
}

// Detect 检测消息意图。强模式命中直接返回；弱命中或无规则时，AI 可用走语义判断，
// AI 不可用时弱意图置信度低于触发阈值（fail-closed 不自动回）。
func (d *IntentDetector) Detect(content string, userID uint, conversationID uint) (*MessageIntent, error) {
	intent, strong := d.detectByRules(content)
	if strong {
		return intent, nil
	}

	if d.aiService != nil && d.aiService.IsConfigured() {
		return d.detectByAI(content, userID, conversationID)
	}

	if intent != nil {
		return intent, nil
	}

	// 默认返回普通聊天
	return &MessageIntent{
		Type:       "chat",
		Confidence: 0.5,
	}, nil
}

// detectByRules 基于规则检测意图。返回 (意图, 是否强命中)：
// 强命中按 command→alert→query→todo 优先级返回高置信意图；仅弱命中时
// 返回置信度 0.45 的低置信意图（不直接触发）；全无匹配返回 nil。
func (d *IntentDetector) detectByRules(content string) (*MessageIntent, bool) {
	logger.WithModule("IntentDetector").Debug("规则检测开始", "content", content[:min(30, len(content))])

	priority := []string{"command", "alert", "query", "todo"}
	confidence := map[string]float32{
		"command": 0.85,
		"alert":   0.8,
		"query":   0.8,
		"todo":    0.8,
	}

	var weakMatch *MessageIntent
	for _, intentType := range priority {
		for _, tier := range d.patterns[intentType] {
			if !tier.pattern.MatchString(content) {
				continue
			}
			if tier.strong {
				logger.WithModule("IntentDetector").Debug("匹配到强模式", "type", intentType, "pattern", tier.pattern.String())
				return &MessageIntent{
					Type:        intentType,
					Confidence:  confidence[intentType],
					ShouldReply: boolPtr(true),
					Entities:    d.extractEntities(content, intentType),
				}, true
			}
			if weakMatch == nil {
				weakMatch = &MessageIntent{
					Type:       intentType,
					Confidence: 0.45,
				}
			}
		}
	}

	if weakMatch != nil {
		logger.WithModule("IntentDetector").Debug("仅弱模式命中", "type", weakMatch.Type)
	} else {
		logger.WithModule("IntentDetector").Debug("规则检测未匹配")
	}
	return weakMatch, false
}

// detectByAI 使用 AI 检测意图
func (d *IntentDetector) detectByAI(content string, userID uint, conversationID uint) (*MessageIntent, error) {
	systemPrompt := aiprompt.CurrentTimeLine() + "\n\n" + `你是一个意图识别助手。分析用户消息的意图，并按 JSON 格式返回结果。

	意图类型：
- "chat": 普通聊天/闲聊
- "command": 管理指令（如"把某人移出群"、"设置管理员"）
- "query": 问题咨询（询问如何做某事、寻求信息）
- "alert": 告警/异常报告（系统故障、错误等）
- "todo": 待办事项/任务安排

	只返回 JSON，格式：{"type": "意图类型", "confidence": 0.0-1.0, "should_reply": true或false, "target": "ai或human_group", "reason": "判断理由", "action": "具体动作", "entities": {"key": "value"}}

should_reply 规则：只有用户明确提问、请求 AI 分析/执行、报告需要处理的故障时为 true；普通陈述、群成员之间闲聊、没有明确对象的安排信息为 false。拿不准时为 false。`

	messages := []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: content},
	}

	result, err := d.aiService.GetCompletion(TaskTypeIntent, messages)
	if err != nil {
		logger.WithModule("IntentDetector").Error("AI 意图检测失败", "error", err)
		return nil, err
	}

	return d.parseAIResult(result)
}

// parseAIResult 解析 AI 返回的意图结果
func (d *IntentDetector) parseAIResult(result string) (*MessageIntent, error) {
	// 尝试从结果中提取 JSON
	jsonStr := result
	if idx := strings.Index(result, "{"); idx >= 0 {
		jsonStr = result[idx:]
		if endIdx := strings.LastIndex(jsonStr, "}"); endIdx >= 0 {
			jsonStr = jsonStr[:endIdx+1]
		}
	}

	var intent MessageIntent
	if err := json.Unmarshal([]byte(jsonStr), &intent); err != nil {
		return nil, fmt.Errorf("解析 AI 意图结果失败: %w", err)
	}

	if intent.Type == "" {
		intent.Type = "chat"
	}
	if intent.Confidence == 0 {
		intent.Confidence = 0.6
	}
	// 兼容旧模型未返回 should_reply 的情况：非 chat 意图继续沿用旧阈值逻辑；
	// 一旦模型明确返回 false，必须尊重它，避免规则外的普通陈述误触发。
	if intent.ShouldReply == nil && intent.Type != "chat" {
		intent.ShouldReply = boolPtr(true)
	}

	return &intent, nil
}

// extractEntities 从消息中提取实体
func (d *IntentDetector) extractEntities(content string, intentType string) map[string]interface{} {
	entities := make(map[string]interface{})

	switch intentType {
	case "command":
		// 提取可能的用户名和群组名
		namePattern := regexp.MustCompile(`[@「"](.*?)[@」"]`)
		if matches := namePattern.FindAllStringSubmatch(content, -1); len(matches) > 0 {
			entities["target_name"] = matches[0][1]
		}

	case "todo":
		// 提取时间信息
		timePatterns := []string{`明天`, `下周`, `后天`, `\d+月\d+日`, `\d+:\d+`}
		for _, p := range timePatterns {
			if matched, _ := regexp.MatchString(p, content); matched {
				entities["has_time"] = true
				break
			}
		}
	}

	return entities
}

// ShouldTriggerAIReply 判断是否应该触发 AI 自动回复
func (d *IntentDetector) ShouldTriggerAIReply(intent *MessageIntent, conversationType string) bool {
	logger.WithModule("IntentDetector").Debug("ShouldTriggerAIReply", "type", intent.Type, "confidence", intent.Confidence, "convType", conversationType)

	// 机器人会话始终回复
	if conversationType == "bot" {
		logger.WithModule("IntentDetector").Debug("机器人会话，始终回复")
		return true
	}
	if intent.ShouldReply != nil && !*intent.ShouldReply {
		logger.WithModule("IntentDetector").Debug("AI 意图明确判断无需回复", "reason", intent.Reason, "target", intent.Target)
		return false
	}

	// 各类意图的触发阈值
	switch intent.Type {
	case "query":
		if intent.Confidence >= 0.75 {
			logger.WithModule("IntentDetector").Info("问题咨询，触发回复", "confidence", intent.Confidence)
			return true
		}
	case "alert":
		if intent.Confidence >= 0.7 {
			logger.WithModule("IntentDetector").Info("告警/异常，触发回复", "confidence", intent.Confidence)
			return true
		}
	case "command":
		if intent.Confidence >= 0.8 {
			logger.WithModule("IntentDetector").Info("管理指令，触发回复", "confidence", intent.Confidence)
			return true
		}
	case "todo":
		if intent.Confidence >= 0.75 {
			logger.WithModule("IntentDetector").Info("待办事项，触发回复", "confidence", intent.Confidence)
			return true
		}
	}

	logger.WithModule("IntentDetector").Debug("不触发回复", "type", intent.Type, "confidence", intent.Confidence)
	return false
}

func boolPtr(v bool) *bool { return &v }
