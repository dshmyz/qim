package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/pkg/logger"

	"github.com/dshmyz/gracedb/pkg/gracedb"
	"github.com/dshmyz/gracedb/pkg/types"
)

// ErrMemoryNotFound 表示记忆不存在或不属于当前作用域（防止 IDOR 越权删除）。
var ErrMemoryNotFound = errors.New("memory not found")

// GroupMemoryService 群聊助手的群级长期记忆。
//
// 与分身记忆（AvatarMemoryService，Namespace="avatar"，按 userID 键）彻底隔离：
// 本服务用 Namespace="group_assistant" + UserID=<groupID> 键，存储本群值得记的消息，
// 召回时只取本群记忆，杜绝"群助手蹭发送者分身记忆"导致的跨上下文（私聊->群）泄露。
type GroupMemoryService struct {
	db        *gracedb.DB
	aiService *ai.AIService
	// conflictCheck 判断新旧两条群记忆是否"同一主题但结论矛盾"：矛盾则更新旧的，
	// 否则新增。nil 时用 LLM 默认实现。可注入以便测试不真调 LLM。
	conflictCheck func(newMemo, oldMemo string) (bool, error)
}

func NewGroupMemoryService(vectorSvc *VectorService, aiService *ai.AIService) *GroupMemoryService {
	if vectorSvc == nil {
		return &GroupMemoryService{db: nil, aiService: aiService}
	}
	return &GroupMemoryService{db: vectorSvc.GetDB(), aiService: aiService}
}

// SetConflictCheck 注入语义冲突判定器（默认 LLM 实现，测试可用假判定）。
func (s *GroupMemoryService) SetConflictCheck(f func(newMemo, oldMemo string) (bool, error)) {
	s.conflictCheck = f
}

// memoryConflicts LLM 判定新旧记忆是否冲突；判定器未注入时用默认 LLM 实现。
func (s *GroupMemoryService) memoryConflicts(newMemo, oldMemo string) (bool, error) {
	if s.conflictCheck != nil {
		return s.conflictCheck(newMemo, oldMemo)
	}
	return checkMemoryConflictWithAI(s.aiService, newMemo, oldMemo)
}

const groupMemoryNamespace = "group_assistant"

// Remember 把一条值得记的群消息写入群级记忆库。
func (s *GroupMemoryService) Remember(groupID uint, conversationID uint, content string, importance float64) error {
	if s.db == nil {
		return nil
	}
	memoryID := fmt.Sprintf("groupmem_%d_%d", groupID, time.Now().UnixMilli())

	_, err := s.db.SaveMemory(types.MemorySaveRequest{
		MemoryID:   memoryID,
		UserID:     fmt.Sprintf("%d", groupID),
		Content:    content,
		Scope:      "user",
		Namespace:  groupMemoryNamespace,
		Importance: importance01(importance), // 1-5 → [0,1]
		Metadata: map[string]interface{}{
			"conversation_id": fmt.Sprintf("%d", conversationID),
			"remembered_at":   fmt.Sprintf("%d", time.Now().Unix()),
			"importance":      fmt.Sprintf("%.1f", importance),
		},
	})
	return err
}

// ConsolidateGroupMessage 群记忆反射闭环（Recall→Consolidate）：召回本群既有的相关记忆
// + 传入的群知识片段，连同当前群消息+对话上下文一起 LLM 折叠成带主题/摘要的结构化记忆再落库。
//
// 相比直接 Remember 原始消息：能折叠重复提及的同一事实、产出结构化摘要，作为更高层
// 群记忆被后续召回。knowledge 由调用方用自建 searchHybrid 语义召回后传入（本服务不持
// 有 groupDocSvc，避免循环依赖）。context 为最近几条对话消息（可选）。
// 返回是否真的落库。
func (s *GroupMemoryService) ConsolidateGroupMessage(groupID, conversationID uint, content string, knowledge []string, context []string, existingMemories ...SearchResult) (bool, error) {
	if s.db == nil {
		return false, nil
	}
	var memories []SearchResult
	if len(existingMemories) > 0 {
		memories = existingMemories
	} else {
		var err error
		memories, err = s.Recall(groupID, content, 3)
		if err != nil {
			memories = nil
		}
	}
	memSnippets := make([]string, 0, len(memories))
	for _, m := range memories {
		if m.Content != "" {
			memSnippets = append(memSnippets, m.Content)
		}
	}

	ref, verdict, err := reflectConsolidated(s.aiService, content, memSnippets, knowledge, context)
	if err != nil {
		return false, err
	}
	if !verdict.ShouldRemember || strings.TrimSpace(ref.Summary) == "" {
		return false, nil
	}

	return s.saveConsolidatedGroupMemory(groupID, conversationID, ref, memories)
}

