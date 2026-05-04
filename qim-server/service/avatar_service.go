package service

import (
	"fmt"
	"gorm.io/gorm"
)

// AvatarService 分身服务
type AvatarService struct {
	db *gorm.DB
}

// NewAvatarService 创建分身服务实例
func NewAvatarService(db *gorm.DB) *AvatarService {
	return &AvatarService{
		db: db,
	}
}

// LearnPersona 学习用户人设（占位实现）
func (s *AvatarService) LearnPersona(userID uint, taskID uint) {
	// TODO: 实现完整的学习逻辑
}

// PreviewReply 预览回复（占位实现）
func (s *AvatarService) PreviewReply(userID uint, message string) (string, error) {
	return "", fmt.Errorf("预览功能暂未实现")
}
