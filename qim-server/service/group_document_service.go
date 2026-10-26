package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/logger"

	"github.com/dshmyz/gracedb/pkg/gracedb"
	"github.com/dshmyz/gracedb/pkg/graph"
	"github.com/dshmyz/gracedb/pkg/types"
	"gorm.io/gorm"
)

// staleProcessingAfter 判定处理状态超时的阈值：超过该时长仍停留在 processing，
// 视为崩溃/被杀/旧代码遗留的僵尸任务，读状态时自动重置为 failed 以便重试。
const staleProcessingAfter = 5 * time.Minute

// ErrQuotedFileTooLarge 被引用文件体积超过 quoteMaxFileSize 时返回的哨兵错误，
// 供 smart_reply_graph 区分"文件过大"与"类型不支持/解析失败"两种读不了场景。
var ErrQuotedFileTooLarge = errors.New("被引用文件过大")

// quoteMaxFileSize 被引用文件允许注入上下文的最大体积（字节）。
// 正文注入前只会截断到约 4000 字符，故无需读超大文件；docx/xlsx/pptx 是 zip 容器，
// Size 为压缩包大小且解析文本量不可预知，故用相对宽松的 20MB 阈值。
const quoteMaxFileSize int64 = 20 * 1024 * 1024

// ErrQuotedImageTooLarge 被引用图片体积超过 quoteMaxImageSize 时返回的哨兵错误，
// 供 smart_reply_graph 区分"图片过大"与"读取失败"两种看不了图片的场景。
var ErrQuotedImageTooLarge = errors.New("被引用图片过大")

// quoteMaxImageSize 被引用图片允许注入多模态上下文的最大体积（字节）。
// 图片以 base64 data URL 注入（base64 膨胀约 1.33x），故阈值取 5MB，
// 与 TranslateImage 的 maxImageTranslateSize 一致，避免超大图撑爆 prompt。
const quoteMaxImageSize int64 = 5 * 1024 * 1024

type GroupDocumentService struct {
	db        *gorm.DB
	gracedbDB *gracedb.DB
	aiService *ai.AIService
	parser    *DocumentParser
	store     StorageAccessor

	// procMu/procInFlight 记录正在处理的 groupDocID，防止重试连点为同一文档堆叠多个处理 goroutine
	procMu       sync.Mutex
	procInFlight map[uint]struct{}
}

func NewGroupDocumentService(db *gorm.DB, store StorageAccessor) *GroupDocumentService {
	return &GroupDocumentService{
		db:           db,
		store:        store,
		procInFlight: make(map[uint]struct{}),
	}
}

func (s *GroupDocumentService) SetVectorServices(vectorSvc *VectorService, aiService *ai.AIService) {
	if vectorSvc == nil {
		return
	}
	s.gracedbDB = vectorSvc.GetDB()
	s.aiService = aiService
	s.parser = &DocumentParser{}
}

func (s *GroupDocumentService) GetDocumentsByGroup(groupID uint) ([]model.GroupDocument, error) {
	var docs []model.GroupDocument
	err := s.db.Where("group_id = ?", groupID).Order("created_at DESC").Find(&docs).Error
	return docs, err
}

func (s *GroupDocumentService) GetDocumentByID(id uint) (*model.GroupDocument, error) {
	var doc model.GroupDocument
	err := s.db.First(&doc, id).Error
	return &doc, err
}

func (s *GroupDocumentService) CreateDocument(doc *model.GroupDocument) error {
	return s.db.Create(doc).Error
}

func (s *GroupDocumentService) UpdateDocument(doc *model.GroupDocument) error {
	return s.db.Save(doc).Error
}

func (s *GroupDocumentService) DeleteDocument(id uint) error {
	return s.db.Delete(&model.GroupDocument{}, id).Error
}

// RemoveGroupDocument 解绑群文档：删除数据库记录，同时清理向量切片与图谱节点/边。
// 旧实现只删 group_documents 行，解绑后的文档内容仍会被检索到、图里也长期残留。
func (s *GroupDocumentService) RemoveGroupDocument(groupID, fileID uint) (int, error) {
	var docs []model.GroupDocument
	if err := s.db.Where("group_id = ? AND file_id = ?", groupID, fileID).Find(&docs).Error; err != nil {
		return 0, err
	}
	collectionName := fmt.Sprintf("group_%d", groupID)
	removed := 0
	for _, d := range docs {
		if err := s.deleteDocumentVectors(collectionName, d.ID); err != nil {
			// 切片没删干净就不能删记录：group_documents 行一旦没了，这批向量就无法再按
			// doc_id 定位，解绑的文档会继续被检索到、继续进 AI 回答。保留记录即可重试。
			logger.WithModule("GroupDocument").Error("删除文档切片失败，保留文档记录",
				"docID", d.ID, "error", err)
			return removed, fmt.Errorf("删除文档切片失败: %w", err)
		}
		if err := s.deleteDocumentGraph(groupID, d.ID); err != nil {
			logger.WithModule("GroupDocument").Warn("删除文档图谱失败", "docID", d.ID, "error", err)
		}
		if err := s.db.Delete(&model.GroupDocument{}, d.ID).Error; err != nil {
			return removed, err
		}
		removed++
	}
	return removed, nil
}

func (s *GroupDocumentService) GetDocumentsWithStatus(groupID uint) ([]map[string]interface{}, error) {
	var docs []model.GroupDocument
	if err := s.db.Where("group_id = ?", groupID).Preload("File").Order("created_at DESC").Find(&docs).Error; err != nil {
		return nil, err
	}

	var results []map[string]interface{}
	for _, doc := range docs {
		var status model.DocumentProcessStatus
		s.db.Where("group_doc_id = ?", doc.ID).Order("created_at DESC").First(&status)

		result := map[string]interface{}{
			"id":         doc.ID,
			"group_id":   doc.GroupID,
			"file_id":    doc.FileID,
			"created_at": doc.CreatedAt,
			"file":       doc.File,
		}

		if status.ID > 0 {
			result["process_status"] = status.Status
			result["process_error"] = status.Error
			result["chunk_count"] = status.ChunkCount

			// 自愈：processing 超时（崩溃/被杀/旧代码遗留）的任务，读到即重置为
			// failed，让用户能重试，而不是永远卡在"处理中"。
			if status.Status == "processing" && time.Since(status.UpdatedAt) > staleProcessingAfter {
				s.db.Model(&status).Updates(map[string]interface{}{
					"status": "failed",
					"error":  "处理超过 5 分钟仍未完成，已重置，请点击重试",
				})
				result["process_status"] = "failed"
				result["process_error"] = "处理超过 5 分钟仍未完成，已重置，请点击重试"
			}
		} else {
			result["process_status"] = "pending"
		}

		results = append(results, result)
	}

	return results, nil
}

// ensureDocProcessStatus 读取该文档最新的处理状态记录；不存在则创建一条 pending。
// 保证即使处理在早期前置条件（如向量服务未初始化）就失败，也能把失败落到状态上，
// 不会出现"无状态记录 → 前端 GetDocumentsWithStatus 显示为等待处理并永远卡住"。
func ensureDocProcessStatus(db *gorm.DB, groupDocID uint) (*model.DocumentProcessStatus, error) {
	var status model.DocumentProcessStatus
	if err := db.Where("group_doc_id = ?", groupDocID).Order("created_at DESC").First(&status).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			return nil, err
		}
		status = model.DocumentProcessStatus{GroupDocID: groupDocID, Status: "pending"}
		if err := db.Create(&status).Error; err != nil {
			return nil, err
		}
	}
	return &status, nil
}

