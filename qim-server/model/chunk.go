package model

import (
	"time"

	"gorm.io/gorm"
)

// FileChunk 文件分片模型
type FileChunk struct {
	ID          uint           `json:"id" gorm:"primarykey"`
	UploadID    string         `json:"upload_id" gorm:"size:64;not null;index:idx_upload_chunk"` // 上传任务唯一标识
	FileHash    string         `json:"file_hash" gorm:"size:64;not null"`                        // 文件整体 MD5
	ChunkIndex  int            `json:"chunk_index" gorm:"not null;index:idx_upload_chunk"`       // 分片序号，从0开始
	ChunkHash   string         `json:"chunk_hash" gorm:"size:64;not null"`                       // 分片 MD5
	ChunkSize   int64          `json:"chunk_size" gorm:"not null"`                               // 分片大小（字节）
	StoragePath string         `json:"storage_path" gorm:"size:500;not null"`                    // 分片存储路径
	Status      string         `json:"status" gorm:"size:20;not null;default:'pending'"`        // pending/uploaded/merged
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

// UploadTask 上传任务模型
type UploadTask struct {
	ID              uint           `json:"id" gorm:"primarykey"`
	UploadID        string         `json:"upload_id" gorm:"size:64;uniqueIndex;not null"` // 上传任务唯一标识
	UserID          uint           `json:"user_id" gorm:"not null;index"`                 // 用户ID
	Filename        string         `json:"filename" gorm:"size:255;not null"`             // 文件名
	FileSize        int64          `json:"file_size" gorm:"not null"`                     // 文件大小（字节）
	FileHash        string         `json:"file_hash" gorm:"size:64;not null"`             // 文件整体 MD5
	TotalChunks     int            `json:"total_chunks" gorm:"not null"`                  // 总分片数
	UploadedChunks  int            `json:"uploaded_chunks" gorm:"default:0"`              // 已上传分片数
	FolderID        *uint          `json:"folder_id" gorm:"index"`                        // 文件夹ID，可为空
	Status          string         `json:"status" gorm:"size:20;not null;default:'pending'"` // pending/uploading/completed/failed/cancelled
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `json:"-" gorm:"index"`
	User            User           `json:"user,omitempty" gorm:"foreignkey:UserID"`
}
