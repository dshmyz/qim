package service

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/logger"

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

// DecideReply 是分身触发决策的唯一入口。
// 排除列表 / 时间窗 在所有模式下统一生效；mode 分发（mention/offline/keyword/all/smart）
// 全部收敛到这里，供实时路径 shouldTriggerAvatar 与预览接口 CheckTrigger 共用，
// 避免两套判断逻辑各自为政。
func (s *AvatarTriggerService) DecideReply(config model.AvatarConfig, conversationID uint, message string, senderName string, isGroupChat bool, mentionUserIDs []uint) (bool, string, error) {
	if !config.Enabled {
		return false, "分身未启用", nil
	}

	var rules model.AvatarTriggerRules
	if config.TriggerRulesJSON != "" {
		if err := json.Unmarshal([]byte(config.TriggerRulesJSON), &rules); err != nil {
			logger.WithModule("AvatarTriggerService").Error("解析触发规则失败", "error", err)
			return false, "解析触发规则失败", nil
		}
	}

	if IsAvatarExcluded(rules, conversationID) {
		return false, "在排除列表中", nil
	}

	if !IsAvatarInTimeRange(rules) {
		return false, "不在活跃时间范围内", nil
	}

	effectiveMode := rules.Mode
	if effectiveMode == "" {
		effectiveMode = "mention"
	}

	// 私聊中 mention 模式自动触发
	if !isGroupChat && effectiveMode == "mention" {
		return true, "私聊 mention 自动触发", nil
	}

	switch effectiveMode {
	case "mention":
		for _, uid := range mentionUserIDs {
			if uid == config.UserID {
				return true, "mention 触发", nil
			}
		}
		return false, "未被 mention", nil
	case "offline":
		var user model.User
		if err := s.db.First(&user, config.UserID).Error; err != nil {
			return false, "查找用户失败", err
		}
		if user.Status == "offline" {
			return true, "离线触发", nil
		}
		return false, "用户在线", nil
	case "keyword":
		if len(rules.Keywords) == 0 {
			return true, "keyword 无关键词默认触发", nil
		}
		// 大小写不敏感匹配，与旧 matchKeywords 行为一致
		msgLower := strings.ToLower(message)
		for _, kw := range rules.Keywords {
			if strings.Contains(msgLower, strings.ToLower(kw)) {
				return true, "关键词触发", nil
			}
		}
		return false, "未命中关键词", nil
	case "all":
		return true, "all 模式", nil
	case "smart":
		shouldReply, confidence, reason, err := s.LLMShouldReply(config, message, senderName)
		if err != nil {
			return false, "", err
		}
		// 置信度门控：阈值 > 0 时，即使 should_reply=true，confidence 低于阈值也降级不回复（fail-closed）
		var replyStrategy model.AvatarReplyStrategy
		if config.ReplyStrategyJSON != "" {
			_ = json.Unmarshal([]byte(config.ReplyStrategyJSON), &replyStrategy)
		}
		if shouldReply && replyStrategy.ConfidenceThreshold > 0 && confidence < replyStrategy.ConfidenceThreshold {
			return false, fmt.Sprintf("置信度 %.2f 低于阈值 %.2f", confidence, replyStrategy.ConfidenceThreshold), nil
		}
		return shouldReply, reason, nil
	default:
		return isGroupChat && len(mentionUserIDs) > 0, "未知模式按 mention 处理", nil
	}
}

// LLMShouldReply 仅负责 smart 模式的 LLM 意图判断，由 DecideReply 内部调用。
// 返回 (shouldReply, confidence 0-1, reason, err)，confidence 由 LLM 自评。
func (s *AvatarTriggerService) LLMShouldReply(config model.AvatarConfig, message string, senderName string) (bool, float64, string, error) {
	prompt := fmt.Sprintf(`你是%s的AI分身。判断以下群消息是否需要你代表%s回复。

考虑因素：
1. 消息是否向你（或你代表的用户）提问？
2. 消息内容是否在你的专业领域内？
3. 是否是重要的讨论需要你参与？
4. 是否只是普通闲聊不需要回复？

只返回 JSON：{"should_reply": true/false, "confidence": 0.0-1.0, "reason": "原因"}
confidence 为你对此判断的把握程度。

消息：%s
发送者：%s`, config.Name, config.Name, message, senderName)

	aiMessages := []ai.Message{{Role: "user", Content: prompt}}
	result, err := s.aiService.GetCompletion(ai.TaskTypeChat, aiMessages)
	if err != nil {
		logger.WithModule("AvatarTriggerService").Error("LLM判断失败", "error", err)
		return false, 0, "", err
	}

	// 解析 JSON 返回结果。LLM 可能用 markdown 代码块或前言包裹 JSON，先尝试原文，再尝试抽取 {...} 子串
	var response struct {
		ShouldReply bool    `json:"should_reply"`
		Confidence  float64 `json:"confidence"`
		Reason      string  `json:"reason"`
	}

	raw := strings.TrimSpace(result)
	if err := json.Unmarshal([]byte(raw), &response); err != nil {
		if sub := extractJSONObject(raw); sub != "" {
			if err2 := json.Unmarshal([]byte(sub), &response); err2 == nil {
				return response.ShouldReply, response.Confidence, response.Reason, nil
			}
		}
		// 解析彻底失败：fail-closed，与 DecideReply 其余门一致（避免 LLM 降级时刷屏）
		logger.WithModule("AvatarTriggerService").Error("解析LLM返回失败，静默跳过", "error", err, "raw", result)
		return false, 0, "LLM返回解析失败，静默跳过", nil
	}

	return response.ShouldReply, response.Confidence, response.Reason, nil
}

// extractJSONObject 从可能带 markdown 围栏或前言的文本中抽取第一个 JSON 对象子串
func extractJSONObject(s string) string {
	start := strings.IndexByte(s, '{')
	if start < 0 {
		return ""
	}
	end := strings.LastIndexByte(s, '}')
	if end <= start {
		return ""
	}
	return s[start : end+1]
}

// IsAvatarExcluded 判断会话是否在分身的排除列表中。
// 由 DecideReply 在所有触发模式下统一调用。
func IsAvatarExcluded(rules model.AvatarTriggerRules, conversationID uint) bool {
	for _, excludedID := range rules.ExcludedConversations {
		if excludedID == conversationID {
			return true
		}
	}
	return false
}

// IsAvatarInTimeRange 判断当前时间是否在分身的活跃范围内。
// 未配置任何时间范围时视为无限制（始终允许）。
func IsAvatarInTimeRange(rules model.AvatarTriggerRules) bool {
	if len(rules.TimeRanges) == 0 {
		return true
	}

	now := time.Now()
	currentDay := int(now.Weekday())
	currentHour := now.Hour()

	for _, tr := range rules.TimeRanges {
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
		if currentHour >= tr.StartHour && currentHour <= tr.EndHour {
			return true
		}
	}

	return false
}
