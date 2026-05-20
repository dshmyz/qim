package service

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"qim-server/ai"
	"qim-server/model"
	"qim-server/pkg/logger"

	"gorm.io/gorm"
)

// AvatarTriggerService 智能触发判断服务
type AvatarTriggerService struct {
	aiService *ai.AIService
	db        *gorm.DB
}

// NewAvatarTriggerService 创建智能触发服务实例
func NewAvatarTriggerService(aiService *ai.AIService, db *gorm.DB) *AvatarTriggerService {
	return &AvatarTriggerService{
		aiService: aiService,
		db:        db,
	}
}

// ShouldReply 判断分身是否应该回复当前消息
func (s *AvatarTriggerService) ShouldReply(userID uint, conversationID uint, message string, senderName string) (bool, string, error) {
	// 1. 获取用户分身配置
	var config model.AvatarConfig
	if err := s.db.Where("user_id = ?", userID).First(&config).Error; err != nil {
		return false, "", fmt.Errorf("分身配置不存在")
	}

	if !config.Enabled {
		return false, "", nil
	}

	// 2. 检查是否在排除列表中
	if s.isExcluded(config, conversationID) {
		return false, "在排除列表中", nil
	}

	// 3. 解析触发规则
	var triggerRules model.AvatarTriggerRules
	if config.TriggerRulesJSON != "" {
		if err := json.Unmarshal([]byte(config.TriggerRulesJSON), &triggerRules); err != nil {
			logger.WithModule("AvatarTriggerService").Error("解析触发规则失败", "error", err)
		}
	}

	// 4. 检查关键词规则
	if s.matchKeywords(triggerRules, message) {
		return true, "关键词触发", nil
	}

	// 5. 检查时间范围
	if !s.inTimeRange(triggerRules) {
		return false, "不在活跃时间范围内", nil
	}

	// 6. LLM 智能判断
	if triggerRules.Mode == "smart" {
		return s.llmShouldReply(config, message, senderName)
	}

	// 默认模式：不自动回复
	return false, "非自动模式", nil
}

// matchKeywords 检查消息是否匹配关键词
func (s *AvatarTriggerService) matchKeywords(rules model.AvatarTriggerRules, message string) bool {
	if len(rules.Keywords) == 0 {
		return false
	}

	msgLower := strings.ToLower(message)
	for _, keyword := range rules.Keywords {
		if strings.Contains(msgLower, strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}

// inTimeRange 检查当前时间是否在活跃范围内
func (s *AvatarTriggerService) inTimeRange(rules model.AvatarTriggerRules) bool {
	if len(rules.TimeRanges) == 0 {
		return true // 没有时间限制则始终允许
	}

	now := time.Now()
	currentDay := int(now.Weekday())
	currentHour := now.Hour()

	for _, tr := range rules.TimeRanges {
		// 检查星期几
		dayMatch := false
		for _, day := range tr.DayOfWeek {
			if day == currentDay {
				dayMatch = true
				break
			}
		}
		if !dayMatch {
			continue
		}

		// 检查时间范围
		if currentHour >= tr.StartHour && currentHour <= tr.EndHour {
			return true
		}
	}

	return false
}

// isExcluded 检查会话是否在排除列表中
func (s *AvatarTriggerService) isExcluded(config model.AvatarConfig, conversationID uint) bool {
	var rules model.AvatarTriggerRules
	if config.TriggerRulesJSON != "" {
		if err := json.Unmarshal([]byte(config.TriggerRulesJSON), &rules); err == nil {
			for _, excludedID := range rules.ExcludedConversations {
				if excludedID == conversationID {
					return true
				}
			}
		}
	}
	return false
}

// llmShouldReply 使用 LLM 智能判断是否应该回复
func (s *AvatarTriggerService) llmShouldReply(config model.AvatarConfig, message, senderName string) (bool, string, error) {
	prompt := fmt.Sprintf(`你是%s的AI分身。判断以下群消息是否需要你代表%s回复。

考虑因素：
1. 消息是否向你（或你代表的用户）提问？
2. 消息内容是否在你的专业领域内？
3. 是否是重要的讨论需要你参与？
4. 是否只是普通闲聊不需要回复？

只返回 JSON：{"should_reply": true/false, "reason": "原因"}

消息：%s
发送者：%s`, config.Name, config.Name, message, senderName)

	aiMessages := []ai.Message{{Role: "user", Content: prompt}}
	result, err := s.aiService.GetCompletion(ai.TaskTypeChat, aiMessages)
	if err != nil {
		logger.WithModule("AvatarTriggerService").Error("LLM判断失败", "error", err)
		return false, "", err
	}

	// 解析 JSON 返回结果
	var response struct {
		ShouldReply bool   `json:"should_reply"`
		Reason      string `json:"reason"`
	}

	if err := json.Unmarshal([]byte(result), &response); err != nil {
		logger.WithModule("AvatarTriggerService").Error("解析LLM返回失败", "error", err, "raw", result)
		return false, "解析失败", nil
	}

	return response.ShouldReply, response.Reason, nil
}