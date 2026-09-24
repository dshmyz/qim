package service

import (
	"time"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/logger"

	"gorm.io/gorm"
)

// AI 回复场景标识。
const (
	ReplySceneBotDM        = "bot_dm"
	ReplySceneGroupMention = "group_mention"
	ReplySceneGroupAuto    = "group_auto"
	ReplySceneAvatar       = "avatar"
)

// 中断原因标识。
const (
	InterruptReasonLength  = "length"  // 触达 max_tokens
	InterruptReasonNoDone  = "no_done" // 上游无 [DONE] 静默断连
	InterruptReasonError   = "error"
	InterruptReasonTimeout = "timeout"
)

// AIReplyEventService 回复终态事件的写入。
type AIReplyEventService struct {
	db *gorm.DB
}

func NewAIReplyEventService(db *gorm.DB) *AIReplyEventService {
	return &AIReplyEventService{db: db}
}

// RecordCompleted 记录一次正常完成的回复。
func (s *AIReplyEventService) RecordCompleted(scene string, actorID, convID, messageID uint) {
	s.record(scene, actorID, convID, messageID, "completed", "")
}

// RecordInterrupted 记录一次中断的回复及原因。
func (s *AIReplyEventService) RecordInterrupted(scene string, actorID, convID, messageID uint, reason string) {
	s.record(scene, actorID, convID, messageID, "interrupted", reason)
}

// RecordNotTriggered 记录触发门判断无需回复（智能拦截），不计为中断。
func (s *AIReplyEventService) RecordNotTriggered(scene string, convID uint) {
	s.record(scene, 0, convID, 0, "not_triggered", "")
}

func (s *AIReplyEventService) record(scene string, actorID, convID, messageID uint, outcome, reason string) {
	if s == nil || s.db == nil || convID == 0 {
		return
	}
	if outcome == "interrupted" && reason == "" {
		reason = InterruptReasonError
	}
	event := model.AIReplyEvent{
		Scene:          scene,
		ActorID:        actorID,
		ConversationID: convID,
		MessageID:      messageID,
		Outcome:        outcome,
		Reason:         reason,
		CreatedAt:      time.Now(),
	}
	if err := s.db.Create(&event).Error; err != nil {
		// 终态事件失败不影响主流程，仅日志
		logger.WithModule("AIReplyEvent").Warn("回复终态事件落库失败", "error", err)
	}
}