func (s *GroupDocumentService) ProcessDocument(groupDocID uint) (err error) {
	if s.gracedbDB == nil {
		// 向量服务/RAG 未初始化（常见于服务启动时 NewVectorService 失败）时无法处理，
		// 但仍要落一条 failed 状态记录，否则前端 GetDocumentsWithStatus 因找不到状态
		// 记录而把所有文档显示为 pending（"等待处理"）并永远卡住、用户无法重试。
		if status, serr := ensureDocProcessStatus(s.db, groupDocID); serr == nil {
			s.db.Model(status).Updates(map[string]interface{}{
				"status": "failed",
				"error":  "向量服务未初始化，无法处理文档",
			})
		}
		return fmt.Errorf("向量服务未初始化")
	}

	// 并发防重入：同一文档同时只允许一个处理任务，避免多次重试堆叠 goroutine、
	// 彼此覆盖状态。已有处理中任务时直接拒绝本次调用。
	s.procMu.Lock()
	if _, inFlight := s.procInFlight[groupDocID]; inFlight {
		s.procMu.Unlock()
		return fmt.Errorf("该文档正在处理中，请稍候")
	}
	s.procInFlight[groupDocID] = struct{}{}
	s.procMu.Unlock()
	// 无论正常返回还是 panic，都释放 in-flight 槽位
	defer func() {
		s.procMu.Lock()
		delete(s.procInFlight, groupDocID)
		s.procMu.Unlock()
	}()

	var doc model.GroupDocument
	if err := s.db.Preload("File").First(&doc, groupDocID).Error; err != nil {
		return fmt.Errorf("文档不存在")
	}

	// 用于 panic 兜底：status 在下方初始化，defer 闭包见到的即函数结束时最后写入的值。
	// 若处理过程 panic，先把状态落成 failed（而非卡死在 processing），再原样抛出交由
	// 上层 SafeGoWithLabel 记录，避免僵尸 processing 让用户无法重试。
	status, err := ensureDocProcessStatus(s.db, groupDocID)
	if err != nil {
		return fmt.Errorf("初始化处理状态失败: %w", err)
	}
	defer func() {
		if r := recover(); r != nil {
			if status.ID > 0 {
				s.db.Model(status).Updates(map[string]interface{}{
					"status": "failed",
					"error":  fmt.Sprintf("处理过程异常: %v", r),
				})
			}
			panic(r)
		}
	}()

	s.db.Model(status).Updates(map[string]interface{}{
		"status": "processing",
	})

	// 通过存储抽象读取文件内容到临时文件，再交给 parser（pdf/docx 库需要文件路径）
	reader, err := s.store.GetByPath(context.Background(), doc.File.StoragePath)
	if err != nil {
		s.db.Model(status).Updates(map[string]interface{}{
			"status": "failed",
			"error":  fmt.Sprintf("读取文件失败: %v", err),
		})
		return err
	}
	tmpFile, err := os.CreateTemp("", "qim-doc-*"+filepath.Ext(doc.File.Name))
	if err != nil {
		reader.Close()
		s.db.Model(status).Updates(map[string]interface{}{
			"status": "failed",
			"error":  fmt.Sprintf("创建临时文件失败: %v", err),
		})
		return err
	}
	tmpPath := tmpFile.Name()
	if _, err := io.Copy(tmpFile, reader); err != nil {
		reader.Close()
		tmpFile.Close()
		os.Remove(tmpPath)
		s.db.Model(status).Updates(map[string]interface{}{
			"status": "failed",
			"error":  fmt.Sprintf("写入临时文件失败: %v", err),
		})
		return err
	}
	reader.Close()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	var text string
	if isImageDocument(doc.File.MimeType) {
		// 直接上传的图片（截图/照片/扫描件）：视觉 OCR 识别文字后走既有切片+向量化流程。
		// 扫描件 PDF 不在此列——MIME 为 application/pdf，仍走 parser 并维持"无文字层"明确报错。
		text, err = s.ocrImageToText(tmpPath, doc.File.Name)
		if err != nil {
			s.db.Model(status).Updates(map[string]interface{}{
				"status": "failed",
				"error":  "图片文字识别失败: " + err.Error(),
			})
			return err
		}
	} else {
		text, err = s.parser.Parse(tmpPath)
		if err != nil {
			s.db.Model(status).Updates(map[string]interface{}{
				"status": "failed",
				"error":  "文档解析失败: " + describeParseError(err),
			})
			return err
		}
	}

	if text == "" || len(text) < 10 {
		s.db.Model(status).Updates(map[string]interface{}{
			"status": "failed",
			"error":  "文档内容为空或太短",
		})
		return fmt.Errorf("文档内容为空")
	}

	collectionName := fmt.Sprintf("group_%d", doc.GroupID)
	chunks := ChunkDocument(text, 800)
	if len(chunks) == 0 {
		chunks = []Chunk{{Content: text, Title: doc.File.Name}}
	}

	// 确保群集合存在（gracedb Upsert 不会自动建集合）
	if err := ensureGracedbCollection(s.gracedbDB, collectionName); err != nil {
		s.db.Model(status).Updates(map[string]interface{}{
			"status": "failed",
			"error":  fmt.Sprintf("创建向量集合失败: %v", err),
		})
		return err
	}

	// 先清掉该文档旧向量（重处理场景），再写入新切片
	if err := s.deleteDocumentVectors(collectionName, doc.ID); err != nil {
		logger.WithModule("GroupDocument").Warn("清理文档旧向量失败", "doc_id", doc.ID, "error", err)
	}

	// 逐块向量化（无批量 embedding API，串行调用），收集后一次性批量写入，
	// 把 N 次分散的 Badger 写事务合并为一次 UpsertBatch，降低写入开销。
	// 单块 embedding 带 3 次指数退避重试（见 embedWithRetry）。
	type pendingChunk struct {
		docID string
		vec   []float32
		text  string
		meta  map[string]string
	}
	var pending []pendingChunk
	for i, chunk := range chunks {
		if len(chunk.Content) < 10 {
			continue // 跳过太短的片段
		}
		// 函数级重试（3 次指数退避）：embedding 服务偶发 5xx/网络抖动曾直接废掉
		// 整个文档（旧向量已删，只能全量重来）。只重试，不改变失败语义。
		embedding, err := embedWithRetry(s.aiService, chunk.Content)
		if err != nil {
			s.db.Model(status).Updates(map[string]interface{}{
				"status": "failed",
				"error":  fmt.Sprintf("切片向量化失败: %v", err),
			})
			return err
		}
		pending = append(pending, pendingChunk{
			docID: fmt.Sprintf("group_%d_doc_%d_chunk_%d", doc.GroupID, doc.ID, i),
			vec:   embedding,
			text:  chunk.Content,
			meta: map[string]string{
				"group_id": fmt.Sprintf("%d", doc.GroupID),
				"doc_id":   fmt.Sprintf("%d", doc.ID),
				"file_id":  fmt.Sprintf("%d", doc.FileID),
				"title":    doc.File.Name,
			},
		})
	}

	// 批量写入所有切片
	if len(pending) > 0 {
		vectors := make([][]float32, 0, len(pending))
		contents := make([]string, 0, len(pending))
		docIDs := make([]string, 0, len(pending))
		metas := make([]map[string]string, 0, len(pending))
		for _, p := range pending {
			vectors = append(vectors, p.vec)
			contents = append(contents, p.text)
			docIDs = append(docIDs, p.docID)
			metas = append(metas, p.meta)
		}
		if err := s.gracedbDB.UpsertBatch(collectionName, vectors, contents, docIDs, metas); err != nil {
			s.db.Model(status).Updates(map[string]interface{}{
				"status": "failed",
				"error":  fmt.Sprintf("切片批量存储失败: %v", err),
			})
			return err
		}
	}

	chunkCount := len(pending)

	// 知识图谱（GraphRAG MVP）：处理完成后为文档建立实体关系图，
	// 供群助手"关系问答"（如 PRD-2024-001 关联了什么）在向量召回外扩展邻接上下文。
	// 建图失败不阻断主流程，仅记日志（图谱是检索增强，向量切片已成功落库）。
	if err := s.buildDocumentGraph(doc.GroupID, doc.ID, doc.File.Name, text); err != nil {
		logger.WithModule("GroupDocument").Warn("构建文档知识图谱失败", "doc_id", doc.ID, "error", err)
	}

	s.db.Model(status).Updates(map[string]interface{}{
		"status":      "completed",
		"chunk_count": chunkCount,
	})

	logger.WithModule("GroupDocument").Info("文档处理完成", "doc_id", doc.ID, "chunks", chunkCount)
	return nil
}

// embedWithRetry 对单条 embedding 调用做 3 次指数退避重试（200ms/400ms/800ms）。
// Embedding 服务偶发 5xx/网络抖动不应让整个文档处理失败（旧向量已删，
// 失败意味着用户必须手动重试全量）。3 次后仍失败按原错误返回。
func embedWithRetry(aiSvc *ai.AIService, text string) ([]float32, error) {
	const attempts = 3
	var lastErr error
	for i := 0; i < attempts; i++ {
		if i > 0 {
			time.Sleep(time.Duration(1<<(i-1)) * 200 * time.Millisecond)
		}
		vec, err := aiSvc.Embed(text)
		if err == nil {
			return vec, nil
		}
		lastErr = err
		logger.WithModule("GroupDocument").Warn("embedding 调用失败，准备重试",
			"attempt", i+1, "maxAttempts", attempts, "error", err)
	}
	return nil, lastErr
}