// saveConsolidatedGroupMemory 落库群记忆：与最相似旧记忆（score≥0.7）语义冲突时
// 更新旧记忆内容（保留 memoryID），否则新增一条。返回是否真的落库。
// 提取为独立方法便于测试。
func (s *GroupMemoryService) saveConsolidatedGroupMemory(groupID, conversationID uint, ref MemoryReflection, memories []SearchResult) (bool, error) {
	// 语义冲突检测：新记忆与最相似的旧记忆冲突时更新旧的，避免"用MySQL→改用
	// PostgreSQL"这类矛盾群记忆并存、群助手后续混淆。
	if len(memories) > 0 && memories[0].Score >= 0.7 {
		conflict, cerr := s.memoryConflicts(ref.Summary, memories[0].Content)
		if cerr == nil && conflict {
			content := ref.Summary
			_, uerr := s.db.UpdateMemory(types.MemoryUpdateRequest{
				MemoryID: memories[0].DocID,
				Content:  &content,
				Importance: func() *float64 { v := importance01(ref.Importance); return &v }(),
			})
			if uerr != nil {
				return false, uerr
			}
			logger.WithModule("GroupMemoryService").Info("群记忆冲突，更新旧记忆",
				"groupID", groupID, "memoryID", memories[0].DocID, "new", ref.Summary)
			return true, nil
		}
	}

	memoryID := fmt.Sprintf("groupmem_%d_%d", groupID, time.Now().UnixMilli())
	_, err := s.db.SaveMemory(types.MemorySaveRequest{
		MemoryID:  memoryID,
		UserID:    fmt.Sprintf("%d", groupID),
		Content:   ref.Summary,
		Scope:     "user",
		Namespace: groupMemoryNamespace,
		Importance: importance01(ref.Importance),
		Metadata: map[string]interface{}{
			"conversation_id":           fmt.Sprintf("%d", conversationID),
			"remembered_at":             fmt.Sprintf("%d", time.Now().Unix()),
			"importance":                fmt.Sprintf("%.1f", ref.Importance),
			"knowledge_memory_summary":  "true",
			"knowledge_memory_themes":   ref.Themes,
			"knowledge_memory_entities": ref.Entities,
		},
	})
	if err != nil {
		return false, err
	}
	logger.WithModule("GroupMemoryService").Info("群记忆反射落库",
		"groupID", groupID, "content", ref.Summary)
	return true, nil
}

// Recall 按群检索与当前消息最相关的若干条群记忆。
func (s *GroupMemoryService) Recall(groupID uint, query string, topK int) ([]SearchResult, error) {
	if s.db == nil {
		return nil, nil
	}

	resp, err := s.db.SearchMemory(types.MemorySearchRequest{
		Query:     query,
		UserID:    fmt.Sprintf("%d", groupID),
		Scope:     "user",
		Namespace: groupMemoryNamespace,
		TopK:      topK,
		// 提升重要度与新颖度权重：与分身 Recall 对齐，重要且较新的群记忆更靠前。
		SemanticWeight:   0.55,
		LexicalWeight:   0.15,
		ImportanceWeight: 0.20,
		RecencyWeight:   0.10,
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
	v, err := s.ShouldRememberWithImportance(message)
	if err != nil {
		return false, err
	}
	return v.ShouldRemember, nil
}

// ShouldRememberWithImportance 判断群消息是否值得记，并给出重要度档位（1-5）。
func (s *GroupMemoryService) ShouldRememberWithImportance(message string) (RememberVerdict, error) {
	const prompt = `判断以下群聊内容是否包含值得群助手长期记忆的信息。
值得记忆的信息包括：群内决定、约定事项、项目关键信息、群偏好与共识。
普通闲聊、简短回复、打招呼不需要记忆。`
	return evaluateRemember(s.aiService, prompt, message)
}

// GetGroupMemories 列出本群记忆条目（按相关性近似排序）。
func (s *GroupMemoryService) GetGroupMemories(groupID uint, limit int) ([]MemoryRecord, error) {
	if s.db == nil {
		logger.WithModule("GroupMemoryService").Info("向量数据库未初始化，返回空群记忆列表")
		return []MemoryRecord{}, nil
	}

	// 懒触发本群的弱群记忆归档：顺带 soft-hide "既弱又长期闲置"的群记忆，
	// 避免群记忆库无限膨胀拖垮面板/图谱/召回。尽力而为、受冷却节流、失败不阻断。
	if _, err := lazyArchiveWeakMemories(s.db, fmt.Sprintf("%d", groupID), groupMemoryNamespace); err != nil {
		logger.WithModule("GroupMemoryService").Warn("懒归档不阻断列表",
			"groupID", groupID, "error", err)
	}

	// 空 Query（且无 QueryVector）走内存桶列表路径，精确枚举本群记忆（排除过期/归档），
	// 按重要度+新颖度排序；比旧版用 "all memories" 做一次无意义语义召回更准确也更省。
	resp, err := s.db.SearchMemory(types.MemorySearchRequest{
		UserID:    fmt.Sprintf("%d", groupID),
		Scope:     "user",
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
// 安全校验：先确认 memoryDocID 属于 groupID 对应的群，防止跨群 IDOR 删除。
func (s *GroupMemoryService) DeleteMemory(groupID uint, memoryDocID string) error {
	if s.db == nil {
		return nil
	}
	// 先列出该群的全部记忆，确认 memoryDocID 在其中，防止跨群删除
	memories, err := s.GetGroupMemories(groupID, 10000)
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
	return s.db.DeleteMemory(memoryDocID)
}

// ForgetAll 清空本群全部群记忆（列出后逐条删除）。
func (s *GroupMemoryService) ForgetAll(groupID uint) (int, error) {
	memories, err := s.GetGroupMemories(groupID, 10000)
	if err != nil {
		return 0, err
	}
	deleted := 0
	for _, m := range memories {
		if err := s.DeleteMemory(groupID, m.DocID); err == nil {
			deleted++
		}
	}
	return deleted, nil
}
