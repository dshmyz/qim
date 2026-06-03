package ai

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/dshmyz/qim/qim-server/pkg/logger"
)

// MessageIntent 表示消息的意图分类
type MessageIntent struct {
	Type       string                 `json:"type"` // "chat", "command", "query", "alert", "todo"
	Confidence float32                `json:"confidence"`
	Action     string                 `json:"action,omitempty"`
	Entities   map[string]interface{} `json:"entities,omitempty"`
}

// IntentDetector 意图检测器
type IntentDetector struct {
	aiService *AIService

	// 预定义的模式匹配规则
	patterns map[string][]*regexp.Regexp
}

// NewIntentDetector 创建意图检测器
func NewIntentDetector(aiService *AIService) *IntentDetector {
	detector := &IntentDetector{
		aiService: aiService,
		patterns:  make(map[string][]*regexp.Regexp),
	}

	detector.initPatterns()
	return detector
}

// initPatterns 初始化规则匹配模式
func (d *IntentDetector) initPatterns() {
	// 管理命令模式
	d.patterns["command"] = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(移除|删除|踢出|添加|邀请|禁言|解封|设置管理员|取消管理员)`),
		regexp.MustCompile(`(?i)(create|delete|remove|add|ban|unban|promote|demote)`),
	}

	// 问题咨询模式
	d.patterns["query"] = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(怎么|如何|什么|为什么|哪里|谁|什么时候|多少)`),
		regexp.MustCompile(`(?i)(how|what|why|where|when|who|can you|could you)`),
	}

	// 告警/异常模式
	d.patterns["alert"] = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(挂了|崩溃|错误|异常|故障|宕机|无法访问|连不上)`),
		regexp.MustCompile(`(?i)(down|crash|error|fail|broken|unreachable|urgent|critical)`),
	}

	// 待办提取模式
	d.patterns["todo"] = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(记得|要|需要|安排|计划|待办|任务|明天|下周)`),
		regexp.MustCompile(`(?i)(remember to|need to|schedule|plan|todo|task|deadline)`),
	}
}

// Detect 检测消息意图
func (d *IntentDetector) Detect(content string, userID uint, conversationID uint) (*MessageIntent, error) {
	// 先尝试规则匹配
	if intent := d.detectByRules(content); intent != nil {
		return intent, nil
	}

	// 规则匹配失败时，使用 AI 进行语义理解
	if d.aiService != nil && d.aiService.IsConfigured() {
		return d.detectByAI(content, userID, conversationID)
	}

	// 默认返回普通聊天
	return &MessageIntent{
		Type:       "chat",
		Confidence: 0.5,
	}, nil
}

// detectByRules 基于规则检测意图（按优先级）
func (d *IntentDetector) detectByRules(content string) *MessageIntent {
	logger.WithModule("IntentDetector").Debug("规则检测开始", "content", content[:min(30, len(content))])

	// command 优先级最高（管理操作）
	commandPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(移除|删除|踢出|添加|邀请|禁言|解封|设置管理员|取消管理员)`),
		regexp.MustCompile(`(?i)(create|delete|remove|add|ban|unban|promote|demote)`),
	}
	for _, pattern := range commandPatterns {
		if pattern.MatchString(content) {
			logger.WithModule("IntentDetector").Debug("匹配到 command 模式", "pattern", pattern.String())
			entities := d.extractEntities(content, "command")
			return &MessageIntent{
				Type:       "command",
				Confidence: 0.8,
				Entities:   entities,
			}
		}
	}

	// alert 次之（告警）
	alertPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(挂了|崩溃|错误|异常|故障|宕机|无法访问|连不上)`),
		regexp.MustCompile(`(?i)(down|crash|error|fail|broken|unreachable|urgent|critical)`),
	}
	for _, pattern := range alertPatterns {
		if pattern.MatchString(content) {
			entities := d.extractEntities(content, "alert")
			return &MessageIntent{
				Type:       "alert",
				Confidence: 0.7,
				Entities:   entities,
			}
		}
	}

	// query 再次（问题咨询）
	queryPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(怎么|如何|什么|为什么|哪里|谁|什么时候|多少)`),
		regexp.MustCompile(`(?i)(how|what|why|where|when|who|can you|could you)`),
	}
	for _, pattern := range queryPatterns {
		if pattern.MatchString(content) {
			entities := d.extractEntities(content, "query")
			return &MessageIntent{
				Type:       "query",
				Confidence: 0.6,
				Entities:   entities,
			}
		}
	}

	// todo 最后（待办）
	todoPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(记得|要|需要|安排|计划|待办|任务|明天|下周)`),
		regexp.MustCompile(`(?i)(remember to|need to|schedule|plan|todo|task|deadline)`),
	}
	for _, pattern := range todoPatterns {
		if pattern.MatchString(content) {
			entities := d.extractEntities(content, "todo")
			return &MessageIntent{
				Type:       "todo",
				Confidence: 0.6,
				Entities:   entities,
			}
		}
	}

	logger.WithModule("IntentDetector").Debug("规则检测未匹配")
	return nil
}

// detectByAI 使用 AI 检测意图
func (d *IntentDetector) detectByAI(content string, userID uint, conversationID uint) (*MessageIntent, error) {
	systemPrompt := `你是一个意图识别助手。分析用户消息的意图，并按 JSON 格式返回结果。

意图类型：
- "chat": 普通聊天/闲聊
- "command": 管理指令（如"把某人移出群"、"设置管理员"）
- "query": 问题咨询（询问如何做某事、寻求信息）
- "alert": 告警/异常报告（系统故障、错误等）
- "todo": 待办事项/任务安排

只返回 JSON，格式：{"type": "意图类型", "confidence": 0.0-1.0, "action": "具体动作", "entities": {"key": "value"}}`

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

	// 各类意图的触发阈值
	switch intent.Type {
	case "query":
		if intent.Confidence >= 0.5 {
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
		if intent.Confidence >= 0.6 {
			logger.WithModule("IntentDetector").Info("待办事项，触发回复", "confidence", intent.Confidence)
			return true
		}
	}

	logger.WithModule("IntentDetector").Debug("不触发回复", "type", intent.Type, "confidence", intent.Confidence)
	return false
}