// deleteDocumentVectors 删除某个文档的所有语义切片（按 metadata.doc_id 过滤）。
//
// gracedb 没有 delete-by-metadata API，仍需遍历集合逐条核对 metadata；但把匹配到的
// ID 汇总后走一次 DeleteEmbeddingBatch（单个事务批量删），替代逐条 DeleteByDocID，
// 减少 Badger 写事务次数。ListEmbeddingIDs 保全部遍历，确保不遗漏任何属于该文档的切片。
func (s *GroupDocumentService) deleteDocumentVectors(collectionName string, docID uint) error {
	if s.gracedbDB == nil {
		return nil
	}
	ids, err := s.gracedbDB.ListEmbeddingIDs(collectionName)
	if err != nil {
		// 集合不存在 = 该文档从未产生切片（绑定后未处理过），按"无可删"返回成功；
		// 其余错误必须上抛，否则调用方会误以为切片已清干净。
		if errors.Is(err, types.ErrNotFound) {
			return nil
		}
		return err
	}
	docIDStr := fmt.Sprintf("%d", docID)
	var toDelete []string
	for _, id := range ids {
		emb, err := s.gracedbDB.GetEmbedding(collectionName, id, true)
		if err != nil || emb == nil {
			continue
		}
		if emb.Metadata["doc_id"] == docIDStr {
			toDelete = append(toDelete, id)
		}
	}
	if len(toDelete) == 0 {
		return nil
	}
	if err := s.gracedbDB.DeleteEmbeddingBatch(collectionName, toDelete); err != nil {
		logger.WithModule("GroupDocument").Warn("批量删除文档切片失败", "docID", docID, "count", len(toDelete), "error", err)
		return err
	}
	return nil
}

// 群知识图谱的预算与构造规则。
//
// 旧实现把「同一文档内实体两两全连接」写成持久化的 co_occurs 边，边数随实体数平方增长；
// 再叠加 BFS 两跳，80 个节点能带出上千条边，前端力导向直接卡死。这里改为：
//   - 存储层只保留 文档→实体 mentions 边（线性规模）；
//   - 实体间关系在「构建时」按跨文档共现计算，且每个实体只保留有限邻居；
//   - 实体节点 ID 带群前缀，避免跨群共享同一 entity 节点导致串味。
const (
	groupGraphDefaultMaxNodes = 80
	groupGraphHardMaxNodes    = 200
	groupGraphEdgesPerNode    = 4 // 默认边预算 = 节点数 × 该系数
	groupGraphHardMaxEdges    = 1200
	groupGraphEdgePerEntity   = 6   // 每个实体最多保留的跨文档共现邻居
	groupGraphMaxDocs         = 500 // 单群参与建图的文档上限（按创建时间倒序取最近）
	groupGraphMinCoDoc        = 2   // 实体对至少共同出现在多少个文档才建边
	groupGraphPairEntityCap   = 40  // 单文档参与共现配对的实体上限
)

func groupDocNodeID(groupID, docID uint) string {
	return fmt.Sprintf("g%d:doc:%d", groupID, docID)
}

func groupEntityNodeID(groupID uint, name string) string {
	return fmt.Sprintf("g%d:entity:%s", groupID, name)
}

// legacyDocNodeID / legacyEntityNodeID 是旧版全局命名空间下的节点 ID，仅用于清理与迁移。
func legacyDocNodeID(docID uint) string { return fmt.Sprintf("doc:%d", docID) }

func legacyEntityNodeID(name string) string { return "entity:" + name }

// validGraphEntity 过滤建图实体：保留 PRD-2024-001 / BUG-123 / bug-123 这类编号 token，
// 丢掉 co-occurs / e-mail 这类自然语言连字符词，以及 utf-8 / sha-256 这类通用技术编码。
// 历史数据里已经写入的噪音实体也靠这道过滤在构建期自然消失，无需重新抽取正文。
func validGraphEntity(name string) bool {
	name = strings.TrimSpace(name)
	r := []rune(name)
	if len(r) < 3 || len(r) > 40 {
		return false
	}
	hasDigit, hasLetter := false, false
	for _, c := range name {
		if c >= '0' && c <= '9' {
			hasDigit = true
		}
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
			hasLetter = true
		}
	}
	if !hasDigit || !hasLetter {
		return false
	}
	switch strings.ToLower(name) {
	case "utf-8", "utf-16", "utf-32", "sha-1", "sha-256", "sha-512", "md5", "base64":
		return false
	}
	return true
}

func filterGraphEntities(names []string) []string {
	seen := make(map[string]bool, len(names))
	out := make([]string, 0, len(names))
	for _, n := range names {
		n = strings.TrimSpace(n)
		if !validGraphEntity(n) || seen[n] {
			continue
		}
		seen[n] = true
		out = append(out, n)
	}
	return out
}

// buildDocumentGraph 为单个群文档建立实体关系图。
//
// 存储结构（幂等）：一个文档节点 + 实体节点 + 文档→实体 mentions 边。
// 实体间关系不再落库（见 groupGraph 常量组的说明），由 BuildGroupKnowledgeGraph
// 按跨文档共现现算，边数天然有界。重复处理同一文档前先清掉旧边，避免实体减少后残留。
func (s *GroupDocumentService) buildDocumentGraph(groupID, docID uint, title, text string) error {
	if s.gracedbDB == nil {
		return nil
	}
	if err := s.deleteDocumentGraph(groupID, docID); err != nil {
		logger.WithModule("GroupDocument").Warn("清理旧图谱失败，继续重建",
			"groupID", groupID, "docID", docID, "error", err)
	}

	entities := filterGraphEntities(extractGraphEntities(text))
	if err := s.writeDocumentGraph(groupID, docID, title, entities); err != nil {
		return err
	}

	logger.WithModule("GroupDocument").Info("文档知识图谱构建完成",
		"group_id", groupID, "doc_id", docID, "entities", len(entities))
	return nil
}

// writeDocumentGraph 写入规范化后的文档节点与 mentions 边（不做清理，调用方负责）。
func (s *GroupDocumentService) writeDocumentGraph(groupID, docID uint, title string, entities []string) error {
	g := s.gracedbDB.Graph()
	docNodeID := groupDocNodeID(groupID, docID)

	props := map[string]string{
		"group_id": fmt.Sprintf("%d", groupID),
		"doc_id":   fmt.Sprintf("%d", docID),
		"title":    title,
	}
	if len(entities) > 0 {
		if raw, err := json.Marshal(entities); err == nil {
			props["entities"] = string(raw)
		}
	}
	if err := g.UpsertNode(&graph.GraphNode{
		ID:         docNodeID,
		Type:       "document",
		Labels:     []string{"group_document"},
		Properties: props,
	}); err != nil {
		return fmt.Errorf("写入文档节点失败: %w", err)
	}

	for _, e := range entities {
		entityID := groupEntityNodeID(groupID, e)
		if err := g.UpsertNode(&graph.GraphNode{
			ID:     entityID,
			Type:   "entity",
			Labels: []string{"extracted"},
			Properties: map[string]string{
				"name":     e,
				"group_id": fmt.Sprintf("%d", groupID),
			},
		}); err != nil {
			return fmt.Errorf("写入实体节点失败: %w", err)
		}
		if err := g.UpsertEdge(&graph.GraphEdge{
			ID:         fmt.Sprintf("%s-mentions-%s", docNodeID, entityID),
			FromNodeID: docNodeID,
			ToNodeID:   entityID,
			Type:       "mentions",
			Weight:     1.0,
		}); err != nil {
			return fmt.Errorf("写入 mentions 边失败: %w", err)
		}
	}
	return nil
}

// extractGraphEntities 用确定性规则从文档正文抽取候选实体名。
//
// MVP 覆盖高置信 token：文档编码（如 PRD-2024-001、BUG-123）、URL 域内的编号串等。
// 返回去重后的实体名列表；抽取不到时返回空（调用方据此跳过建图边）。
func extractGraphEntities(text string) []string {
	if text == "" {
		return nil
	}
	seen := make(map[string]bool)
	var out []string
	// 编码类 token：大写字母(可带数字) 或 数字后跟连字符和数字，如 PRD-2024-001 / BUG-123 / V1.2-3
	codeRe := regexp.MustCompile(`[A-Za-z][A-Za-z0-9]*[-_][0-9A-Za-z]+(?:[-_][0-9A-Za-z]+)*`)
	for _, m := range codeRe.FindAllString(text, -1) {
		if !validGraphEntity(m) {
			continue
		}
		if !seen[m] {
			seen[m] = true
			out = append(out, m)
		}
	}
	return out
}

