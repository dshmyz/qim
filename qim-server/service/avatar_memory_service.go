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
	// mergeCheck 判断新记忆相对最相关旧记忆的合并关系（冲突→更新、复述→合并、不同→新增）。
	// nil 时用共享 LLM 默认实现（等同 R1 统一判定）。可注入以便测试不真调 LLM。
	mergeCheck func(newMemo, oldMemo string) (MemoryMergeKind, error)
	// thresholdSvc 阈值读取服务；nil 时用默认 0.3（与 config 默认一致）。
	thresholdSvc *AiThresholdService
}

func NewAvatarMemoryService(vectorSvc *VectorService, aiService *ai.AIService) *AvatarMemoryService {
	return &AvatarMemoryService{
		db:        vectorSvc.GetDB(),
		aiService: aiService,
	}
}

// SetMergeCheck 注入合并关系判定器（默认 LLM 实现，测试可用假判定）。
func (s *AvatarMemoryService) SetMergeCheck(f func(newMemo, oldMemo string) (MemoryMergeKind, error)) {
	s.mergeCheck = f
}

// SetThresholdService 注入阈值读取服务；nil 时冲突检测用默认 0.3。
func (s *AvatarMemoryService) SetThresholdService(t *AiThresholdService) {
	s.thresholdSvc = t
}

// conflictThreshold 返回合并/冲突检测分数门槛：未注入阈值服务时回退默认 0.3。
func (s *AvatarMemoryService) conflictThreshold() float64 {
	if s.thresholdSvc != nil {
		return s.thresholdSvc.GetFloat("ai.conflict_detection_threshold", 0.3)
	}
	return 0.3
}

// memoryMerge 判定新记忆相对最相关旧记忆的合并关系；判定器未注入时用共享 LLM 默认实现。
func (s *AvatarMemoryService) memoryMerge(newMemo, oldMemo string) (MemoryMergeKind, error) {
	if s.mergeCheck != nil {
		return s.mergeCheck(newMemo, oldMemo)
	}
	return inferMemoryMergeKind(s.aiService, newMemo, oldMemo)
}

