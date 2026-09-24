package service

import (
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/logger"

	"gorm.io/gorm"
)

// FailureCaseService 失败案例快照的写入。
type FailureCaseService struct {
	db *gorm.DB
}

func NewFailureCaseService(db *gorm.DB) *FailureCaseService {
	return &FailureCaseService{db: db}
}

// Record 写入一条失败案例。失败不阻断主流程（差评/回复本身优先），仅日志。
// 对超长内容截断，避免单条案例撑满表。
func (s *FailureCaseService) Record(category, scene string, conversationID uint, question, badReply string) {
	if s == nil || s.db == nil || category == "" {
		return
	}
	row := model.AIFailureCase{
		Category:       category,
		Scene:          scene,
		ConversationID: conversationID,
		Question:       truncateRunes(question, 1999),
		BadReply:       truncateRunes(badReply, 3999),
	}
	if err := s.db.Create(&row).Error; err != nil {
		logger.WithModule("FailureCase").Warn("失败案例落库失败", "error", err)
	}
}