// entityNameFromNode 取节点上的实体名，兼容新旧两种节点 ID 命名空间。
func entityNameFromNode(n *graph.GraphNode) string {
	if n == nil {
		return ""
	}
	if v := strings.TrimSpace(n.Properties["name"]); v != "" {
		return v
	}
	if i := strings.Index(n.ID, ":entity:"); i >= 0 {
		return n.ID[i+len(":entity:"):]
	}
	return strings.TrimPrefix(n.ID, "entity:")
}

// mentionEntities 读取某文档节点出边（mentions）上的实体名。
func (s *GroupDocumentService) mentionEntities(docNodeID string) []string {
	if s.gracedbDB == nil {
		return nil
	}
	nodes, _, err := s.gracedbDB.Graph().GetNeighbors(docNodeID, graph.NeighborOptions{
		Direction: "out", EdgeTypes: []string{"mentions"}, MaxDepth: 1,
	})
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(nodes))
	for _, n := range nodes {
		if n.Type != "entity" {
			continue
		}
		if name := entityNameFromNode(n); name != "" {
			names = append(names, name)
		}
	}
	return filterGraphEntities(names)
}

// documentEntityNames 返回文档参与建图的实体名，兼容三种历史形态：
// 新节点属性 entities（当前写入格式）→ 新节点 mentions 边 → 旧版全局节点 mentions 边。
func (s *GroupDocumentService) documentEntityNames(groupID, docID uint) []string {
	if s.gracedbDB == nil {
		return nil
	}
	g := s.gracedbDB.Graph()
	docNodeID := groupDocNodeID(groupID, docID)
	if n, err := g.GetNode(docNodeID); err == nil && n != nil {
		if raw := n.Properties["entities"]; raw != "" {
			var names []string
			if json.Unmarshal([]byte(raw), &names) == nil {
				if filtered := filterGraphEntities(names); len(filtered) > 0 {
					return filtered
				}
			}
		}
		if names := s.mentionEntities(docNodeID); len(names) > 0 {
			return names
		}
	}
	return s.mentionEntities(legacyDocNodeID(docID))
}

// deleteDocumentGraph 删除文档节点及其 mentions 边，并回收不再被任何文档提及的实体节点。
// 新旧两套命名空间都清理，保证升级/解绑后不残留。
func (s *GroupDocumentService) deleteDocumentGraph(groupID, docID uint) error {
	if s.gracedbDB == nil {
		return nil
	}
	g := s.gracedbDB.Graph()
	for _, docNodeID := range []string{groupDocNodeID(groupID, docID), legacyDocNodeID(docID)} {
		if _, err := g.GetNode(docNodeID); err != nil {
			continue
		}
		nodes, edges, err := g.GetNeighbors(docNodeID, graph.NeighborOptions{Direction: "out"})
		if err == nil {
			for _, e := range edges {
				_ = g.DeleteEdge(e.ID)
			}
			for _, n := range nodes {
				if n.Type == "entity" {
					s.gcEntityNode(n.ID)
				}
			}
		}
		if err := g.DeleteNode(docNodeID); err != nil {
			return err
		}
	}
	return nil
}

// gcEntityNode 在没有 mentions 边引用时删除实体节点（实体仍被其它文档引用则保留）。
func (s *GroupDocumentService) gcEntityNode(entityNodeID string) {
	if s.gracedbDB == nil {
		return
	}
	g := s.gracedbDB.Graph()
	_, mentions, err := g.GetNeighbors(entityNodeID, graph.NeighborOptions{
		Direction: "both", EdgeTypes: []string{"mentions"},
	})
	if err != nil || len(mentions) > 0 {
		return
	}
	_ = g.DeleteNode(entityNodeID)
}

// rebuildLegacyDocumentGraph 清理单个文档的旧版全局命名空间节点与 co_occurs 边。
// 只处理 doc:{id} 旧节点：删除其 mentions/co_occurs 边后，实体节点在不再被任何
// 文档提及时回收。新命名空间 g{id}: 下的节点不受影响。
func (s *GroupDocumentService) rebuildLegacyDocumentGraph(groupID, docID uint) {
	if s.gracedbDB == nil {
		return
	}
	g := s.gracedbDB.Graph()
	docNodeID := legacyDocNodeID(docID)
	if _, err := g.GetNode(docNodeID); err != nil {
		return
	}
	nodes, edges, err := g.GetNeighbors(docNodeID, graph.NeighborOptions{Direction: "both"})
	if err == nil {
		for _, e := range edges {
			_ = g.DeleteEdge(e.ID)
		}
	}
	_ = g.DeleteNode(docNodeID)
	for _, n := range nodes {
		if n.Type != "entity" {
			continue
		}
		_, coEdges, cerr := g.GetNeighbors(n.ID, graph.NeighborOptions{
			Direction: "both", EdgeTypes: []string{"co_occurs"},
		})
		if cerr == nil {
			for _, e := range coEdges {
				_ = g.DeleteEdge(e.ID)
			}
		}
		s.gcEntityNode(n.ID)
	}
}

// purgeDocumentGraphForRebuild 兼容保留：完整清理某文档的新旧两套图谱节点/边。
func (s *GroupDocumentService) purgeDocumentGraphForRebuild(groupID, docID uint) {
	if s.gracedbDB == nil {
		return
	}
	s.rebuildLegacyDocumentGraph(groupID, docID)
	if err := s.deleteDocumentGraph(groupID, docID); err != nil {
		logger.WithModule("GroupDocument").Warn("清理文档图谱失败",
			"groupID", groupID, "docID", docID, "error", err)
	}
}

// GroupGraphRebuildResult 描述一次图谱重建的结果，供管理端展示。
type GroupGraphRebuildResult struct {
	Documents int `json:"documents"`
	Nodes     int `json:"nodes"`
	Edges     int `json:"edges"`
}

// RebuildGroupKnowledgeGraph 按当前存储文档重建某群的图谱：
//   - 把旧版全局命名空间（doc:{id} / entity:{name}）迁移为群内命名空间；
//   - 删除旧版持久化的 co_occurs 全连接边（边爆炸根因）；
//   - 丢弃抽取规则下已判为噪音的实体（构建期同样会过滤，这里顺带清理存储）。
//
// 只读现有图数据，不重新解析文件正文，因此可在管理端同步触发。
func (s *GroupDocumentService) RebuildGroupKnowledgeGraph(groupID uint) (*GroupGraphRebuildResult, error) {
	if s.gracedbDB == nil {
		return nil, fmt.Errorf("图谱服务未初始化")
	}
	var docs []model.GroupDocument
	if err := s.db.Preload("File").Where("group_id = ?", groupID).
		Order("created_at DESC").Limit(groupGraphMaxDocs).Find(&docs).Error; err != nil {
		return nil, fmt.Errorf("读取群文档失败: %w", err)
	}

	result := &GroupGraphRebuildResult{Documents: len(docs)}
	type pendingDoc struct {
		id    uint
		title string
		names []string
	}
	pending := make([]pendingDoc, 0, len(docs))
	for _, d := range docs {
		names := s.documentEntityNames(groupID, d.ID)
		title := fmt.Sprintf("文档#%d", d.ID)
		if d.File.Name != "" {
			title = d.File.Name
		}
		pending = append(pending, pendingDoc{id: d.ID, title: title, names: names})
	}
	// 先写后清：全部新节点写入成功后才清理旧数据，中途失败旧图保持完整（不至于空图）。
	for _, p := range pending {
		if err := s.writeDocumentGraph(groupID, p.id, p.title, p.names); err != nil {
			return result, err
		}
		result.Nodes += 1 + len(p.names)
		result.Edges += len(p.names)
	}
	for _, p := range pending {
		s.rebuildLegacyDocumentGraph(groupID, p.id)
	}
	logger.WithModule("GroupDocument").Info("群知识图谱重建完成",
		"groupID", groupID, "documents", result.Documents, "nodes", result.Nodes, "edges", result.Edges)
	return result, nil
}

func (s *GroupDocumentService) SearchKnowledge(groupID uint, query string, topK int) ([]SearchResult, error) {
	return s.searchKnowledgeByMode(groupID, query, topK)
}

