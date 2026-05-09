package model

import "time"

// DocumentProcessStatus 文档处理状态模型
type DocumentProcessStatus struct {
	ID         uint      `json:"id" gorm:"primarykey"`
	GroupDocID uint      `json:"group_doc_id" gorm:"not null;index"`
	Status     string    `json:"status" gorm:"size:20;default:'pending'"` // pending, processing, completed, failed
	Error      string    `json:"error" gorm:"type:text"`
	ChunkCount int       `json:"chunk_count"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
