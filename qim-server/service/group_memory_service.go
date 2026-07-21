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

// GroupMemoryService 群聊助手的群级长期记忆。
//
// 与分身记忆（AvatarMemoryService，Namespace="avatar"，按 userID 键）彻底隔离：
// 本服务用 Namespace="group_assistant" + UserID=<groupID> 键，存储本群值得记的消息，
// 召回时只取本群记忆，杜绝"群助手蹭发送者分身记忆"导致的跨上下文（私聊->群）泄露。
type GroupMemoryService struct {
	db        *cortexdb.DB
	aiService *ai.AIService
}

func NewGroupMemoryService(vectorSvc *VectorService, aiService *ai.AIService) *GroupMemoryService {
	if vectorSvc == nil {
		return &GroupMemoryService{db: nil, aiService: aiService}
	}
	return &GroupMemoryService{db: vectorSvc.GetDB(), aiService: aiService}
}

const groupMemoryNamespace = "group_assistant"

// Remember 把一条值得记的群消息写入群级记忆库。
func (s *GroupMemoryService) Remember(groupID uint, conversationID uint, content string) error {
	if s.db == nil {
		return nil
	}
	ctx := context.Background()
	memoryID := fmt.Sprintf("groupmem_%d_%d", groupID, time.Now().UnixMilli())

	_, err := s.db.SaveMemory(ctx, cortexdb.MemorySaveRequest{
		MemoryID:  memoryID,
		UserID:    fmt.Sprintf("%d", groupID),
		Content:   content,
		Scope:     cortexdb.MemoryScopeUser,
		Namespace: groupMemoryNamespace,
		Metadata: map[string]interface{}{
			"conversation_id": fmt.Sprintf("%d", conversationID),
			"remembered_at":   fmt.Sprintf("%d", time.Now().Unix()),
		},
	})
	return err
}

// Recall 按群检索与当前消息最相关的若干条群记忆。
func (s *GroupMemoryService) Recall(groupID uint, query string, topK int) ([]SearchResult, error) {
	if s.db == nil {
		return nil, nil
	}
	ctx := context.Background()

	resp, err := s.db.SearchMemory(ctx, cortexdb.MemorySearchRequest{
		Query:     query,
		UserID:    fmt.Sprintf("%d", groupID),
		Scope:     cortexdb.MemoryScopeUser,
		Namespace: groupMemoryNamespace,
		TopK:      topK,
	})
	if err != nil {
		return nil, fmt.Errorf("检索群记忆失败: %w", err)
	}

	var results []SearchResult
	for _, hit := range resp.Results {
		metadataStr := make(map[string]string)
		for k, v := range hit.Memory.Metadata {
			if str, ok := v.(string); ok {
				metadataStr[k] = str
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

// ShouldRemember 用 LLM 判断群消息是否值得记入群级记忆。
// 值得记：群决定、约定事项、项目关键信息、群偏好；普通闲聊/简短回复不记。
func (s *GroupMemoryService) ShouldRemember(message string) (bool, error) {
	if s.aiService == nil {
		return false, nil
	}
	prompt := `判断以下群聊内容是否包含值得群助手长期记忆的信息。
值得记忆的信息包括：群内决定、约定事项、项目关键信息、群偏好与共识。
普通闲聊、简短回复、打招呼不需要记忆。
只返回 true 或 false。

内容：` + message

	aiMessages := []ai.Message{{Role: "user", Content: prompt}}
	result, err := s.aiService.GetCompletion(ai.TaskTypeAnalysis, aiMessages)
	if err != nil {
		return false, err
	}
	return strings.Contains(strings.ToLower(strings.TrimSpace(result)), "true"), nil
}

// GetGroupMemories 列出本群记忆条目（按相关性近似排序）。
func (s *GroupMemoryService) GetGroupMemories(groupID uint, limit int) ([]MemoryRecord, error) {
	if s.db == nil {
		logger.WithModule("GroupMemoryService").Info("向量数据库未初始化，返回空群记忆列表")
		return []MemoryRecord{}, nil
	}
	ctx := context.Background()

	resp, err := s.db.SearchMemory(ctx, cortexdb.MemorySearchRequest{
		Query:     "all memories",
		UserID:    fmt.Sprintf("%d", groupID),
		Scope:     cortexdb.MemoryScopeUser,
		Namespace: groupMemoryNamespace,
		TopK:      limit,
	})
	if err != nil {
		logger.WithModule("GroupMemoryService").Error("获取群记忆失败", "groupID", groupID, "error", err)
		return nil, err
	}

	var records []MemoryRecord
	for _, hit := range resp.Results {
		metadataStr := make(map[string]string)
		for k, v := range hit.Memory.Metadata {
			if str, ok := v.(string); ok {
				metadataStr[k] = str
			}
		}
		records = append(records, MemoryRecord{
			DocID:    hit.Memory.ID,
			Content:  hit.Memory.Content,
			Metadata: metadataStr,
		})
	}
	return records, nil
}

// GetMemoryCount 返回本群记忆条目数。
func (s *GroupMemoryService) GetMemoryCount(groupID uint) (int64, error) {
	memories, err := s.GetGroupMemories(groupID, 10000)
	if err != nil {
		return 0, err
	}
	return int64(len(memories)), nil
}

// DeleteMemory 删除单条群记忆。
func (s *GroupMemoryService) DeleteMemory(memoryDocID string) error {
	if s.db == nil {
		return nil
	}
	ctx := context.Background()
	_, err := s.db.DeleteMemory(ctx, cortexdb.MemoryDeleteRequest{MemoryID: memoryDocID})
	return err
}

// ForgetAll 清空本群全部群记忆（列出后逐条删除）。
func (s *GroupMemoryService) ForgetAll(groupID uint) (int, error) {
	memories, err := s.GetGroupMemories(groupID, 10000)
	if err != nil {
		return 0, err
	}
	deleted := 0
	for _, m := range memories {
		if err := s.DeleteMemory(m.DocID); err == nil {
			deleted++
		}
	}
	return deleted, nil
}
