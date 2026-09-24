package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/logger"

	"gorm.io/gorm"
)

// QualityRejectedError 分身回复质量核验两轮（初验 → 改进重试 → 再验）均未通过时返回。
// 携带草稿正文与审核原因，供 worker 落库为待审批草稿，而非像旧路径那样丢弃草稿只发 toast。
type QualityRejectedError struct {
	Draft  string
	Reason string
}

func (e *QualityRejectedError) Error() string {
	return "reply quality rejected: " + e.Reason
}

// AvatarReplyDraftService 待审批草稿的持久化与通知中心留痕。不负责发送：批准发送需
// worker pool 的让位门复验与 sendReply，由池在加载本服务后编排。
type AvatarReplyDraftService struct {
	db *gorm.DB
}

func NewAvatarReplyDraftService(db *gorm.DB) *AvatarReplyDraftService {
	return &AvatarReplyDraftService{db: db}
}

// avatarDraftTTL 草稿有效期：过期不展示、不可批准，防止主人批准一条数十小时前的过时回复。
const avatarDraftTTL = 24 * time.Hour

// AvatarDraftInput 创建待审批草稿所需的输入参数（收敛多参数）。
type AvatarDraftInput struct {
	OwnerID          uint
	ConversationID   uint
	TriggerMessageID uint
	TriggerUserID    uint
	Content          string
	Reason           string
	Sources          []KnowledgeSource
}

// Create 落库待审批草稿并在通知中心留痕（红点 + 离线可见）。通知落库失败不影响草稿
// 本身（卡片仍可用），仅记日志。返回带 ID 的草稿。
func (s *AvatarReplyDraftService) Create(in AvatarDraftInput) *model.AvatarReplyDraft {
	sourcesJSON := ""
	if len(in.Sources) > 0 {
		if b, err := json.Marshal(in.Sources); err == nil {
			sourcesJSON = string(b)
		}
	}
	draft := &model.AvatarReplyDraft{
		UserID:           in.OwnerID,
		ConversationID:   in.ConversationID,
		TriggerMessageID: in.TriggerMessageID,
		TriggerUserID:    in.TriggerUserID,
		Content:          in.Content,
		Reason:           in.Reason,
		SourcesJSON:      sourcesJSON,
		Status:           "pending",
		ExpiresAt:        time.Now().Add(avatarDraftTTL),
	}
	if err := s.db.Create(draft).Error; err != nil {
		logger.WithModule("AvatarWorkerPool").Error("待审批草稿落库失败", "userID", in.OwnerID, "error", err)
		return nil
	}

	notification := &model.Notification{
		Type:          "avatar_reply_draft",
		Title:         "分身有一条回复待你确认",
		Content:       clipSnippet(in.Content),
		ActionType:    "avatar_reply_draft",
		ActionPayload: fmt.Sprintf(`{"draft_id":%d,"conversation_id":%d}`, draft.ID, draft.ConversationID),
	}
	notification.UserID = in.OwnerID
	if err := s.db.Create(notification).Error; err != nil {
		logger.WithModule("AvatarWorkerPool").Warn("草稿通知落库失败，卡片不受影响", "draftID", draft.ID, "error", err)
		return draft
	}
	if err := s.db.Model(draft).Update("notification_id", notification.ID).Error; err == nil {
		draft.NotificationID = &notification.ID
	}
	return draft
}

// ListPending 列出用户未过期的待审批草稿；conversationID>0 时只看该会话。
func (s *AvatarReplyDraftService) ListPending(ownerID, conversationID uint) []model.AvatarReplyDraft {
	q := s.db.Where("user_id = ? AND status = ? AND expires_at > ?", ownerID, "pending", time.Now())
	if conversationID > 0 {
		q = q.Where("conversation_id = ?", conversationID)
	}
	var drafts []model.AvatarReplyDraft
	if err := q.Order("created_at DESC").Find(&drafts).Error; err != nil {
		logger.WithModule("AvatarReplyDraft").Warn("查询待审批草稿失败", "error", err)
		return nil
	}
	return drafts
}

// GetForOwner 带归属校验读取草稿。不存在/跨用户均返回错误。
func (s *AvatarReplyDraftService) GetForOwner(ownerID, draftID uint) (*model.AvatarReplyDraft, error) {
	var draft model.AvatarReplyDraft
	if err := s.db.Where("id = ? AND user_id = ?", draftID, ownerID).First(&draft).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrMemoryNotFound
		}
		return nil, err
	}
	return &draft, nil
}

// MarkResolved 更新草稿终态（approved/discarded），并把对应通知标记已处理（消红点）。
func (s *AvatarReplyDraftService) MarkResolved(draft *model.AvatarReplyDraft, status string) error {
	if err := s.db.Model(draft).Update("status", status).Error; err != nil {
		return err
	}
	draft.Status = status
	if draft.NotificationID != nil {
		s.db.Model(&model.Notification{}).Where("id = ?", *draft.NotificationID).
			Updates(map[string]interface{}{"handled": true, "read": true})
	}
	return nil
}
