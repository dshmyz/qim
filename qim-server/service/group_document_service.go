package service

import (
	"context"
	"fmt"
	"log"

	"qim-server/model"

	"github.com/liliang-cn/cortexdb/v2/pkg/cortexdb"
	"gorm.io/gorm"
)

type GroupDocumentService struct {
	db       *gorm.DB
	cortexDB *cortexdb.DB
	parser   *DocumentParser
}

func NewGroupDocumentService(db *gorm.DB) *GroupDocumentService {
	return &GroupDocumentService{db: db}
}

func (s *GroupDocumentService) SetVectorServices(vectorSvc *VectorService) {
	s.cortexDB = vectorSvc.GetDB()
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
		} else {
			result["process_status"] = "pending"
		}

		results = append(results, result)
	}

	return results, nil
}

func (s *GroupDocumentService) ProcessDocument(groupDocID uint) error {
	if s.cortexDB == nil {
		return fmt.Errorf("向量服务未初始化")
	}

	var doc model.GroupDocument
	if err := s.db.Preload("File").First(&doc, groupDocID).Error; err != nil {
		return fmt.Errorf("文档不存在")
	}

	var status model.DocumentProcessStatus
	s.db.Where("group_doc_id = ?", groupDocID).Order("created_at DESC").First(&status)

	if status.ID == 0 {
		status = model.DocumentProcessStatus{
			GroupDocID: groupDocID,
			Status:     "pending",
		}
		s.db.Create(&status)
	}

	s.db.Model(&status).Updates(map[string]interface{}{
		"status": "processing",
	})

	filePath := doc.File.StoragePath
	text, err := s.parser.Parse(filePath)
	if err != nil {
		s.db.Model(&status).Updates(map[string]interface{}{
			"status": "failed",
			"error":  fmt.Sprintf("文档解析失败: %v", err),
		})
		return err
	}

	if text == "" || len(text) < 10 {
		s.db.Model(&status).Updates(map[string]interface{}{
			"status": "failed",
			"error":  "文档内容为空或太短",
		})
		return fmt.Errorf("文档内容为空")
	}

	collectionName := fmt.Sprintf("group_%d", doc.GroupID)
	knowledgeID := fmt.Sprintf("group_%d_doc_%d", doc.GroupID, doc.ID)

	ctx := context.Background()
	resp, err := s.cortexDB.SaveKnowledge(ctx, cortexdb.KnowledgeSaveRequest{
		KnowledgeID: knowledgeID,
		Title:       doc.File.Name,
		Content:     text,
		Collection:  collectionName,
		ChunkSize:   800,
		Metadata: map[string]string{
			"group_id": fmt.Sprintf("%d", doc.GroupID),
			"doc_id":   fmt.Sprintf("%d", doc.ID),
			"file_id":  fmt.Sprintf("%d", doc.FileID),
			"title":    doc.File.Name,
		},
	})

	if err != nil {
		s.db.Model(&status).Updates(map[string]interface{}{
			"status": "failed",
			"error":  fmt.Sprintf("知识存储失败: %v", err),
		})
		return err
	}

	chunkCount := len(resp.Knowledge.ChunkIDs)
	s.db.Model(&status).Updates(map[string]interface{}{
		"status":      "completed",
		"chunk_count": chunkCount,
	})

	log.Printf("[GroupDocument] 文档处理完成: doc_id=%d, knowledge_id=%s, chunks=%d, entities=%d",
		doc.ID, knowledgeID, chunkCount, len(resp.EntityNodeIDs))
	return nil
}

func (s *GroupDocumentService) SearchKnowledge(groupID uint, query string, topK int) ([]SearchResult, error) {
	return s.searchKnowledgeByMode(groupID, query, topK, cortexdb.RetrievalModeAuto)
}

func (s *GroupDocumentService) SearchKnowledgeWithMode(groupID uint, query string, topK int, mode string, graphLight bool) (*cortexdb.KnowledgeSearchResponse, error) {
	if s.cortexDB == nil {
		return nil, fmt.Errorf("向量服务未初始化")
	}

	collectionName := fmt.Sprintf("group_%d", groupID)
	resp, err := s.cortexDB.SearchKnowledge(context.Background(), cortexdb.KnowledgeSearchRequest{
		Query:         query,
		Collection:    collectionName,
		TopK:          topK,
		RetrievalMode: mode,
		GraphLight:    graphLight,
	})
	if err != nil {
		return nil, fmt.Errorf("搜索知识库失败: %v", err)
	}
	return resp, nil
}

func (s *GroupDocumentService) searchKnowledgeByMode(groupID uint, query string, topK int, mode string) ([]SearchResult, error) {
	if s.cortexDB == nil {
		return nil, fmt.Errorf("向量服务未初始化")
	}

	collectionName := fmt.Sprintf("group_%d", groupID)
	resp, err := s.cortexDB.SearchKnowledge(context.Background(), cortexdb.KnowledgeSearchRequest{
		Query:         query,
		Collection:    collectionName,
		TopK:          topK,
		RetrievalMode: mode,
	})

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

func (s *GroupDocumentService) DeleteGroupVectors(groupID uint) error {
	if s.cortexDB == nil {
		return nil
	}

	var docs []model.GroupDocument
	s.db.Where("group_id = ?", groupID).Find(&docs)

	for _, doc := range docs {
		knowledgeID := fmt.Sprintf("group_%d_doc_%d", groupID, doc.ID)
		s.cortexDB.DeleteKnowledge(context.Background(), cortexdb.KnowledgeDeleteRequest{
			KnowledgeID: knowledgeID,
		})
	}

	s.db.Where("group_id = ?", groupID).Delete(&model.DocumentProcessStatus{})
	log.Printf("[GroupDocument] 群 %d 知识向量清理完成", groupID)
	return nil
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
