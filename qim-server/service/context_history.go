package service

import (
	"time"

	"github.com/dshmyz/qim/qim-server/model"
)

// normalizedConversationHistory 是分身和群助手共用的历史消息规范。
// 统一在进入 prompt 前完成媒体过滤、当前消息排除、时间顺序、单条截断和 AI 回复裁剪。
type normalizedConversationHistory struct {
	Content     string
	SenderName  string
	IsAssistant bool
	CreatedAt   time.Time
}

func normalizeConversationHistory(messages []model.Message, currentUserID uint, originalContent string, maxRunes int, maxRecentAI int) []normalizedConversationHistory {
	filtered := make([]model.Message, 0, len(messages))
	for _, msg := range messages {
		if msg.Type != "text" && msg.Type != "markdown" {
			continue
		}
		if originalContent != "" && msg.SenderID == currentUserID && msg.Content == originalContent {
			continue
		}
		filtered = append(filtered, msg)
	}
	for i, j := 0, len(filtered)-1; i < j; i, j = i+1, j-1 {
		filtered[i], filtered[j] = filtered[j], filtered[i]
	}

	result := make([]normalizedConversationHistory, 0, len(filtered))
	keptAI := 0
	for _, msg := range filtered {
		isAssistant := msg.Origin == "assistant" || msg.Origin == "avatar"
		if isAssistant {
			if !isNearSelf(msg) || keptAI >= maxRecentAI {
				continue
			}
			keptAI++
		}
		name := msg.Sender.Nickname
		if name == "" {
			name = msg.Sender.Username
		}
		content := msg.Content
		if maxRunes > 0 {
			content = truncateRunes(content, maxRunes)
		}
		result = append(result, normalizedConversationHistory{
			Content: content, SenderName: name, IsAssistant: isAssistant, CreatedAt: msg.CreatedAt,
		})
	}
	return result
}
