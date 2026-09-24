package service

import (
	"encoding/json"
	"time"

	"github.com/dshmyz/qim/qim-server/model"

	"gorm.io/gorm"
)

// SignalTypeDraftEdit 草稿编辑 diff：质量核验拒绝的草稿经主人亲手改写后发出，
// original→final 即"系统所写 vs 理想回复"的对照样本。
const SignalTypeDraftEdit = "draft_edit"

// editSignalPayload 单条信号的载荷（JSON 存储）。
type editSignalPayload struct {
	DraftID         uint   `json:"draft_id"`
	OriginalContent string `json:"original_content"`
	FinalContent    string `json:"final_content"`
	RejectReason    string `json:"reject_reason"`
}

// AILearningSignalService 学习信号的写入与查询。
type AILearningSignalService struct {
	db *gorm.DB
}

func NewAILearningSignalService(db *gorm.DB) *AILearningSignalService {
	return &AILearningSignalService{db: db}
}

// RecordDraftEdit 记录一条草稿编辑 diff。失败不影响用户正常发送，错误返回给调用方记日志。
func (s *AILearningSignalService) RecordDraftEdit(ownerID, conversationID, draftID uint, original, final, rejectReason string) error {
	if s == nil || s.db == nil || ownerID == 0 {
		return nil
	}

	payload := editSignalPayload{
		DraftID:         draftID,
		OriginalContent: truncateRunes(original, 3999),
		FinalContent:    truncateRunes(final, 3999),
		RejectReason:    truncateRunes(rejectReason, 499),
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	return s.db.Create(&model.AILearningSignal{
		OwnerID:        ownerID,
		ConversationID: conversationID,
		SignalType:     SignalTypeDraftEdit,
		PayloadJSON:    string(raw),
		CreatedAt:      time.Now(),
	}).Error
}
