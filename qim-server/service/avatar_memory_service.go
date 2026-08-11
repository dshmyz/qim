package service

import (
	"fmt"
	"strings"
	"time"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/pkg/logger"

	"github.com/dshmyz/gracedb/pkg/gracedb"
	"github.com/dshmyz/gracedb/pkg/types"
)

type AvatarMemoryService struct {
	db        *gracedb.DB
	aiService *ai.AIService
	// conflictCheck 判断新旧两条记忆是否"同一主题但结论矛盾"：矛盾则更新旧的
	// （保留 memoryID），否则新增。nil 时用 LLM 默认实现。可注入以便测试不真调 LLM。
	conflictCheck func(newMemo, oldMemo string) (bool, error)
}

func NewAvatarMemoryService(vectorSvc *VectorService, aiService *ai.AIService) *AvatarMemoryService {
	return &AvatarMemoryService{
		db:        vectorSvc.GetDB(),
		aiService: aiService,
	}
}

// SetConflictCheck 注入语义冲突判定器（默认 LLM 实现，测试可用假判定）。
func (s *AvatarMemoryService) SetConflictCheck(f func(newMemo, oldMemo string) (bool, error)) {
	s.conflictCheck = f
}

// memoryConflicts LLM 判定新旧记忆是否冲突；判定器未注入时用默认 LLM 实现。
func (s *AvatarMemoryService) memoryConflicts(newMemo, oldMemo string) (bool, error) {
	if s.conflictCheck != nil {
		return s.conflictCheck(newMemo, oldMemo)
	}
	return checkMemoryConflictWithAI(s.aiService, newMemo, oldMemo)
}

func (s *AvatarMemoryService) Remember(userID uint, conversationID uint, content string, importance float64) error {
	memoryID := fmt.Sprintf("memory_%d_%d", userID, time.Now().UnixMilli())

	_, err := s.db.SaveMemory(types.MemorySaveRequest{
		MemoryID: memoryID,
		UserID:   fmt.Sprintf("%d", userID),
		Content:  content,
		Scope:    "user",
		Namespace: "avatar",
		Importance: importance01(importance), // 1-5 → [0,1]，重要记忆在召回时更靠前
		Metadata: map[string]interface{}{
			"conversation_id": fmt.Sprintf("%d", conversationID),
			"remembered_at":   fmt.Sprintf("%d", time.Now().Unix()),
			"importance":      fmt.Sprintf("%.1f", importance), // 保留 1-5 档位供展示
		},
	})
	return err
}

// ConsolidateMessage 记忆反射闭环（Recall→Consolidate）：先把该用户既有的相关记忆
// 召回进来，连同当前消息+对话上下文一起 LLM 折叠成带主题/摘要的结构化记忆，再落库。
//
// 相比直接 Remember 原始消息，反射能：
//   - 折叠重复提及的同一事实（合并去重）
//   - 产出 Summary/Themes 结构，能在回落时作为更高层记忆被召回
//
// context 为最近几条对话消息（可选），帮助 LLM 理解"这句话在讨论什么"再判断是否值得记。
// 返回是否真的落库（ShouldRemember=false 时为 false）。
func (s *AvatarMemoryService) ConsolidateMessage(userID, conversationID uint, content string, context []string, existingMemories ...SearchResult) (bool, error) {
	if s.db == nil {
		return false, nil
	}
	var memories []SearchResult
	if len(existingMemories) > 0 {
		memories = existingMemories
	} else {
		var err error
		memories, err = s.Recall(userID, content, 3)
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

	ref, verdict, err := reflectConsolidated(s.aiService, content, memSnippets, nil, context)
	if err != nil {
		return false, err
	}
	if !verdict.ShouldRemember || strings.TrimSpace(ref.Summary) == "" {
		return false, nil
	}

	return s.saveConsolidatedMemory(userID, conversationID, "avatar", ref, memories)
}

// saveConsolidatedMemory 落库反射记忆：与最相似旧记忆（score≥0.7）语义冲突时
// 更新旧记忆内容（保留 memoryID），否则新增一条。返回是否真的落库。
// 提取为独立方法便于测试（不依赖整条 LLM 反射流程）。
func (s *AvatarMemoryService) saveConsolidatedMemory(userID, conversationID uint, namespace string, ref MemoryReflection, memories []SearchResult) (bool, error) {
	uid := fmt.Sprintf("%d", userID)
	cid := fmt.Sprintf("%d", conversationID)

	// 语义冲突检测：新记忆与最相似的旧记忆冲突时，更新旧的而非新增。
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
			logger.WithModule("AvatarMemory").Info("记忆冲突，更新旧记忆",
				"userID", userID, "memoryID", memories[0].DocID, "new", ref.Summary)
			return true, nil
		}
	}

	memoryID := fmt.Sprintf("memory_%d_%d", userID, time.Now().UnixMilli())
	_, err := s.db.SaveMemory(types.MemorySaveRequest{
		MemoryID:  memoryID,
		UserID:    uid,
		Content:   ref.Summary,
		Scope:     "user",
		Namespace: namespace,
		Importance: importance01(ref.Importance),
		Metadata: map[string]interface{}{
			"conversation_id":          cid,
			"remembered_at":            fmt.Sprintf("%d", time.Now().Unix()),
			"importance":               fmt.Sprintf("%.1f", ref.Importance),
			"knowledge_memory_summary": "true", // 标记为反射摘要记忆
			"knowledge_memory_themes":  ref.Themes,
			"knowledge_memory_entities": ref.Entities,
		},
	})
	if err != nil {
		return false, err
	}
	logger.WithModule("AvatarMemoryService").Info("记忆反射落库",
		"userID", userID, "content", ref.Summary)
	return true, nil
}

