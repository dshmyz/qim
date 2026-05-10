package service

import (
	"context"
	"fmt"
	"log"

	"qim-server/ai"
)

// NoteVectorService 笔记向量化和检索服务
type NoteVectorService struct {
	vectorSvc *VectorService
	aiService *ai.AIService
}

// NewNoteVectorService 创建笔记向量服务实例
func NewNoteVectorService(vectorSvc *VectorService, aiService *ai.AIService) *NoteVectorService {
	return &NoteVectorService{
		vectorSvc: vectorSvc,
		aiService: aiService,
	}
}

// VectorizeNote 将笔记内容向量化存储
func (s *NoteVectorService) VectorizeNote(userID, noteID uint, title, content string) error {
	ctx := context.Background()
	collectionName := fmt.Sprintf("user_notes_%d", userID)

	// 先删除旧向量
	s.deleteNoteVectors(ctx, noteID)

	// 按标题切片
	chunks := SplitMarkdownByHeading(content)
	if len(chunks) == 0 {
		chunks = []Chunk{{Content: content, Title: title}}
	}

	for i, chunk := range chunks {
		if len(chunk.Content) < 10 {
			continue // 跳过太短的片段
		}

		// 生成向量
		embedding, err := s.aiService.Embed(chunk.Content)
		if err != nil {
			log.Printf("[NoteVectorService] 笔记 %d 第 %d 块向量化失败: %v", noteID, i, err)
			continue
		}

		// 存储向量
		docID := fmt.Sprintf("note_%d_chunk_%d", noteID, i)
		metadata := map[string]string{
			"note_id":  fmt.Sprintf("%d", noteID),
			"chunk_id": fmt.Sprintf("%d", i),
			"title":    title,
			"type":     "note",
		}

		if err := s.vectorSvc.AddVector(ctx, collectionName, docID, embedding, chunk.Content, metadata); err != nil {
			log.Printf("[NoteVectorService] 笔记 %d 第 %d 块存储失败: %v", noteID, i, err)
		}
	}

	log.Printf("[NoteVectorService] 笔记 %d 向量化完成，共 %d 块", noteID, len(chunks))
	return nil
}

// SearchNotes 在用户笔记中搜索相关内容
func (s *NoteVectorService) SearchNotes(userID uint, query string, topK int) ([]SearchResult, error) {
	ctx := context.Background()
	collectionName := fmt.Sprintf("user_notes_%d", userID)

	// 生成查询向量
	queryVector, err := s.aiService.Embed(query)
	if err != nil {
		return nil, fmt.Errorf("生成查询向量失败: %w", err)
	}

	// 搜索
	scoredResults, err := s.vectorSvc.Search(ctx, collectionName, queryVector, topK)
	if err != nil {
		return nil, fmt.Errorf("搜索笔记失败: %w", err)
	}

	// 转换结果
	var results []SearchResult
	for _, se := range scoredResults {
		results = append(results, ScoredEmbeddingToSearchResult(se))
	}

	return results, nil
}

// DeleteNoteVectors 删除指定笔记的所有向量
func (s *NoteVectorService) DeleteNoteVectors(userID, noteID uint) error {
	ctx := context.Background()
	return s.deleteNoteVectors(ctx, noteID)
}

func (s *NoteVectorService) deleteNoteVectors(ctx context.Context, noteID uint) error {
	noteIDStr := fmt.Sprintf("%d", noteID)
	filter := map[string]string{"note_id": noteIDStr}
	deleted, err := s.vectorSvc.DeleteByFilter(ctx, "", filter)
	if err != nil {
		return fmt.Errorf("删除笔记 %d 的向量失败: %w", noteID, err)
	}
	log.Printf("[NoteVectorService] 删除笔记 %d 的向量，共 %d 条", noteID, deleted)
	return nil
}