// SearchKnowledgeWithMode 检索群知识库（语义 + 词法混合召回）。
// 保留旧签名以兼容上层调用；mode/graphLight 参数在 gracedb 语义层下已无意义，忽略之。
func (s *GroupDocumentService) SearchKnowledgeWithMode(groupID uint, query string, topK int, mode string, graphLight bool) (*types.KnowledgeSearchResponse, error) {
	if s.gracedbDB == nil {
		return nil, fmt.Errorf("向量服务未初始化")
	}

	collectionName := fmt.Sprintf("group_%d", groupID)
	resp, err := s.searchHybrid(collectionName, query, topK, nil)
	if err != nil {
		return nil, fmt.Errorf("搜索知识库失败: %v", err)
	}
	return resp, nil
}

// SearchKnowledgeByDoc 在群知识库中只检索某个文档（按 metadata.doc_id 精确过滤）。
// 供"删除前校验某文档是否已向量化"或"仅在某文档范围内检索"等场景使用。
func (s *GroupDocumentService) SearchKnowledgeByDoc(groupID uint, docID uint, query string, topK int) (*types.KnowledgeSearchResponse, error) {
	if s.gracedbDB == nil {
		return nil, fmt.Errorf("向量服务未初始化")
	}
	collectionName := fmt.Sprintf("group_%d", groupID)
	filter := map[string]string{"doc_id": fmt.Sprintf("%d", docID)}
	return s.searchHybrid(collectionName, query, topK, filter)
}

// ExpandGraphKnowledge 基于向量召回的顶命中文档，扩展其知识图谱邻接上下文（GraphRAG）。
//
// 命中文档 → 该文档抽取的实体 → 这些实体 mentions 反查到的其它文档标题。
// 存储层只保留 mentions 边，实体间关系不落库，因此"跨文档关联"通过实体反查
// 实现（旧版靠 BFS 两跳，被历史 co_occurs 全连接边放大；此处反查天然有界：
// 单文档最多 8 个实体、每实体最多 3 个关联文档）。无图谱数据时返回空串，不阻断。
func (s *GroupDocumentService) ExpandGraphKnowledge(groupID uint, query string, topK int) string {
	if s.gracedbDB == nil {
		return ""
	}
	collectionName := fmt.Sprintf("group_%d", groupID)
	resp, err := s.searchHybrid(collectionName, query, topK, nil)
	if err != nil || len(resp.Results) == 0 {
		return ""
	}

	const (
		maxEntitiesPerDoc = 8
		maxRelatedPerEnt  = 3
	)
	var parts []string
	seen := make(map[string]bool)
	for _, hit := range resp.Results {
		if len(parts) >= 12 {
			break
		}
		docIDStr := hit.Metadata["doc_id"]
		if docIDStr == "" || seen[docIDStr] {
			continue
		}
		seen[docIDStr] = true
		docID, perr := strconv.ParseUint(docIDStr, 10, 32)
		if perr != nil {
			continue
		}
		names := s.documentEntityNames(groupID, uint(docID))
		if len(names) > maxEntitiesPerDoc {
			names = names[:maxEntitiesPerDoc]
		}
		if len(names) == 0 {
			continue
		}

		// 实体反查：这些实体还出现在哪些文档（供"XX 关联了哪些文档"类问答）。
		relatedSet := make(map[string]bool)
		var related []string
		for _, name := range names {
			nodes, _, nerr := s.gracedbDB.Graph().GetNeighbors(groupEntityNodeID(groupID, name),
				graph.NeighborOptions{Direction: "both", EdgeTypes: []string{"mentions"}, MaxDepth: 1})
			if nerr != nil {
				continue
			}
			for _, n := range nodes {
				if n.Type != "document" || !strings.HasPrefix(n.ID, fmt.Sprintf("g%d:doc:", groupID)) {
					continue
				}
				title := n.Properties["title"]
				if title == "" || title == hit.Title || relatedSet[title] {
					continue
				}
				relatedSet[title] = true
				related = append(related, title)
				if len(related) >= maxEntitiesPerDoc*maxRelatedPerEnt {
					break
				}
			}
			if len(related) >= maxEntitiesPerDoc*maxRelatedPerEnt {
				break
			}
		}

		line := fmt.Sprintf("[图谱] 文档 %s 关联: %s", hit.Title, strings.Join(names, "、"))
		if len(related) > 0 {
			line += fmt.Sprintf("；这些实体还出现在：%s", strings.Join(related, "、"))
		}
		parts = append(parts, line)
	}
	if len(parts) == 0 {
		return ""
	}
	return "【知识图谱】\n" + strings.Join(parts, "\n")
}

// GroupKnowledgeGraphResult 表示群聊知识图谱的归一化渲染结果，供 admin/client 两个图谱接口共用。
// 对齐分身 BuildMemoryGraph 的形态：实体/知识节点 + 关系边 + 节点反查（related）。
// TotalNodes/TotalEdges 是裁剪前的真实规模，Truncated 表示本次响应已按预算截断。
type GroupKnowledgeGraphResult struct {
	Nodes          []map[string]interface{}
	Edges          []map[string]interface{}
	TotalNodes     int
	TotalEdges     int
	KnowledgeCount int
	Truncated      bool
}

func emptyGroupKnowledgeGraph() *GroupKnowledgeGraphResult {
	return &GroupKnowledgeGraphResult{
		Nodes: []map[string]interface{}{},
		Edges: []map[string]interface{}{},
	}
}

