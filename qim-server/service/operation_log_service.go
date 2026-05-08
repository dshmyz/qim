package service

import (
	"qim-server/model"

	"gorm.io/gorm"
)

type OperationLogService struct {
	db *gorm.DB
}

func NewOperationLogService(db *gorm.DB) *OperationLogService {
	return &OperationLogService{db: db}
}

func (s *OperationLogService) GetLogs(page, pageSize int) ([]model.OperationLog, int64, error) {
	var logs []model.OperationLog
	var total int64

	s.db.Model(&model.OperationLog{}).Count(&total)
	err := s.db.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&logs).Error
	return logs, total, err
}

func (s *OperationLogService) CreateLog(log *model.OperationLog) error {
	return s.db.Create(log).Error
}