func (s *AvatarMemoryService) Remember(userID uint, conversationID uint, content string, importance float64) error {
	avatarWriteMutex.Lock(fmt.Sprintf("%d", userID))
	defer avatarWriteMutex.Unlock(fmt.Sprintf("%d", userID))
	memoryID := fmt.Sprintf("memory_%d_%d", userID, time.Now().UnixMilli())

	_, err := s.db.SaveMemory(types.MemorySaveRequest{
		MemoryID:   memoryID,
		UserID:     fmt.Sprintf("%d", userID),
		Content:    content,
		Scope:      "user",
		Namespace:  "avatar",
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

// saveConsolidatedMemory 落库反射记忆：与最相似旧记忆（score≥冲突检测门槛）语义冲突时
// 更新旧记忆内容（保留 memoryID），否则新增一条。返回是否真的落库。
// 提取为独立方法便于测试（不依赖整条 LLM 反射流程）。
func (s *AvatarMemoryService) saveConsolidatedMemory(userID, conversationID uint, namespace string, ref MemoryReflection, memories []SearchResult) (bool, error) {
	// 并发串行化：同一用户的写路径互斥，防止多条近义/冲突记忆并发对同一目标竞态。
	avatarWriteMutex.Lock(fmt.Sprintf("%d", userID))
	defer avatarWriteMutex.Unlock(fmt.Sprintf("%d", userID))
	uid := fmt.Sprintf("%d", userID)
	cid := fmt.Sprintf("%d", conversationID)

	// 合并关系判定（R1）：命中足够相似的旧记忆时，按其关系决定更新/合并/新增。
	if len(memories) > 0 && memories[0].Score >= s.conflictThreshold() {
		kind, kerr := s.memoryMerge(ref.Summary, memories[0].Content)
		switch {
		case kerr != nil:
			// 判定失败：退化为新增（不更新、不误并，安全方向），由下方插入路径处理
		case kind == MemoryMergeConflict:
			// 冲突合并只修正内容与重要度；scope（召回边界）与 conversation_id 沿用旧记录，
			// 避免一次在别的对话提及就让"专属约定"漂移为新对话的 global 记忆。
			content := ref.Summary
			_, uerr := s.db.UpdateMemory(types.MemoryUpdateRequest{
				MemoryID:   memories[0].DocID,
				Content:    &content,
				Importance: func() *float64 { v := importance01(mergedImportance(memories[0].Metadata, ref.Importance)); return &v }(),
			})
			if uerr != nil {
				return false, uerr
			}
			logger.WithModule("AvatarMemory").Info("记忆冲突，更新旧记忆",
				"userID", userID, "memoryID", memories[0].DocID, "new", ref.Summary)
			return true, nil
		case kind == MemoryMergeDuplicate:
			// 近义复述/确认：不新增，仅把旧记忆重要度刷新为较高档位，防止重复记忆占位。
			_, uerr := s.db.UpdateMemory(types.MemoryUpdateRequest{
				MemoryID: memories[0].DocID,
				Importance: func() *float64 {
					v := importance01(mergedImportance(memories[0].Metadata, ref.Importance))
					return &v
				}(),
			})
			if uerr != nil {
				return false, uerr
			}
			logger.WithModule("AvatarMemory").Info("记忆近义复述，合并刷新重要度",
				"userID", userID, "memoryID", memories[0].DocID)
			return true, nil
		}
	}

	memoryID := fmt.Sprintf("memory_%d_%d", userID, time.Now().UnixMilli())
	_, err := s.db.SaveMemory(types.MemorySaveRequest{
		MemoryID:   memoryID,
		UserID:     uid,
		Content:    ref.Summary,
		Scope:      "user",
		Namespace:  namespace,
		Importance: importance01(ref.Importance),
		Metadata: map[string]interface{}{
			"conversation_id":           cid,
			"remembered_at":             fmt.Sprintf("%d", time.Now().Unix()),
			"importance":                fmt.Sprintf("%.1f", ref.Importance),
			"knowledge_memory_summary":  "true", // 标记为反射摘要记忆
			"knowledge_memory_themes":   ref.Themes,
			"knowledge_memory_entities": ref.Entities,
			// 可迁移范围：global（跨对话召回）/ conversation（仅记录时所在对话召回）。
			// 分身回复按对话召回时据此过滤，避免把 A-B 对话的专属互动带进其他对话。
			"knowledge_memory_scope": ref.Scope,
		},
	})
	if err != nil {
		return false, err
	}
	logger.WithModule("AvatarMemoryService").Info("记忆反射落库",
		"userID", userID, "content", ref.Summary)
	return true, nil
}

// Recall 全局召回该用户的分身记忆（不分对话）。用于写入去重、记忆合并、手动检索等场景。
func (s *AvatarMemoryService) Recall(userID uint, query string, topK int) ([]SearchResult, error) {
	return s.recall(userID, query, topK, 0, false)
}

// RecallForConversation 在指定对话内召回分身记忆，供分身回复使用：
// global 级记忆（事实/偏好/决定/FAQ）任意对话可召回；conversation 级记忆（针对特定对象的
// 回应/私约/承诺）仅在其记录时所在对话（metadata.conversation_id 匹配）可召回——
// 避免 A-B 对话里记下的专属互动被带进 C-B 对话，造成答非所问或泄露语境。
func (s *AvatarMemoryService) RecallForConversation(userID, conversationID uint, query string, topK int) ([]SearchResult, error) {
	return s.recall(userID, query, topK, conversationID, true)
}

// recall 公共召回实现。filterByConversation=true 时对 conversation 级记忆按 conversationID 过滤；
// 请求时多召回一倍，避免过滤后不足 TopK。
func (s *AvatarMemoryService) recall(userID uint, query string, topK int, conversationID uint, filterByConversation bool) ([]SearchResult, error) {
	queryTopK := topK
	if filterByConversation {
		// TODO(记忆): TopK 加倍是经验值。当 conversation 级记忆占比高时，过滤后仍可能不足
		// TopK，导致回复注入变少。可考虑过滤后不足时用候选补齐或调高倍数。
		queryTopK = topK * 2
	}
	resp, err := s.db.SearchMemory(types.MemorySearchRequest{
		Query:     query,
		UserID:    fmt.Sprintf("%d", userID),
		Scope:     "user",
		Namespace: "avatar",
		TopK:      queryTopK,
		// 提升重要度与新颖度的排序权重：反射落库的重要记忆（Importance 1-5 → [0,1]）与
		// 较新的记忆在召回时更靠前，避免被默认权重（importance 0.10 / recency 0.05）稀释。
		SemanticWeight:   0.55,
		LexicalWeight:    0.15,
		ImportanceWeight: 0.20,
		RecencyWeight:    0.10,
	})
	if err != nil {
		return nil, fmt.Errorf("检索记忆失败: %w", err)
	}

	currentConv := fmt.Sprintf("%d", conversationID)
	var results []SearchResult
	for _, hit := range resp.Results {
		metadataStr := make(map[string]string)
		for k, v := range hit.Memory.Metadata {
			// 仅取字符串值用于过滤判断：scope 与 conversation_id 当前均以字符串落库，一致。
			// TODO(记忆): 若未来有其他路径以非字符串存 conversation_id，这里会判非当前对话而
			// 剔除（安全方向），但需确认预期的数字类型处理。
			if s, ok := v.(string); ok {
				metadataStr[k] = s
			}
		}
		// 对话级记忆过滤：scope=conversation 且记录对话与当前对话不一致时剔除。
		// scope 缺失/为空视为 global（保守不丢知识）；conversation 级但缺 conversation_id
		// 无法匹配，一并剔除（安全）。
		if filterByConversation && metadataStr["knowledge_memory_scope"] == "conversation" &&
			metadataStr["conversation_id"] != currentConv {
			continue
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
值得记忆：个人偏好、重要决定、项目关键信息、约定事项、答疑形成的可复用知识（步骤、配置、口径、规范）。
` + rememberVerdictNegativeClause
	return evaluateRemember(s.aiService, prompt, message)
}

// ForgetMemory 删除单条用户记忆（带归属校验）。
// 保留旧方法名兼容内部调用，实际委托给 DeleteMemory 统一走归属校验。
func (s *AvatarMemoryService) ForgetMemory(userID uint, memoryDocID string) error {
	return s.DeleteMemory(userID, memoryDocID)
}

func (s *AvatarMemoryService) GetMemoryCount(userID uint) (int64, error) {
	memories, err := s.GetUserMemories(userID, 100000)
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
	Nodes    []MemoryGraphNode   `json:"nodes"`
	Edges    []MemoryGraphEdge   `json:"edges"`
	Memories []MemoryGraphMemory `json:"memories"`
}

type MemoryGraphNode struct {
	ID    string            `json:"id"`
	Name  string            `json:"name"`
	Type  string            `json:"type"` // entity | theme | note
	Count int               `json:"count"`
	Data  map[string]string `json:"data,omitempty"`
}

type MemoryGraphEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Weight int    `json:"weight"`
}

type MemoryGraphMemory struct {
	ID      string `json:"id"`
	Content string `json:"content"`
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
	memories, err := s.GetUserMemories(userID, 100000)
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
	memories, err := s.GetUserMemories(userID, 100000)
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
