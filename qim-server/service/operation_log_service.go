package service

import (
	"github.com/dshmyz/qim/qim-server/model"
	"time"

	"github.com/gin-gonic/gin"
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

func (s *OperationLogService) LogUserOperation(c *gin.Context, module string, action string, details ...map[string]interface{}) {
	userID, _ := c.Get("user_id")
	username, _ := c.Get("username")

	var uid uint
	if id, ok := userID.(uint); ok {
		uid = id
	}

	var uname string
	if name, ok := username.(string); ok {
		uname = name
	}

	log := model.OperationLog{
		UserID:     uid,
		Username:   uname,
		Action:     action,
		Module:     module,
		IP:         c.ClientIP(),
		UserAgent:  c.GetHeader("User-Agent"),
		RequestURL: c.Request.URL.Path,
		CreatedAt:  time.Now(),
	}

	s.db.Create(&log)
}
