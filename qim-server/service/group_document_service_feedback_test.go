package service

import (
	"testing"

	"github.com/dshmyz/gracedb/pkg/types"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// ===== 展示评分优化测试 =====

// TestHybridDisplayScores_FTSOnly_RankBased
// 仅 FTS 命中时，分数应按 FTS 排名递减，不再一律 0.5。
func TestHybridDisplayScores_FTSOnly_RankBased(t *testing.T) {
	fts := []types.ScoredEmbedding{
		{Embedding: types.Embedding{ID: "a"}},
		{Embedding: types.Embedding{ID: "b"}},
		{Embedding: types.Embedding{ID: "c"}},
	}
	scored := hybridDisplayScores(fts, nil, fts)
	require.Len(t, scored, 3)
	assert.Greater(t, scored[0].Score, scored[2].Score, "FTS 排名靠前应得分更高")
	assert.GreaterOrEqual(t, scored[0].Score, float32(0.7), "第1名应≥0.7")
	assert.LessOrEqual(t, scored[2].Score, float32(0.6), "末名应≤0.6")
}

// TestHybridDisplayScores_DualPathBoost
// 语义+词法双路命中的块，展示分应在余弦相似度基础上加成 +0.08。
func TestHybridDisplayScores_DualPathBoost(t *testing.T) {
	sem := []types.ScoredEmbedding{
		{Embedding: types.Embedding{ID: "a"}, Score: 0.80},
	}
	fts := []types.ScoredEmbedding{
		{Embedding: types.Embedding{ID: "a"}},
	}
	scored := []types.ScoredEmbedding{
		{Embedding: types.Embedding{ID: "a"}, Score: 0.80},
	}
	result := hybridDisplayScores(scored, sem, fts)
	require.Len(t, result, 1)
	assert.InDelta(t, 0.88, result[0].Score, 0.01, "双路命中应加成 +0.08")
}

// TestHybridDisplayScores_SemanticOnly_NoBoost
// 仅语义命中（FTS 未命中同一块）时，保持原始余弦分，不加成。
func TestHybridDisplayScores_SemanticOnly_NoBoost(t *testing.T) {
	sem := []types.ScoredEmbedding{
		{Embedding: types.Embedding{ID: "a"}, Score: 0.75},
	}
	scored := []types.ScoredEmbedding{
		{Embedding: types.Embedding{ID: "a"}, Score: 0.75},
	}
	result := hybridDisplayScores(scored, sem, nil)
	require.Len(t, result, 1)
	assert.InDelta(t, 0.75, result[0].Score, 0.001, "仅语义命中不加成")
}

// TestHybridDisplayScores_BoostCappedAt1
// 双路加成后超过 1.0 应截断为 1.0。
func TestHybridDisplayScores_BoostCappedAt1(t *testing.T) {
	sem := []types.ScoredEmbedding{
		{Embedding: types.Embedding{ID: "a"}, Score: 0.95},
	}
	fts := []types.ScoredEmbedding{
		{Embedding: types.Embedding{ID: "a"}},
	}
	scored := []types.ScoredEmbedding{
		{Embedding: types.Embedding{ID: "a"}, Score: 0.95},
	}
	result := hybridDisplayScores(scored, sem, fts)
	require.Len(t, result, 1)
	assert.Equal(t, float32(1.0), result[0].Score, "0.95+0.08 应截断为 1.0")
}
