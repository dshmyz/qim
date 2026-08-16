package service

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/dshmyz/gracedb/pkg/gracedb"
	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// ocrCaptureProvider 视觉 OCR 测试用 Provider 桩：在复用 capturingAvatarProvider 的
// Chat 消息捕获与固定回复基础上，Embedding 返回对齐 fakeEmbedder 维度的伪向量，
// 使 OCR 结果能走完整「切片 → 向量化 → 落库」流程。
type ocrCaptureProvider struct {
	capturingAvatarProvider
}

func (p *ocrCaptureProvider) Embedding(text string) ([]float32, error) {
	return []float32{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8}, nil
}

// WithModel 覆盖 fakeAvatarProvider 的实现：默认返回被嵌入的裸 fakeAvatarProvider，
// 会丢掉消息捕获。这里返回自身，使 GetCompletion（带 model 路由时走
// provider.WithModel(model).Chat(...)）仍能命中 capturingAvatarProvider.Chat 的记录。
func (p *ocrCaptureProvider) WithModel(model string) ai.Provider {
	return p
}

// newOcrTestService 构造群文档 OCR 测试所需的服务：内存 sqlite + 临时 gracedb +
// 本地文件存储 + 带/不带视觉路由的 AI 服务。imageData 写入存储，供 ProcessDocument
// 落临时文件后转 base64 data URL。
func newOcrTestService(t *testing.T, imageMime string, visionRoute bool, ocrReply string) (*GroupDocumentService, *ocrCaptureProvider, uint) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.GroupDocument{}, &model.File{}, &model.DocumentProcessStatus{}))

	gdb, err := gracedb.Open(t.TempDir()+"/gracedb", gracedb.WithEmbedder(fakeEmbedder{}))
	require.NoError(t, err)
	t.Cleanup(func() { _ = gdb.Close() })

	store := newTestAccessor(t)
	imageData := []byte("fake png bytes for OCR test")
	storagePath, err := store.Put(context.Background(), "screenshot.png", bytes.NewReader(imageData), int64(len(imageData)), imageMime)
	require.NoError(t, err)

	file := model.File{
		Name: "screenshot.png", OriginalName: "screenshot.png",
		MimeType: imageMime, Size: int64(len(imageData)), StoragePath: storagePath,
	}
	require.NoError(t, db.Create(&file).Error)
	doc := model.GroupDocument{GroupID: 10, FileID: file.ID}
	require.NoError(t, db.Create(&doc).Error)

	routes := map[ai.TaskType]ai.Route{
		ai.TaskTypeChat: {Provider: "ocr", Model: "chat"},
	}
	if visionRoute {
		routes[ai.TaskTypeVision] = ai.Route{Provider: "ocr", Model: "vision"}
	}
	aiSvc := ai.NewAIService(&ai.AIConfig{
		Router: ai.RouterConfig{
			DefaultTask: ai.TaskTypeChat,
			Routes:      routes,
		},
	})
	capProv := &ocrCaptureProvider{}
	capProv.reply = ocrReply
	aiSvc.SetProviderForTesting("ocr", capProv)

	svc := NewGroupDocumentService(db, store)
	svc.gracedbDB = gdb
	svc.aiService = aiSvc
	svc.parser = &DocumentParser{}
	return svc, capProv, doc.ID
}

// TestProcessDocument_ImageOCR 验证直接上传的图片走视觉 OCR 入库：
// 1) OCR prompt 携带图片 base64 data URL；2) OCR 识别结果进入切片向量化流程
// （状态 completed、chunk_count>0）；3) 切片内容真实落库（经 gracedb 读回）。
func TestProcessDocument_ImageOCR(t *testing.T) {
	svc, capProv, docID := newOcrTestService(t, "image/png", true, "确认函：已收到采购订单号 PO-2024-001")
	require.NoError(t, svc.ProcessDocument(docID))

	var status model.DocumentProcessStatus
	require.NoError(t, svc.db.Where("group_doc_id = ?", docID).First(&status).Error)
	assert.Equal(t, "completed", status.Status, "图片 OCR 应走完整入库流程")
	assert.Greater(t, status.ChunkCount, 0, "OCR 结果应切成向量切片")

	// OCR prompt 携带图片 base64 data URL
	var lastUser ai.Message
	for _, m := range capProv.lastMessages {
		if m.Role == "user" {
			lastUser = m
		}
	}
	assert.True(t, strings.HasPrefix(lastUser.ImageURL, "data:image/png;base64,"), "OCR prompt 应携带图片 data URL")
	assert.Contains(t, lastUser.Content, "逐字识别", "OCR prompt 应要求逐字识别图片文字")

	// OCR 结果真实落库：collection 内存在该文档的切片，且内容为识别出的文字
	ids, err := svc.gracedbDB.ListEmbeddingIDs("group_10")
	require.NoError(t, err)
	found := false
	for _, id := range ids {
		emb, err := svc.gracedbDB.GetEmbedding("group_10", id, true)
		if err != nil || emb == nil {
			continue
		}
		if emb.Metadata["doc_id"] == fmt.Sprintf("%d", docID) && strings.Contains(emb.Content, "PO-2024-001") {
			found = true
			break
		}
	}
	assert.True(t, found, "OCR 识别出的文字应进入向量切片并落库")
}

// TestProcessDocument_ImageOCRNoVisionRoute 验证未配置视觉路由时，图片入库降级为
// failed + 明确文案（提示配置视觉模型），而不是卡在 pending 让用户无法重试。
func TestProcessDocument_ImageOCRNoVisionRoute(t *testing.T) {
	svc, _, docID := newOcrTestService(t, "image/png", false, "不应被调用")
	err := svc.ProcessDocument(docID)
	require.Error(t, err)

	var status model.DocumentProcessStatus
	require.NoError(t, svc.db.Where("group_doc_id = ?", docID).First(&status).Error)
	assert.Equal(t, "failed", status.Status, "未配置视觉路由时应标记 failed 而非卡 pending")
	assert.Contains(t, status.Error, "视觉", "failed 状态应带明确错误说明")
}
