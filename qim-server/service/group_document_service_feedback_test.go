package service

import (
	"testing"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/sqlite"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// setupGroupDocumentTestDB 构造群文档测试所需的 in-memory DB（仅相关表）。
func setupGroupDocumentTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := db.AutoMigrate(&model.GroupDocument{}, &model.File{}, &model.DocumentProcessStatus{}); err != nil {
		t.Fatalf("migrate test db: %v", err)
	}
	return db
}

// TestProcessDocument_GracedbNil_MarksFailedInsteadOfStuckPending 验证：当向量服务/RAG
// 未初始化（gracedbDB==nil，即 SetVectorServices 未注入——常见于服务启动时
// NewVectorService 失败）时，绑定文档后触发 ProcessDocument 应将处理状态落成
// failed（带明确错误），而不是不落任何 status 记录。
//
// 背景：若 ProcessDocument 在依赖未就绪时直接 return 且不创建状态，前端
// GetDocumentsWithStatus 会因找不到 status 记录而把所有文档显示为 pending
// （"等待处理"）并永远卡住——用户无法自动恢复，只能误以为在处理中。
// 此测试钉住：这种情况必须落 failed，让用户看到明确失败。
func TestProcessDocument_GracedbNil_MarksFailedInsteadOfStuckPending(t *testing.T) {
	db := setupGroupDocumentTestDB(t)

	file := model.File{
		Name: "report.pdf", OriginalName: "report.pdf",
		MimeType: "application/pdf", Size: 1024, StoragePath: "/fake/report.pdf",
	}
	assert.NoError(t, db.Create(&file).Error)
	doc := model.GroupDocument{GroupID: 10, FileID: file.ID}
	assert.NoError(t, db.Create(&doc).Error)

	// 关键：不调用 SetVectorServices —— gracedbDB 保持 nil（向量服务未初始化）
	svc := NewGroupDocumentService(db, nil)

	err := svc.ProcessDocument(doc.ID)
	assert.Error(t, err, "向量服务未初始化时应返回错误")

	// 断言：必须存在 failed 状态记录，而非"无记录 → 前端显示 pending 永远卡住"
	var status model.DocumentProcessStatus
	assert.NoError(t,
		db.Where("group_doc_id = ?", doc.ID).First(&status).Error,
		"依赖未就绪时也必须落一条处理状态记录，否则前端会一直显示等待处理")
	assert.Equal(t, "failed", status.Status, "依赖未就绪时应标记为 failed")
	assert.NotEmpty(t, status.Error, "failed 状态应带明确错误说明")
}
