package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"qim-server/ai"

	"gorm.io/gorm"
)

// AvatarMemoryService 分身长期记忆存储和检索服务
type AvatarMemoryService struct {
	vectorSvc *VectorService
	aiService *ai.AIService
	db        *gorm.DB
}

// NewAvatarMemoryService 创建分身记忆服务实例
func NewAvatarMemoryService(vectorSvc *VectorService, aiService *ai.AIService, db *gorm.DB) *AvatarMemoryService {
	return &AvatarMemoryService{
		vectorSvc: vectorSvc,
		aiService: aiService,
		db:        db,
	}
}

// Remember 记忆重要信息
func (s *AvatarMemoryService) Remember(userID uint, conversationID uint, content string) error {
	ctx := context.Background()
	collectionName := fmt.Sprintf("avatar_memory_%d", userID)

	// 生成向量
	embedding, err := s.aiService.Embed(content)
	if err != nil {
		return fmt.Errorf("生成记忆向量失败: %w", err)
	}

	// 存储记忆
	docID := fmt.Sprintf("memory_%d_%d", userID, time.Now().UnixMilli())
	metadata := map[string]string{
		"conversation_id": fmt.Sprintf("%d", conversationID),
		"remembered_at":   fmt.Sprintf("%d", time.Now().Unix()),
		"type":            "memory",
	}

	return s.vectorSvc.AddVector(ctx, collectionName, docID, embedding, content, metadata)
}

// Recall 检索相关记忆
func (s *AvatarMemoryService) Recall(userID uint, query string, topK int) ([]SearchResult, error) {
	ctx := context.Background()
	collectionName := fmt.Sprintf("avatar_memory_%d", userID)

	// 生成查询向量
	queryVector, err := s.aiService.Embed(query)
	if err != nil {
		return nil, fmt.Errorf("生成查询向量失败: %w", err)
	}

	// 搜索记忆
	scoredResults, err := s.vectorSvc.Search(ctx, collectionName, queryVector, topK)
	if err != nil {
		return nil, fmt.Errorf("检索记忆失败: %w", err)
	}

	var results []SearchResult
	for _, se := range scoredResults {
		results = append(results, ScoredEmbeddingToSearchResult(se))
	}

	return results, nil
}

// ShouldRemember 判断信息是否值得记忆（用 LLM 判断）
func (s *AvatarMemoryService) ShouldRemember(message string) (bool, error) {
	prompt := `判断以下对话内容是否包含值得记忆的长期信息。
值得记忆的信息包括：个人偏好、重要决定、项目关键信息、约定事项。
普通闲聊、简短回复不需要记忆。
只返回 true 或 false。

内容：` + message

	aiMessages := []ai.Message{{Role: "user", Content: prompt}}
	result, err := s.aiService.GetCompletion(aiMessages)
	if err != nil {
		return false, err
	}

	return strings.Contains(strings.ToLower(strings.TrimSpace(result)), "true"), nil
}

// ForgetMemory 忘记指定记忆
func (s *AvatarMemoryService) ForgetMemory(userID uint, memoryDocID string) error {
	ctx := context.Background()
	return s.vectorSvc.store.DeleteByDocID(ctx, memoryDocID)
}

// GetMemoryCount 获取用户的记忆条数
func (s *AvatarMemoryService) GetMemoryCount(userID uint) (int64, error) {
	// 简化实现：通过搜索所有记忆来统计
	ctx := context.Background()
	collectionName := fmt.Sprintf("avatar_memory_%d", userID)

	// 使用一个空向量搜索所有（这里简化处理）
	// 实际应该维护一个计数表
	stats, err := s.vectorSvc.store.Stats(ctx)
	if err != nil {
		return 0, err
	}
	_ = collectionName // 保留用于后续优化
	return stats.Count, nil
}

// GetUserMemories 获取用户的所有记忆列表（用于管理界面）
func (s *AvatarMemoryService) GetUserMemories(userID uint, limit int) ([]MemoryRecord, error) {
	ctx := context.Background()
	collectionName := fmt.Sprintf("avatar_memory_%d", userID)

	memories, err := s.vectorSvc.GetByCollection(ctx, collectionName, limit)
	if err != nil {
		return nil, err
	}

	var records []MemoryRecord
	for _, m := range memories {
		records = append(records, MemoryRecord{
			DocID:          m.DocID,
			Content:        m.Content,
			Metadata:       m.Metadata,
		})
	}
	return records, nil
}

// DeleteMemory 删除指定记忆（通过 docID）
func (s *AvatarMemoryService) DeleteMemory(memoryDocID string) error {
	ctx := context.Background()
	return s.vectorSvc.store.DeleteByDocID(ctx, memoryDocID)
}

// MemoryRecord 记忆记录结构
type MemoryRecord struct {
	DocID    string            `json:"doc_id"`
	Content  string            `json:"content"`
	Metadata map[string]string `json:"metadata"`
}
