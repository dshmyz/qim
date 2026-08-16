package service

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
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
		embedding, err := s.aiService.Embed(chunk.Content)
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
	}
	return nil
}

// buildDocumentGraph 为单个群文档建立实体关系图（GraphRAG MVP）。
//
// 写入方式（幂等）：UpsertNode/UpsertEdge 覆盖式更新，重复处理同一文档不会产生重复节点。
// 结构：一个 document 节点 + 从正文抽取的实体节点，文档→实体 "mentions" 边，
// 实体间在同一文档内共现的 "co_occurs" 边。
//
// MVP 采用确定性抽取（无额外 LLM 调用），仅识别文档编码等明确 token；
// LLM 级实体抽取作为后续增强，不在本次范围。
func (s *GroupDocumentService) buildDocumentGraph(groupID, docID uint, title, text string) error {
	if s.gracedbDB == nil {
		return nil
	}
	g := s.gracedbDB.Graph()

	docNodeID := fmt.Sprintf("doc:%d", docID)
	docNode := &graph.GraphNode{
		ID:     docNodeID,
		Type:   "document",
		Labels: []string{"group_document"},
		Properties: map[string]string{
			"group_id": fmt.Sprintf("%d", groupID),
			"doc_id":   fmt.Sprintf("%d", docID),
			"title":    title,
		},
	}
	if err := g.UpsertNode(docNode); err != nil {
		return fmt.Errorf("写入文档节点失败: %w", err)
	}

	entities := extractGraphEntities(text)
	entitySet := make(map[string]bool)
	for _, e := range entities {
		if entitySet[e] {
			continue
		}
		entitySet[e] = true

		// 实体节点
		entityID := "entity:" + e
		if err := g.UpsertNode(&graph.GraphNode{
			ID:     entityID,
			Type:   "entity",
			Labels: []string{"extracted"},
			Properties: map[string]string{
				"name":   e,
				"doc_id": fmt.Sprintf("%d", docID),
			},
		}); err != nil {
			return fmt.Errorf("写入实体节点失败: %w", err)
		}

		// 文档 → 实体 mentions 边
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

	// 实体共现边（同一文档内的实体彼此 co_occurs），便于关系问答联动多个实体
	entList := make([]string, 0, len(entitySet))
	for e := range entitySet {
		entList = append(entList, e)
	}
	for i := 0; i < len(entList); i++ {
		for j := i + 1; j < len(entList); j++ {
			a, b := "entity:"+entList[i], "entity:"+entList[j]
			if err := g.UpsertEdge(&graph.GraphEdge{
				ID:         fmt.Sprintf("%s-co-%s", a, b),
				FromNodeID: a,
				ToNodeID:   b,
				Type:       "co_occurs",
				Weight:     1.0,
			}); err != nil {
				return fmt.Errorf("写入共现边失败: %w", err)
			}
		}
	}

	logger.WithModule("GroupDocument").Info("文档知识图谱构建完成",
		"doc_id", docID, "entities", len(entities))
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
		if len(m) < 3 || len(m) > 40 {
			continue
		}
		if !seen[m] {
			seen[m] = true
			out = append(out, m)
		}
	}
	return out
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
// 流程：searchHybrid 取 topK 命中 → 对顶命中的 doc 节点做两跳 GraphBFS → 把命中的
// 实体/关联文档格式化为文本。用于"关系问答"（如 PRD-2024-001 关联了哪些人/文档），
// 普通向量检索难答，"实体→邻接"补上。无图谱数据时返回空串，不阻断正常答复。
func (s *GroupDocumentService) ExpandGraphKnowledge(groupID uint, query string, topK int) string {
	if s.gracedbDB == nil {
		return ""
	}
	collectionName := fmt.Sprintf("group_%d", groupID)
	resp, err := s.searchHybrid(collectionName, query, topK, nil)
	if err != nil || len(resp.Results) == 0 {
		return ""
	}

	g := s.gracedbDB.Graph()
	var parts []string
	seen := make(map[string]bool)
	for _, hit := range resp.Results {
		if len(parts) >= 12 {
			break
		}
		docID := hit.Metadata["doc_id"]
		if docID == "" {
			continue
		}
		docNodeID := "doc:" + docID
		if seen[docNodeID] {
			continue
		}
		seen[docNodeID] = true

		res, err := g.BFS(docNodeID, graph.NeighborOptions{MaxDepth: 2})
		if err != nil || res == nil || len(res.Nodes) == 0 {
			continue
		}
		// 汇总去重后的实体/关联节点名
		var names []string
		for _, n := range res.Nodes {
			if n.ID == docNodeID {
				continue
			}
			label := n.ID
			if v, ok := n.Properties["name"]; ok {
				label = v
			} else if v, ok := n.Properties["title"]; ok {
				label = "文档:" + v
			}
			if !containsStr(names, label) {
				names = append(names, label)
			}
		}
		if len(names) > 0 {
			parts = append(parts, fmt.Sprintf("[图谱] 文档 %s 关联: %s",
				hit.Title, strings.Join(names, "、")))
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return "【知识图谱】\n" + strings.Join(parts, "\n")
}

// GroupKnowledgeGraphResult 表示群聊知识图谱的归一化渲染结果，供 admin/client 两个图谱接口共用。
// 对齐分身 BuildMemoryGraph 的形态：实体/知识节点 + 关系边 + 节点反查（related）。
type GroupKnowledgeGraphResult struct {
	Nodes          []map[string]interface{}
	Edges          []map[string]interface{}
	TotalNodes     int
	TotalEdges     int
	KnowledgeCount int
}

func emptyGroupKnowledgeGraph() *GroupKnowledgeGraphResult {
	return &GroupKnowledgeGraphResult{
		Nodes: []map[string]interface{}{},
		Edges: []map[string]interface{}{},
	}
}

// BuildGroupKnowledgeGraph 从 gracedb 存储图读取某群的知识拓扑并归一化为可渲染数据（GraphRAG）。
//
// 相比旧实现（直接平铺向量块、无关系边），这里真正体现"文档→实体"拓扑：取该群全部文档，
// 对每个 doc 节点做两跳 BFS 采集邻接实体与 mentions/co_occurs 边，并携带实体反查
// （该实体出现在哪些文档，对应分身 memories[].terms 的关联回查）。无图谱数据时返回
// 空结构，不报错、不阻断。
func (s *GroupDocumentService) BuildGroupKnowledgeGraph(groupID uint, query string, maxNodes int) (*GroupKnowledgeGraphResult, error) {
	if s.gracedbDB == nil {
		return emptyGroupKnowledgeGraph(), nil
	}
	if maxNodes <= 0 {
		maxNodes = 50
	}

	// 预加载 File 以拿到文档标题（GetDocumentsByGroup 不预加载，这里单独查一次）
	var docs []model.GroupDocument
	if err := s.db.Preload("File").Where("group_id = ?", groupID).Find(&docs).Error; err != nil {
		return nil, fmt.Errorf("读取群文档失败: %w", err)
	}

	g := s.gracedbDB.Graph()

	// 节点聚合：以存储图节点 ID（doc:{id} / entity:{name}）为键
	type aggNode struct {
		id      string
		label   string
		typ     string
		count   int
		related []string // 实体反查：所在文档标题
	}
	nodeMap := make(map[string]*aggNode)
	var nodeOrder []string
	addNode := func(id, label, typ string) *aggNode {
		a, ok := nodeMap[id]
		if !ok {
			a = &aggNode{id: id, label: label, typ: typ}
			nodeMap[id] = a
			nodeOrder = append(nodeOrder, id)
		}
		return a
	}

	// 边聚合：以 "from|to" 为键去重并累计权重
	type aggEdge struct {
		from, to string
		label    string
		weight   float64
	}
	edgeMap := make(map[string]*aggEdge)
	var edgeOrder []string
	addEdge := func(from, to, label string, w float64) {
		key := from + "\x00" + to
		e, ok := edgeMap[key]
		if !ok {
			e = &aggEdge{from: from, to: to, label: label}
			edgeMap[key] = e
			edgeOrder = append(edgeOrder, key)
		}
		e.weight += w
	}

	for _, d := range docs {
		title := fmt.Sprintf("文档#%d", d.ID)
		if d.File.Name != "" {
			title = d.File.Name
		}
		docNodeID := fmt.Sprintf("doc:%d", d.ID)
		docAgg := addNode(docNodeID, title, "knowledge")
		docAgg.count++

		res, err := g.BFS(docNodeID, graph.NeighborOptions{MaxDepth: 2})
		if err != nil || res == nil {
			continue
		}
		for _, n := range res.Nodes {
			if n.ID == docNodeID || n.Type != "entity" {
				continue
			}
			name := n.ID
			if v, ok := n.Properties["name"]; ok && v != "" {
				name = v
			}
			a := addNode(n.ID, name, "entity")
			a.count++
			if !containsStr(a.related, title) {
				a.related = append(a.related, title)
			}
			addEdge(docNodeID, n.ID, "mentions", 1)
		}
		for _, e := range res.Edges {
			if e.Type == "co_occurs" {
				addEdge(e.FromNodeID, e.ToNodeID, "co_occurs", e.Weight)
			}
		}
	}

	// 裁剪到 maxNodes：优先保留文档节点，再按出现顺序补实体节点
	kept := make(map[string]bool)
	for _, id := range nodeOrder {
		if len(kept) >= maxNodes {
			break
		}
		kept[id] = true
	}
	nodes := make([]map[string]interface{}, 0, len(kept))
	for _, id := range nodeOrder {
		if !kept[id] {
			continue
		}
		a := nodeMap[id]
		data := map[string]interface{}{
			"content": a.label,
			"count":   a.count,
		}
		if a.typ == "entity" && len(a.related) > 0 {
			data["related"] = a.related
		}
		nodes = append(nodes, map[string]interface{}{
			"id":    a.id,
			"label": a.label,
			"type":  a.typ,
			"data":  data,
		})
	}

	// 只有两端都保留的边才输出
	edges := make([]map[string]interface{}, 0)
	for _, key := range edgeOrder {
		e := edgeMap[key]
		if !kept[e.from] || !kept[e.to] {
			continue
		}
		edges = append(edges, map[string]interface{}{
			"source": e.from,
			"target": e.to,
			"label":  e.label,
			"type":   e.label,
			"weight": e.weight,
		})
	}

	// 可选查询节点：连到所有保留的知识节点（保留 admin 旧行为）
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

	knowledgeCount := 0
	for _, n := range nodes {
		if t, _ := n["type"].(string); t == "knowledge" {
			knowledgeCount++
		}
	}

	return &GroupKnowledgeGraphResult{
		Nodes:          nodes,
		Edges:          edges,
		TotalNodes:     len(nodes),
		TotalEdges:     len(edges),
		KnowledgeCount: knowledgeCount,
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