// BuildGroupKnowledgeGraph 从存储图读取某群的知识拓扑并归一化为可渲染数据。
//
// 存储层每个文档只有 文档→实体 mentions 边；实体间关系在这里按「跨文档共现」现算：
// 只有共同出现在 >= groupGraphMinCoDoc 个文档的实体对才建边，且每个实体最多保留
// groupGraphEdgePerEntity 个邻居。这样边数从 O(实体²) 降到 O(实体×常数)，且关系
// 有实际含义（"这两个编码常在项目文档里一起出现"），不是旧版的无差别全连接。
func (s *GroupDocumentService) BuildGroupKnowledgeGraph(groupID uint, query string, maxNodes, maxEdges int) (*GroupKnowledgeGraphResult, error) {
	if s.gracedbDB == nil {
		return emptyGroupKnowledgeGraph(), nil
	}
	if maxNodes <= 0 {
		maxNodes = groupGraphDefaultMaxNodes
	}
	if maxNodes > groupGraphHardMaxNodes {
		maxNodes = groupGraphHardMaxNodes
	}
	if maxEdges <= 0 {
		maxEdges = maxNodes * groupGraphEdgesPerNode
	}
	if maxEdges > groupGraphHardMaxEdges {
		maxEdges = groupGraphHardMaxEdges
	}

	// 预加载 File 以拿到文档标题（GetDocumentsByGroup 不预加载，这里单独查一次）。
	// 单群最多取最近 groupGraphMaxDocs 篇，避免超大群一次构建拖垮接口。
	var docs []model.GroupDocument
	if err := s.db.Preload("File").Where("group_id = ?", groupID).
		Order("created_at DESC").Limit(groupGraphMaxDocs).Find(&docs).Error; err != nil {
		return nil, fmt.Errorf("读取群文档失败: %w", err)
	}

	type docInfo struct {
		id       uint
		title    string
		entities []string
	}
	infos := make([]docInfo, 0, len(docs))
	entityDocFreq := make(map[string]int)
	entityTitles := make(map[string][]string)
	for _, d := range docs {
		title := fmt.Sprintf("文档#%d", d.ID)
		if d.File.Name != "" {
			title = d.File.Name
		}
		ents := s.documentEntityNames(groupID, d.ID)
		infos = append(infos, docInfo{id: d.ID, title: title, entities: ents})
		for _, e := range ents {
			entityDocFreq[e]++
			if len(entityTitles[e]) < 8 && !containsStr(entityTitles[e], title) {
				entityTitles[e] = append(entityTitles[e], title)
			}
		}
	}

	// 跨文档共现统计：单文档内两两全连接是旧版边爆炸的根因，这里只在文档维度计数。
	pairCount := make(map[string]int)
	for _, info := range infos {
		freq := make([]string, 0, len(info.entities))
		for _, e := range info.entities {
			if entityDocFreq[e] >= groupGraphMinCoDoc {
				freq = append(freq, e)
			}
		}
		if len(freq) > groupGraphPairEntityCap {
			sort.Slice(freq, func(i, j int) bool {
				if entityDocFreq[freq[i]] != entityDocFreq[freq[j]] {
					return entityDocFreq[freq[i]] > entityDocFreq[freq[j]]
				}
				return freq[i] < freq[j]
			})
			freq = freq[:groupGraphPairEntityCap]
		}
		for i := 0; i < len(freq); i++ {
			for j := i + 1; j < len(freq); j++ {
				a, b := freq[i], freq[j]
				if a > b {
					a, b = b, a
				}
				pairCount[a+"\x00"+b]++
			}
		}
	}

	// 节点预算：文档最多占三分之一，其余按文档频次留给实体。
	totalNodes := len(infos) + len(entityDocFreq)
	docQuota := maxNodes / 3
	if docQuota < 1 {
		docQuota = 1
	}
	if docQuota > len(infos) {
		docQuota = len(infos)
	}
	keptDocs := infos[:docQuota]

	entityNames := make([]string, 0, len(entityDocFreq))
	for e := range entityDocFreq {
		entityNames = append(entityNames, e)
	}
	sort.Slice(entityNames, func(i, j int) bool {
		if entityDocFreq[entityNames[i]] != entityDocFreq[entityNames[j]] {
			return entityDocFreq[entityNames[i]] > entityDocFreq[entityNames[j]]
		}
		return entityNames[i] < entityNames[j]
	})
	entityQuota := maxNodes - len(keptDocs)
	if entityQuota > len(entityNames) {
		entityQuota = len(entityNames)
	}
	keptEntities := entityNames[:entityQuota]

	keptEntity := make(map[string]bool, len(keptEntities))
	for _, e := range keptEntities {
		keptEntity[e] = true
	}
	keptDoc := make(map[uint]bool, len(keptDocs))
	for _, d := range keptDocs {
		keptDoc[d.id] = true
	}

	nodes := make([]map[string]interface{}, 0, len(keptDocs)+len(keptEntities))
	for _, d := range keptDocs {
		nodes = append(nodes, map[string]interface{}{
			"id":    groupDocNodeID(groupID, d.id),
			"label": d.title,
			"type":  "knowledge",
			"data":  map[string]interface{}{"content": d.title, "count": 1},
		})
	}
	for _, e := range keptEntities {
		data := map[string]interface{}{
			"content": e,
			"count":   entityDocFreq[e],
		}
		if titles := entityTitles[e]; len(titles) > 0 {
			data["related"] = titles
		}
		nodes = append(nodes, map[string]interface{}{
			"id":    groupEntityNodeID(groupID, e),
			"label": e,
			"type":  "entity",
			"data":  data,
		})
	}

	type candidateEdge struct {
		from, to string
		label    string
		weight   int
	}
	candidates := make([]candidateEdge, 0)
	totalMentions := 0
	for _, info := range infos {
		totalMentions += len(info.entities)
		if !keptDoc[info.id] {
			continue
		}
		from := groupDocNodeID(groupID, info.id)
		for _, e := range info.entities {
			if keptEntity[e] {
				candidates = append(candidates, candidateEdge{from, groupEntityNodeID(groupID, e), "mentions", 1})
			}
		}
	}

	type neighborRef struct {
		name   string
		weight int
	}
	neighbors := make(map[string][]neighborRef)
	totalCoOccurs := 0
	for key, count := range pairCount {
		if count < groupGraphMinCoDoc {
			continue
		}
		totalCoOccurs++
		parts := strings.SplitN(key, "\x00", 2)
		if len(parts) != 2 {
			continue
		}
		a, b := parts[0], parts[1]
		if !keptEntity[a] || !keptEntity[b] {
			continue
		}
		neighbors[a] = append(neighbors[a], neighborRef{b, count})
		neighbors[b] = append(neighbors[b], neighborRef{a, count})
	}
	coEdges := make(map[string]int)
	for name, list := range neighbors {
		sort.Slice(list, func(i, j int) bool {
			if list[i].weight != list[j].weight {
				return list[i].weight > list[j].weight
			}
			return list[i].name < list[j].name
		})
		if len(list) > groupGraphEdgePerEntity {
			list = list[:groupGraphEdgePerEntity]
		}
		for _, it := range list {
			a, b := name, it.name
			if a > b {
				a, b = b, a
			}
			key := a + "\x00" + b
			if w, ok := coEdges[key]; !ok || it.weight > w {
				coEdges[key] = it.weight
			}
		}
	}
	// 按 key 升序遍历：map 迭代顺序随机，下面 sort.SliceStable 会保留同权重项的原始
	// 顺序，若不先定序，每次构建在边预算边界上保留的 co_occurs 边会不一样（同一份数据
	// 刷新一次拓扑就"自己变了"）。
	coKeys := make([]string, 0, len(coEdges))
	for key := range coEdges {
		coKeys = append(coKeys, key)
	}
	sort.Strings(coKeys)
	for _, key := range coKeys {
		parts := strings.SplitN(key, "\x00", 2)
		if len(parts) != 2 {
			continue
		}
		candidates = append(candidates, candidateEdge{
			groupEntityNodeID(groupID, parts[0]),
			groupEntityNodeID(groupID, parts[1]),
			"co_occurs",
			coEdges[key],
		})
	}

	// mentions 优先保留（文档与实体的挂接是理解图谱的前提），其余按权重降序。
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].label != candidates[j].label {
			return candidates[i].label == "mentions"
		}
		return candidates[i].weight > candidates[j].weight
	})
	totalEdges := totalMentions + totalCoOccurs
	if len(candidates) > maxEdges {
		candidates = candidates[:maxEdges]
	}
	edges := make([]map[string]interface{}, 0, len(candidates))
	for _, e := range candidates {
		edges = append(edges, map[string]interface{}{
			"source": e.from,
			"target": e.to,
			"label":  e.label,
			"type":   e.label,
			"weight": e.weight,
		})
	}

	truncated := totalNodes > len(nodes) || totalEdges > len(edges)

	// 可选查询节点：连到保留的知识节点（保留 admin 旧行为）。
	if query != "" {
		queryNode := map[string]interface{}{
			"id":    "query_node",
			"label": fmt.Sprintf("搜索: %s", query),
			"type":  "query",
			"data":  map[string]interface{}{"query": query},
		}
		nodes = append(nodes, queryNode)
		for _, n := range nodes {
			if t, _ := n["type"].(string); t == "knowledge" {
				edges = append(edges, map[string]interface{}{
					"source": "query_node",
					"target": n["id"],
					"label":  "related",
					"type":   "search_relation",
				})
			}
		}
	}

	return &GroupKnowledgeGraphResult{
		Nodes:          nodes,
		Edges:          edges,
		TotalNodes:     totalNodes,
		TotalEdges:     totalEdges,
		KnowledgeCount: len(keptDocs),
		Truncated:      truncated,
	}, nil
}

