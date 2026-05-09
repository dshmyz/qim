package service

import (
	"context"
	"fmt"
	"log"

	"qim-server/model"

	"gorm.io/gorm"
)

type GroupDocumentService struct {
	db          *gorm.DB
	vectorSvc   *VectorService
	aiService   interface{ Embed(text string) ([]float32, error) }
	parser      *DocumentParser
}

func NewGroupDocumentService(db *gorm.DB) *GroupDocumentService {
	return &GroupDocumentService{db: db}
}

func (s *GroupDocumentService) SetVectorServices(vectorSvc *VectorService, aiService interface{ Embed(text string) ([]float32, error) }) {
	s.vectorSvc = vectorSvc
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
		} else {
			result["process_status"] = "pending"
		}

		results = append(results, result)
	}

	return results, nil
}

func (s *GroupDocumentService) ProcessDocument(groupDocID uint) error {
	var doc model.GroupDocument
	if err := s.db.First(&doc, groupDocID).Error; err != nil {
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

	var chunks []Chunk
	if doc.File.MimeType == "text/plain" || doc.File.MimeType == "text/markdown" {
		chunks = SplitMarkdownByHeading(text)
	} else {
		parts := SplitBySize(text, 800)
		for _, part := range parts {
			chunks = append(chunks, Chunk{Content: part, Title: doc.File.Name})
		}
	}

	if len(chunks) == 0 {
		chunks = []Chunk{{Content: text, Title: doc.File.Name}}
	}

	collectionName := fmt.Sprintf("group_%d", doc.GroupID)

	chunkCount := 0
	for i, chunk := range chunks {
		if len(chunk.Content) < 20 {
			continue
		}

		embedding, err := s.aiService.Embed(chunk.Content)
		if err != nil {
			log.Printf("[GroupDocument] 生成向量失败 for chunk %d: %v", i, err)
			continue
		}

		docID := fmt.Sprintf("group_%d_doc_%d_chunk_%d", doc.GroupID, doc.ID, i)
		if err := s.vectorSvc.AddVector(context.Background(), collectionName, docID, embedding, chunk.Content, map[string]string{
			"group_id": fmt.Sprintf("%d", doc.GroupID),
			"doc_id":   fmt.Sprintf("%d", doc.ID),
			"file_id":  fmt.Sprintf("%d", doc.FileID),
			"chunk_id": fmt.Sprintf("%d", i),
			"title":    doc.File.Name,
		}); err != nil {
			log.Printf("[GroupDocument] 存储向量失败 for chunk %d: %v", i, err)
			continue
		}

		chunkCount++
	}

	s.db.Model(&status).Updates(map[string]interface{}{
		"status":      "completed",
		"chunk_count": chunkCount,
	})

	log.Printf("[GroupDocument] 文档处理完成: doc_id=%d, chunks=%d", doc.ID, chunkCount)
	return nil
}

func (s *GroupDocumentService) SearchKnowledge(groupID uint, query string, topK int) ([]SearchResult, error) {
	if s.vectorSvc == nil || s.aiService == nil {
		return nil, fmt.Errorf("向量服务未初始化")
	}

	queryVector, err := s.aiService.Embed(query)
	if err != nil {
		return nil, fmt.Errorf("生成查询向量失败: %v", err)
	}

	collectionName := fmt.Sprintf("group_%d", groupID)
	results, err := s.vectorSvc.Search(context.Background(), collectionName, queryVector, topK)
	if err != nil {
		return nil, fmt.Errorf("搜索知识库失败: %v", err)
	}

	var searchResults []SearchResult
	for _, r := range results {
		searchResults = append(searchResults, ScoredEmbeddingToSearchResult(r))
	}

	return searchResults, nil
}

func (s *GroupDocumentService) DeleteGroupVectors(groupID uint) error {
	collectionName := fmt.Sprintf("group_%d", groupID)
	return s.vectorSvc.DeleteByCollection(context.Background(), collectionName)
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