func (s *AvatarMemoryService) Recall(userID uint, query string, topK int) ([]SearchResult, error) {
	resp, err := s.db.SearchMemory(types.MemorySearchRequest{
		Query:     query,
		UserID:    fmt.Sprintf("%d", userID),
		Scope:     "user",
		Namespace: "avatar",
		TopK:      topK,
		// 提升重要度与新颖度的排序权重：反射落库的重要记忆（Importance 1-5 → [0,1]）与
		// 较新的记忆在召回时更靠前，避免被默认权重（importance 0.10 / recency 0.05）稀释。
		SemanticWeight:   0.55,
		LexicalWeight:   0.15,
		ImportanceWeight: 0.20,
		RecencyWeight:   0.10,
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
	v, err := s.ShouldRememberWithImportance(message)
	if err != nil {
		return false, err
	}
	return v.ShouldRemember, nil
}

// ShouldRememberWithImportance 判断内容是否值得记，并给出重要度档位（1-5）。
func (s *AvatarMemoryService) ShouldRememberWithImportance(message string) (RememberVerdict, error) {
	const prompt = `判断以下对话内容是否包含值得记忆的长期信息。
值得记忆的信息包括：个人偏好、重要决定、项目关键信息、约定事项。
普通闲聊、简短回复不需要记忆。`
	return evaluateRemember(s.aiService, prompt, message)
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

	// 懒触发本用户的弱记忆归档：顺带把"既弱又长期闲置"的分身记忆 soft-hide，
	// 避免记忆库随使用无限膨胀拖垮面板列表、知识图谱与召回。尽力而为、受冷却节流、
	// 失败不阻断列表读取。前端与知识图谱均零改动（归档记忆自动从枚举/召回消失）。
	if _, err := lazyArchiveWeakMemories(s.db, fmt.Sprintf("%d", userID), "avatar"); err != nil {
		logger.WithModule("AvatarMemoryService").Warn("懒归档不阻断列表",
			"userID", userID, "error", err)
	}

	// gracedb 空 Query（且无 QueryVector）时走内存桶列表路径，精确枚举该用户 avatar 桶内的
	// 全部记忆（排除已过期/已归档），无需像旧版那样用多组查询词做近似召回再删重。
	resp, err := s.db.SearchMemory(types.MemorySearchRequest{
		UserID:    fmt.Sprintf("%d", userID),
		Scope:     "user",
		Namespace: "avatar",
		TopK:      limit,
	})
	if err != nil {
		logger.WithModule("AvatarMemoryService").Error("获取用户记忆失败", "userID", userID, "error", err)
		return nil, err
	}

	records := make([]MemoryRecord, 0, len(resp.Results))
	for _, hit := range resp.Results {
		metadataStr := make(map[string]string)
		var entities, themes []string
		for k, v := range hit.Memory.Metadata {
			if s, ok := v.(string); ok {
				metadataStr[k] = s
				continue
			}
			// gracedb 把 map[string]any 的值 JSON 序列化后读回为 []interface{}
			switch k {
			case "knowledge_memory_entities":
				entities = toStringSlice(v)
			case "knowledge_memory_themes":
				themes = toStringSlice(v)
			}
		}
		records = append(records, MemoryRecord{
			DocID:    hit.Memory.ID,
			Content:  hit.Memory.Content,
			Metadata: metadataStr,
			Entities: entities,
			Themes:   themes,
		})
	}

	logger.WithModule("AvatarMemoryService").Info("获取用户记忆成功", "userID", userID, "count", len(records))
	return records, nil
}

// toStringSlice 把 gracedb 读回的 metadata 数组值（JSON 反序列化后是 []interface{}，
// 实际元素均为 string）转成 []string，忽略非字符串元素。
func toStringSlice(v interface{}) []string {
	arr, ok := v.([]interface{})
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
			out = append(out, s)
		}
	}
	return out
}

// MemoryGraph 是分身知识图谱「记忆」来源的聚合结果。
type MemoryGraph struct {
	Nodes    []MemoryGraphNode    `json:"nodes"`
	Edges    []MemoryGraphEdge    `json:"edges"`
	Memories []MemoryGraphMemory  `json:"memories"`
}

type MemoryGraphNode struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Type  string `json:"type"` // entity | theme
	Count int    `json:"count"`
}

type MemoryGraphEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Weight int    `json:"weight"`
}

type MemoryGraphMemory struct {
	ID      string   `json:"id"`
	Content string   `json:"content"`
	// Terms 该条记忆的主题/实体集合，供前端点节点时按名字回查"包含此名词的记忆"
	Terms []string `json:"terms"`
}

// BuildMemoryGraph 从该用户的 avatar 记忆里聚合出实体/主题共现图谱。
// 节点 = 记忆反射落库的 entities/themes（每条记忆的），边 = 同一条记忆里共同出现的
// 实体/主题对（weight = 共同出现的记忆条数）。点节点的关联记忆由调用方用返回的
// memories（含每条记忆的 entities/themes 可回查）拼装。
func (s *AvatarMemoryService) BuildMemoryGraph(userID uint, limit int) (*MemoryGraph, error) {
	records, err := s.GetUserMemories(userID, limit)
	if err != nil {
		return nil, err
	}

	graph := buildMemoryGraphFromRecords(records)

	logger.WithModule("AvatarMemoryService").Info("构建记忆图谱",
		"userID", userID, "memories", len(records), "nodes", len(graph.Nodes), "edges", len(graph.Edges))
	return graph, nil
}

// buildMemoryGraphFromRecords 把「该用户记忆记录」聚合成语义图谱（纯函数，无 IO，便于单测）。
func buildMemoryGraphFromRecords(records []MemoryRecord) *MemoryGraph {
	graph := &MemoryGraph{
		Nodes:    make([]MemoryGraphNode, 0),
		Edges:    make([]MemoryGraphEdge, 0),
		Memories: make([]MemoryGraphMemory, 0, len(records)),
	}

	// name -> node 索引
	nodeIdx := make(map[string]int)

	// 每条记忆的名词集合（含主题与实体），用于共现与回查
	memoryTerms := make([][]string, 0, len(records))

	for _, r := range records {
		graph.Memories = append(graph.Memories, MemoryGraphMemory{ID: r.DocID, Content: r.Content})
		terms := make([]string, 0)
		seenTerm := make(map[string]bool)
		addNode := func(name, typ string) {
			if name == "" || seenTerm[name] {
				return
			}
			seenTerm[name] = true
			terms = append(terms, name)
			if idx, ok := nodeIdx[name]; ok {
				graph.Nodes[idx].Count++
			} else {
				nodeIdx[name] = len(graph.Nodes)
				graph.Nodes = append(graph.Nodes, MemoryGraphNode{
					ID:    name,
					Name:  name,
					Type:  typ,
					Count: 1,
				})
			}
		}
		for _, e := range r.Entities {
			addNode(strings.TrimSpace(e), "entity")
		}
		for _, t := range r.Themes {
			addNode(strings.TrimSpace(t), "theme")
		}
		memoryTerms = append(memoryTerms, terms)
		graph.Memories[len(graph.Memories)-1].Terms = terms
	}

	// 共现边：同一条记忆内所有名词两两配对
	edgeKey := func(a, b string) string {
		if a < b {
			return a + "\x00" + b
		}
		return b + "\x00" + a
	}
	edgeW := make(map[string]int)
	for _, terms := range memoryTerms {
		for i := 0; i < len(terms); i++ {
			for j := i + 1; j < len(terms); j++ {
				if terms[i] == terms[j] {
					continue
				}
				k := edgeKey(terms[i], terms[j])
				edgeW[k]++
			}
		}
	}
	for k, w := range edgeW {
		parts := strings.Split(k, "\x00")
		graph.Edges = append(graph.Edges, MemoryGraphEdge{Source: parts[0], Target: parts[1], Weight: w})
	}

	return graph
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
	return s.db.DeleteMemory(memoryDocID)
}

// UpdateMemory 纠正一条记忆：把 memoryID 指向的记忆内容替换为 newContent。
// 先校验该记忆确属当前用户（防越权纠正），再做深度更新（内容 + 向量 + 更新向量 embedding，
// 使纠正后能正确召回）。返回 ErrMemoryNotFound 表示记录不存在或不属于该用户。
func (s *AvatarMemoryService) UpdateMemory(userID uint, memoryDocID, newContent string) error {
	if s.db == nil {
		return nil
	}
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
	content := newContent
	var uerr error
	_, uerr = s.db.UpdateMemory(types.MemoryUpdateRequest{
		MemoryID: memoryDocID,
		Content:  &content,
	})
	return uerr
}

type MemoryRecord struct {
	DocID    string            `json:"doc_id"`
	Content  string            `json:"content"`
	Metadata map[string]string `json:"metadata"`
	// Entities/Themes 来自记忆反射（reflectConsolidated）时落库的
	// knowledge_memory_entities / knowledge_memory_themes（gracedb 存为 JSON 数组，
	// 读回时是 []interface{}，须在枚举处单独提取为 []string），供知识图谱聚合使用。
	Entities []string `json:"entities,omitempty"`
	Themes   []string `json:"themes,omitempty"`
}