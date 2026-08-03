package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/pkg/logger"

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

// ForgetMemory 删除单条用户记忆（带归属校验）。
// 保留旧方法名兼容内部调用，实际委托给 DeleteMemory 统一走归属校验。
func (s *AvatarMemoryService) ForgetMemory(userID uint, memoryDocID string) error {
	return s.DeleteMemory(userID, memoryDocID)
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
		logger.WithModule("AvatarMemoryService").Info("向量数据库未初始化，返回空记忆列表")
		return []MemoryRecord{}, nil
	}

	ctx := context.Background()

	// 注意：CortexDB 无 ListMemories API，只能通过 SearchMemory 近似枚举。
	// 使用多组查询词并去重，降低单一语义偏向导致的结果缺失。
	queries := []string{" ", "工作 生活 学习", "决定 偏好 约定"}
	seen := make(map[string]bool)
	var records []MemoryRecord

	for _, q := range queries {
		if len(records) >= limit {
			break
		}
		remaining := limit - len(records)
		resp, err := s.db.SearchMemory(ctx, cortexdb.MemorySearchRequest{
			Query:     q,
			UserID:    fmt.Sprintf("%d", userID),
			Scope:     cortexdb.MemoryScopeUser,
			Namespace: "avatar",
			TopK:      remaining,
		})
		if err != nil {
			logger.WithModule("AvatarMemoryService").Error("获取用户记忆失败", "userID", userID, "query", q, "error", err)
			continue
		}
		for _, hit := range resp.Results {
			if seen[hit.Memory.ID] {
				continue
			}
			seen[hit.Memory.ID] = true
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
	}

	logger.WithModule("AvatarMemoryService").Info("获取用户记忆成功", "userID", userID, "count", len(records))
	return records, nil
}

// DeleteMemory 删除单条用户记忆。
// 安全校验：先确认 memoryDocID 属于 userID 对应的用户，防止 IDOR 越权删除他人记忆。
func (s *AvatarMemoryService) DeleteMemory(userID uint, memoryDocID string) error {
	if s.db == nil {
		return nil
	}
	// 先列出该用户的全部记忆，确认 memoryDocID 在其中，防止越权删除
	memories, err := s.GetUserMemories(userID, 10000)
	if err != nil {
		return err
	}
	owned := false
	for _, m := range memories {
		if m.DocID == memoryDocID {
			owned = true
			break
		}
	}
	if !owned {
		return ErrMemoryNotFound
	}
	ctx := context.Background()
	_, err = s.db.DeleteMemory(ctx, cortexdb.MemoryDeleteRequest{MemoryID: memoryDocID})
	return err
}

type MemoryRecord struct {
	DocID    string            `json:"doc_id"`
	Content  string            `json:"content"`
	Metadata map[string]string `json:"metadata"`
}