func containsStr(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func (s *GroupDocumentService) searchKnowledgeByMode(groupID uint, query string, topK int) ([]SearchResult, error) {
	if s.gracedbDB == nil {
		return nil, fmt.Errorf("向量服务未初始化")
	}

	collectionName := fmt.Sprintf("group_%d", groupID)
	resp, err := s.searchHybrid(collectionName, query, topK, nil)
	if err != nil {
		return nil, fmt.Errorf("搜索知识库失败: %v", err)
	}

	var searchResults []SearchResult
	for _, hit := range resp.Results {
		metadata := hit.Metadata
		if metadata == nil {
			metadata = make(map[string]string)
		}
		metadata["knowledge_id"] = hit.KnowledgeID

		searchResults = append(searchResults, SearchResult{
			Content:    hit.Snippet,
			Score:      hit.Score,
			Metadata:   metadata,
			Collection: collectionName,
			DocID:      hit.KnowledgeID,
		})
	}

	return searchResults, nil
}

// searchSemantic 纯语义检索群知识库：把 query 向量化后按余弦相似度召回切片，
// 供 searchHybrid 作为召回的一路使用，也保留对"仅语义"场景的调用。
// filter 非空时按 metadata 精确过滤（如 {"doc_id": "12"}），nil 表示不过滤。
func (s *GroupDocumentService) searchSemantic(collectionName, query string, topK int, filter map[string]string) ([]types.ScoredEmbedding, error) {
	if s.gracedbDB == nil {
		return nil, fmt.Errorf("向量服务未初始化")
	}

	queryVec, err := s.aiService.Embed(query)
	if err != nil {
		return nil, fmt.Errorf("生成查询向量失败: %v", err)
	}

	opts := types.SearchOptions{TopK: topK}
	if len(filter) > 0 {
		opts.MetadataFilter = filter
	}
	return s.gracedbDB.Search(collectionName, queryVec, opts)
}

// SearchOptions 语义 + 词法（FTS）混合召回群知识库。
//
// 收益：纯语义检索经常漏掉"精确词/编号/人名"命中（如 PRD-2024-001、张三），
// 词法能补上；反之词法对同义改写无能为力，语义能补上。两者用 RRF（倒数排名融合）
// 合并排序，比单一维度召回更全。
//
// Score 语义保持：返回给上层消费的 Score 沿用"向量余弦相似度"（0-1），避免前端
// 知识来源展示（score*100%）因 RRF 分数尺度变小而退化。FTS 命中但语义未命中的文档，
// 无余弦分可用，取中性值 0.5 兜底展示。
//
// filter 非空时按 metadata 精确过滤（如 {"doc_id": "12"}），语义与 FTS 两路都约束在
// 该范围内，nil 表示不过滤。
func (s *GroupDocumentService) searchHybrid(collectionName, query string, topK int, filter map[string]string) (*types.KnowledgeSearchResponse, error) {
	if s.gracedbDB == nil {
		return nil, fmt.Errorf("向量服务未初始化")
	}
	if topK <= 0 {
		topK = 5
	}
	// 多取一些候选便于 RRF 重排后仍有足够结果
	fetchK := topK * 3

	// 路 1：语义向量检索（Score 为余弦相似度 0-1）
	semanticRes, semErr := s.searchSemantic(collectionName, query, fetchK, filter)
	if semErr != nil {
		semanticRes = nil
	}

	// 路 2：中文词法 FTS 检索
	ftsRes, ftsErr := s.gracedbDB.SearchFTSWithContent(collectionName, query, fetchK)
	if ftsErr != nil {
		ftsRes = nil
	}
	// SearchFTSWithContent 不带 filter 参数，过滤由这里后置完成：
	// 仅当 filter 非空时，剔除 metadata 不匹配的 FTS 命中。
	if len(filter) > 0 && len(ftsRes) > 0 {
		kept := ftsRes[:0]
		for _, r := range ftsRes {
			if metadataMatchFilter(r.Metadata, filter) {
				kept = append(kept, r)
			}
		}
		ftsRes = kept
	}

	if len(semanticRes) == 0 && len(ftsRes) == 0 {
		return &types.KnowledgeSearchResponse{Query: query, Results: []types.KnowledgeSearchHit{}}, nil
	}

	// 仅一路有结果时直接使用，避免无谓 RRF
	if len(semanticRes) == 0 {
		ftsRes = hybridDisplayScores(ftsRes, nil, ftsRes)
		return s.hitsFromScored(collectionName, query, ftsRes), nil
	}
	if len(ftsRes) == 0 {
		return s.hitsFromScored(collectionName, query, semanticRes), nil
	}

	merged := mergeRRF(semanticRes, ftsRes, topK)
	// 尽量保留语义余弦分用于展示；FTS 独占的按排名给分
	display := hybridDisplayScores(merged, semanticRes, ftsRes)
	return s.hitsFromScored(collectionName, query, display), nil
}

// hitsFromScored 把 ScoredEmbedding 列表转成 gracedb 的 KnowledgeSearchResponse 结构。
func (s *GroupDocumentService) hitsFromScored(collectionName, query string, scored []types.ScoredEmbedding) *types.KnowledgeSearchResponse {
	results := make([]types.KnowledgeSearchHit, 0, len(scored))
	for _, se := range scored {
		metadata := se.Metadata
		if metadata == nil {
			metadata = make(map[string]string)
		}
		title := metadata["title"]
		results = append(results, types.KnowledgeSearchHit{
			KnowledgeID: se.DocID,
			Title:       title,
			Snippet:     se.Content,
			Score:       float64(se.Score),
			Metadata:    metadata,
		})
	}
	return &types.KnowledgeSearchResponse{Query: query, Results: results}
}

// mergeRRF 对语义/词法两路结果做倒数排名融合（Reciprocal Rank Fusion）后取 topK。
func mergeRRF(semantic, fts []types.ScoredEmbedding, topK int) []types.ScoredEmbedding {
	const k = 60.0
	type entry struct {
		emb types.ScoredEmbedding
		rrf float64
	}
	order := make([]string, 0)
	scores := make(map[string]*entry)

	addRank := func(list []types.ScoredEmbedding) {
		for i, emb := range list {
			id := emb.ID
			if id == "" {
				id = emb.DocID
			}
			score := 1.0 / (k + float64(i+1))
			if e, ok := scores[id]; ok {
				e.rrf += score
			} else {
				scores[id] = &entry{emb: emb, rrf: score}
				order = append(order, id)
			}
		}
	}
	addRank(semantic)
	addRank(fts)

	// 稳定排序：分数高者优先，同分保持语义在前
	weighted := make([]entry, 0, len(order))
	for _, id := range order {
		weighted = append(weighted, *scores[id])
	}
	for i := 1; i < len(weighted); i++ {
		for j := 0; j < len(weighted)-i; j++ {
			if weighted[j].rrf < weighted[j+1].rrf {
				weighted[j], weighted[j+1] = weighted[j+1], weighted[j]
			}
		}
	}
	if len(weighted) > topK {
		weighted = weighted[:topK]
	}
	out := make([]types.ScoredEmbedding, 0, len(weighted))
	for _, w := range weighted {
		out = append(out, w.emb)
	}
	return out
}

// hybridDisplayScores 把 RRF 融合后的展示分数还原为对用户有意义的 0-1 值：
//   - 语义+词法双路命中：余弦分 + 0.08 加成（上限 1.0），双路确认 = 更高置信度
//   - 仅语义命中：保持原始余弦相似度
//   - 仅词法命中：用归一化 BM25 分（0.5~0.8），无 BM25 分时回退到排名估算
//
// RRF 分数尺度极小（~0.016）不适合直接展示，故还原为余弦语义；FTS 独占命中
// 优先使用 gracedb 返回的 BM25 原始分归一化，比纯排名估算更精确。
func hybridDisplayScores(scored, semantic, fts []types.ScoredEmbedding) []types.ScoredEmbedding {
	semScore := make(map[string]float32, len(semantic))
	for _, se := range semantic {
		id := se.ID
		if id == "" {
			id = se.DocID
		}
		semScore[id] = se.Score
	}
	// FTS 分数映射（BM25 原始分）+ 最大值（用于归一化到 0.5~0.8）
	ftsScoreMap := make(map[string]float32, len(fts))
	var maxFTS float32
	for _, se := range fts {
		id := se.ID
		if id == "" {
			id = se.DocID
		}
		ftsScoreMap[id] = se.Score
		if se.Score > maxFTS {
			maxFTS = se.Score
		}
	}
	ftsRank := make(map[string]int, len(fts))
	for i, se := range fts {
		id := se.ID
		if id == "" {
			id = se.DocID
		}
		ftsRank[id] = i
	}
	ftsTotal := len(fts)

	out := make([]types.ScoredEmbedding, 0, len(scored))
	for _, se := range scored {
		id := se.ID
		if id == "" {
			id = se.DocID
		}
		semVal, inSem := semScore[id]
		_, inFTS := ftsScoreMap[id]
		rank := ftsRank[id]

		switch {
		case inSem && inFTS:
			// 双路命中：余弦分 + 加成，截断 1.0
			se.Score = semVal + 0.08
			if se.Score > 1.0 {
				se.Score = 1.0
			}
		case inSem:
			// 仅语义：保持原始余弦分
			se.Score = semVal
		case inFTS:
			// 仅词法：优先用归一化 BM25 分，回退到排名估算
			if maxFTS > 0 {
				se.Score = 0.5 + 0.3*ftsScoreMap[id]/maxFTS
			} else {
				se.Score = 0.8 - 0.3*float32(rank)/float32(ftsTotal)
			}
			if se.Score < 0.5 {
				se.Score = 0.5
			}
		default:
			se.Score = 0.5
		}
		out = append(out, se)
	}
	return out
}

// metadataMatchFilter 判断 embedding 的 metadata 是否精确满足 filter 的每个 key=value。
// 用于对不带 filter 参数的 FTS 检索结果做后置过滤（语义检索已在 SearchOptions.MetadataFilter
// 内层过滤，此处仅补 FTS 一路）。
func metadataMatchFilter(metadata, filter map[string]string) bool {
	for k, v := range filter {
		if metadata[k] != v {
			return false
		}
	}
	return true
}

func (s *GroupDocumentService) DeleteGroupVectors(groupID uint) error {
	if s.gracedbDB == nil {
		return nil
	}

	collectionName := fmt.Sprintf("group_%d", groupID)
	if err := s.gracedbDB.DeleteCollection(collectionName); err != nil {
		logger.WithModule("GroupDocument").Warn("清空群知识向量失败", "groupID", groupID, "error", err)
	}

	s.db.Where("group_id = ?", groupID).Delete(&model.DocumentProcessStatus{})
	logger.WithModule("GroupDocument").Info("群知识向量清理完成", "groupID", groupID)
	return nil
}

// ExtractTextForContext 读取指定文件（model.File）的正文并返回（不入库、不建向量）。
// 供群 AI 回复时按引用消息读取被引用文件正文注入上下文。
// 返回文件名与解析后的正文；内容非文档类型或解析失败时返回错误，由调用方优雅降级。
func (s *GroupDocumentService) ExtractTextForContext(fileID uint) (name string, text string, err error) {
	var file model.File
	if err := s.db.First(&file, fileID).Error; err != nil {
		return "", "", err
	}
	if s.store == nil {
		return file.Name, "", fmt.Errorf("存储未初始化")
	}
	// 大小护栏：进入内存/解析前直接拦截超大文件，避免整个大文件进内存再解析（注：正文只需 4000 字符）。
	// docx/xlsx/pptx 为 zip 容器，Size 是压缩包大小，解析文本量不可预知，故用相对宽松的 20MB 阈值。
	if file.Size > quoteMaxFileSize {
		return file.Name, "", fmt.Errorf("%w: 文件大小 %d 字节超过 %d 上限", ErrQuotedFileTooLarge, file.Size, quoteMaxFileSize)
	}
	if s.parser == nil {
		s.parser = &DocumentParser{}
	}
	reader, err := s.store.GetByPath(context.Background(), file.StoragePath)
	if err != nil {
		return file.Name, "", fmt.Errorf("读取文件失败: %w", err)
	}
	tmpFile, err := os.CreateTemp("", "qim-quote-*"+filepath.Ext(file.Name))
	if err != nil {
		reader.Close()
		return file.Name, "", fmt.Errorf("创建临时文件失败: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)
	if _, err := io.Copy(tmpFile, reader); err != nil {
		reader.Close()
		tmpFile.Close()
		return file.Name, "", fmt.Errorf("写入临时文件失败: %w", err)
	}
	reader.Close()
	tmpFile.Close()

	text, err = s.parser.Parse(tmpPath)
	if err != nil {
		return file.Name, "", fmt.Errorf("文档解析失败: %w", err)
	}
	return file.Name, text, nil
}

// ImageURLForContext 读取指定图片（model.File）原始字节并转为 base64 data URL，供群 AI 多模态识别。
// 不入库、不落临时文件，直接流式读内存；超过 quoteMaxImageSize（5MB）时返回哨兵错误
// ErrQuotedImageTooLarge，由调用方区分"图片过大"与"读取失败"两种看不了图片的场景。
func (s *GroupDocumentService) ImageURLForContext(fileID uint) (name string, dataURL string, err error) {
	var file model.File
	if err := s.db.First(&file, fileID).Error; err != nil {
		return "", "", err
	}
	if s.store == nil {
		return file.Name, "", fmt.Errorf("存储未初始化")
	}
	// 大小护栏：进入内存前直接拦截超大图片，避免整个大图 base64 膨胀后进 prompt。
	// 与 ExtractedTextForContext 一致，按 file.Size 先行拦截，不读入再判断。
	if file.Size > quoteMaxImageSize {
		return file.Name, "", fmt.Errorf("%w: 图片大小 %d 字节超过 %d 上限", ErrQuotedImageTooLarge, file.Size, quoteMaxImageSize)
	}
	reader, err := s.store.GetByPath(context.Background(), file.StoragePath)
	if err != nil {
		return file.Name, "", fmt.Errorf("读取图片失败: %w", err)
	}
	defer reader.Close()
	data, err := io.ReadAll(reader)
	if err != nil {
		return file.Name, "", fmt.Errorf("读取图片字节失败: %w", err)
	}
	// 读入后二次护栏：防 Store 记录的 Size 与实际字节不一致导致的意外大图。
	if len(data) > int(quoteMaxImageSize) {
		return file.Name, "", fmt.Errorf("%w: 图片实际 %d 字节超过 %d 上限", ErrQuotedImageTooLarge, len(data), quoteMaxImageSize)
	}
	contentType := mime.TypeByExtension(filepath.Ext(file.Name))
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}
	return file.Name, "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(data), nil
}

// isImageDocument 判断文档 MIME 是否为图片类型：先去掉 "; charset=..." 后缀，
// 再按 image/ 前缀匹配。直接上传的图片经此判断走视觉 OCR 入库。
func isImageDocument(mimeType string) bool {
	base := mimeType
	if idx := strings.Index(base, ";"); idx > 0 {
		base = strings.TrimSpace(base[:idx])
	}
	return strings.HasPrefix(base, "image/")
}

// readImageAsDataURL 读取本地文件为 base64 data URL，供图片 OCR 等视觉任务使用。
// 超过 quoteMaxImageSize（5MB）时返回哨兵错误，避免超大图 base64 膨胀后进 prompt。
func readImageAsDataURL(path, fileName string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("读取图片失败: %w", err)
	}
	if len(data) > int(quoteMaxImageSize) {
		return "", fmt.Errorf("%w: 图片实际 %d 字节超过 %d 上限", ErrQuotedImageTooLarge, len(data), quoteMaxImageSize)
	}
	contentType := mime.TypeByExtension(filepath.Ext(fileName))
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}
	return "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(data), nil
}

