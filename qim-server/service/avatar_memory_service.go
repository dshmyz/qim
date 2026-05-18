package service

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"qim-server/ai"

	"github.com/liliang-cn/cortexdb/v2/pkg/cortexdb"
)

type AvatarMemoryService struct {
	db        *cortexdb.DB
	aiService *ai.AIService
}

func NewAvatarMemoryService(vectorSvc *VectorService, aiService *ai.AIService) *AvatarMemoryService {
	return &AvatarMemoryService{
		db:        vectorSvc.GetDB(),
		aiService: aiService,
	}
}

func (s *AvatarMemoryService) Remember(userID uint, conversationID uint, content string) error {
	ctx := context.Background()
	memoryID := fmt.Sprintf("memory_%d_%d", userID, time.Now().UnixMilli())

	_, err := s.db.SaveMemory(ctx, cortexdb.MemorySaveRequest{
		MemoryID: memoryID,
		UserID:   fmt.Sprintf("%d", userID),
		Content:  content,
		Scope:    cortexdb.MemoryScopeUser,
		Namespace: "avatar",
		Metadata: map[string]interface{}{
			"conversation_id": fmt.Sprintf("%d", conversationID),
			"remembered_at":   fmt.Sprintf("%d", time.Now().Unix()),
		},
	})
	return err
}

func (s *AvatarMemoryService) Recall(userID uint, query string, topK int) ([]SearchResult, error) {
	ctx := context.Background()

	resp, err := s.db.SearchMemory(ctx, cortexdb.MemorySearchRequest{
		Query:     query,
		UserID:    fmt.Sprintf("%d", userID),
		Scope:     cortexdb.MemoryScopeUser,
		Namespace: "avatar",
		TopK:      topK,
	})
	if err != nil {
		return nil, fmt.Errorf("检索记忆失败: %w", err)
	}

	var results []SearchResult
	for _, hit := range resp.Results {
		metadataStr := make(map[string]string)
		for k, v := range hit.Memory.Metadata {
			if s, ok := v.(string); ok {
				metadataStr[k] = s
			}
		}
		results = append(results, SearchResult{
			Content:  hit.Memory.Content,
			Score:    hit.Score,
			Metadata: metadataStr,
			DocID:    hit.Memory.ID,
		})
	}
	return results, nil
}

func (s *AvatarMemoryService) ShouldRemember(message string) (bool, error) {
	prompt := `判断以下对话内容是否包含值得记忆的长期信息。
值得记忆的信息包括：个人偏好、重要决定、项目关键信息、约定事项。
普通闲聊、简短回复不需要记忆。
只返回 true 或 false。

内容：` + message

	aiMessages := []ai.Message{{Role: "user", Content: prompt}}
	result, err := s.aiService.GetCompletion(ai.TaskTypeAnalysis, aiMessages)
	if err != nil {
		return false, err
	}

	return strings.Contains(strings.ToLower(strings.TrimSpace(result)), "true"), nil
}

func (s *AvatarMemoryService) ForgetMemory(userID uint, memoryDocID string) error {
	ctx := context.Background()
	_, err := s.db.DeleteMemory(ctx, cortexdb.MemoryDeleteRequest{MemoryID: memoryDocID})
	return err
}

func (s *AvatarMemoryService) GetMemoryCount(userID uint) (int64, error) {
	memories, err := s.GetUserMemories(userID, 10000)
	if err != nil {
		return 0, err
	}
	return int64(len(memories)), nil
}

func (s *AvatarMemoryService) GetUserMemories(userID uint, limit int) ([]MemoryRecord, error) {
	if s.db == nil {
		log.Printf("[AvatarMemoryService] 向量数据库未初始化，返回空记忆列表")
		return []MemoryRecord{}, nil
	}

	ctx := context.Background()

	// 使用通用查询词获取所有相关记忆
	resp, err := s.db.SearchMemory(ctx, cortexdb.MemorySearchRequest{
		Query:     "all memories",
		UserID:    fmt.Sprintf("%d", userID),
		Scope:     cortexdb.MemoryScopeUser,
		Namespace: "avatar",
		TopK:      limit,
	})
	if err != nil {
		log.Printf("[AvatarMemoryService] 获取用户记忆失败: userID=%d, error=%v", userID, err)
		return nil, err
	}

	var records []MemoryRecord
	for _, hit := range resp.Results {
		metadataStr := make(map[string]string)
		for k, v := range hit.Memory.Metadata {
			if s, ok := v.(string); ok {
				metadataStr[k] = s
			}
		}
		records = append(records, MemoryRecord{
			DocID:    hit.Memory.ID,
			Content:  hit.Memory.Content,
			Metadata: metadataStr,
		})
	}
	log.Printf("[AvatarMemoryService] 获取用户记忆成功: userID=%d, count=%d", userID, len(records))
	return records, nil
}

func (s *AvatarMemoryService) DeleteMemory(memoryDocID string) error {
	ctx := context.Background()
	_, err := s.db.DeleteMemory(ctx, cortexdb.MemoryDeleteRequest{MemoryID: memoryDocID})
	return err
}

type MemoryRecord struct {
	DocID    string            `json:"doc_id"`
	Content  string            `json:"content"`
	Metadata map[string]string `json:"metadata"`
}