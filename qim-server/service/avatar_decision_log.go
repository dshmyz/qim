package service

import (
	"time"

	"github.com/dshmyz/qim/qim-server/model"

	"gorm.io/gorm"
)

// avatarDecisionRetention 每主人最多保留的决策事件数，超出部分在写入时顺带清除。
const avatarDecisionRetention = 500

// AvatarDecisionLogService 分身决策留痕的写入与查询。触发侧（SmartReplyEngine）
// 与 worker 侧（AvatarWorkerPool）各自持有无状态实例，共用同一张表。
type AvatarDecisionLogService struct {
	db *gorm.DB
}

func NewAvatarDecisionLogService(db *gorm.DB) *AvatarDecisionLogService {
	return &AvatarDecisionLogService{db: db}
}

// AvatarDecisionInput 一次决策事件的写入内容。
type AvatarDecisionInput struct {
	OwnerID          uint
	ConversationID   uint
	TriggerMessageID uint
	TriggerUserID    uint
	TriggerName      string
	MessageSnippet   string
	Decision         string // replied / skipped
	Stage            string
	Reason           string
}

// Record 写入一条决策事件并按主人做数量裁剪。失败只影响可观测性，不干预回复流程，
// 故错误仅由调用方日志感知，不向上抛出。
func (s *AvatarDecisionLogService) Record(in AvatarDecisionInput) error {
	if s == nil || s.db == nil || in.OwnerID == 0 {
		return nil
	}

	event := model.AvatarDecisionEvent{
		OwnerID:          in.OwnerID,
		ConversationID:   in.ConversationID,
		TriggerMessageID: in.TriggerMessageID,
		TriggerUserID:    in.TriggerUserID,
		TriggerName:      truncateRunes(in.TriggerName, 63),    // 含追加省略号后 ≤64
		MessageSnippet:   truncateRunes(in.MessageSnippet, 99), // 含追加省略号后 ≤100
		Decision:         in.Decision,
		Stage:            in.Stage,
		Reason:           truncateRunes(in.Reason, 254), // 含追加省略号后 ≤255
		CreatedAt:        time.Now(),
	}
	if err := s.db.Create(&event).Error; err != nil {
		return err
	}

	s.prune(in.OwnerID)
	return nil
}

// prune 保留该主人最近 avatarDecisionRetention 条，删除更旧的事件。
func (s *AvatarDecisionLogService) prune(ownerID uint) {
	var keepIDs []uint
	if err := s.db.Model(&model.AvatarDecisionEvent{}).
		Where("owner_id = ?", ownerID).
		Order("created_at DESC, id DESC").
		Limit(avatarDecisionRetention).
		Pluck("id", &keepIDs).Error; err != nil {
		return
	}
	if len(keepIDs) < avatarDecisionRetention {
		return
	}
	s.db.Where("owner_id = ? AND id NOT IN ?", ownerID, keepIDs).
		Delete(&model.AvatarDecisionEvent{})
}

// List 查询主人的决策事件（conversationID=0 表示全部会话），按时间倒序。
func (s *AvatarDecisionLogService) List(ownerID, conversationID uint, limit int) []model.AvatarDecisionEvent {
	if s == nil || s.db == nil || ownerID == 0 {
		return nil
	}
	if limit <= 0 || limit > 200 {
		limit = 100
	}

	query := s.db.Where("owner_id = ?", ownerID)
	if conversationID != 0 {
		query = query.Where("conversation_id = ?", conversationID)
	}

	var events []model.AvatarDecisionEvent
	query.Order("created_at DESC, id DESC").Limit(limit).Find(&events)
	return events
}