// ocrImageToText 读取已落地的本地临时图片并调用视觉模型逐字识别其中的文字，
// 返回识别出的原文文本（供后续切片+向量化）。未配置视觉路由/读图失败时返回错误，
// 由调用方落 failed 状态与明确文案，而不是卡在 pending 让用户无法重试。
func (s *GroupDocumentService) ocrImageToText(tmpPath, fileName string) (string, error) {
	if s.aiService == nil {
		return "", fmt.Errorf("AI 服务未初始化")
	}
	dataURL, err := readImageAsDataURL(tmpPath, fileName)
	if err != nil {
		return "", err
	}
	if !s.aiService.HasVisionRoute() {
		return "", fmt.Errorf("当前未配置支持视觉的 AI 模型，无法识别图片文字（请在 AI 设置中启用视觉/多模态路由）")
	}
	ocrText, err := s.aiService.GetCompletion(ai.TaskTypeVision, []ai.Message{
		{Role: "user", Content: "请逐字识别图片中的全部文字，输出原文文本，不要添加任何解释。", ImageURL: dataURL},
	})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(ocrText), nil
}

func (s *GroupDocumentService) RetryDocument(groupDocID uint) error {
	var status model.DocumentProcessStatus
	s.db.Where("group_doc_id = ?", groupDocID).Order("created_at DESC").First(&status)

	if status.Status == "completed" {
		return fmt.Errorf("文档已成功处理，无需重试")
	}

	s.db.Model(&status).Updates(map[string]interface{}{
		"status": "pending",
		"error":  "",
	})

	return s.ProcessDocument(groupDocID)
}

func (s *GroupDocumentService) BatchProcessDocuments(groupDocIDs []uint) (map[string]interface{}, error) {
	results := make(map[string]interface{})
	success := 0
	failed := 0
	var errors []string

	for _, id := range groupDocIDs {
		err := s.ProcessDocument(id)
		if err != nil {
			failed++
			errors = append(errors, fmt.Sprintf("文档 %d: %v", id, err))
		} else {
			success++
		}
	}

	results["success"] = success
	results["failed"] = failed
	if len(errors) > 0 {
		results["errors"] = errors
	}

	return results, nil
}

func (s *GroupDocumentService) BatchRetryDocuments(groupDocIDs []uint) (map[string]interface{}, error) {
	results := make(map[string]interface{})
	success := 0
	failed := 0
	var errors []string

	for _, id := range groupDocIDs {
		var status model.DocumentProcessStatus
		s.db.Where("group_doc_id = ?", id).Order("created_at DESC").First(&status)

		if status.Status == "completed" {
			continue
		}

		s.db.Model(&status).Updates(map[string]interface{}{
			"status": "pending",
			"error":  "",
		})

		err := s.ProcessDocument(id)
		if err != nil {
			failed++
			errors = append(errors, fmt.Sprintf("文档 %d: %v", id, err))
		} else {
			success++
		}
	}

	results["success"] = success
	results["failed"] = failed
	if len(errors) > 0 {
		results["errors"] = errors
	}

	return results, nil
}
