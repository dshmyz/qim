package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/pkg/logger"
)

// NoteVectorService 笔记向量化和检索服务
type NoteVectorService struct {
	vectorSvc  *VectorService
	aiService  *ai.AIService
	entityCache sync.Map // key: "userID:noteID" → value: string (逗号分隔实体)
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

	// 先删除旧向量（同时清缓存）
	s.deleteNoteVectors(ctx, collectionName, noteID)
	s.entityCache.Delete(fmt.Sprintf("%d:%d", userID, noteID))

	// 按标题切片
	chunks := SplitMarkdownByHeading(content)
	if len(chunks) == 0 {
		chunks = []Chunk{{Content: content, Title: title}}
	}

	// 预提取实体：从完整内容提取一次，写入每个 chunk 的 metadata。
	// 这样 BuildNoteGraph 从 metadata 读实体，零 LLM 调用。
	entities := ""
	if len(content) >= 10 {
		ents := extractNoteEntities(s.aiService, content)
		if len(ents) > 0 {
			entities = strings.Join(ents, ",")
		}
	}

	for i, chunk := range chunks {
		if len(chunk.Content) < 10 {
			continue // 跳过太短的片段
		}

		// 生成向量（aiService 为 nil 时跳过，仅供测试用）
		if s.aiService == nil {
			continue
		}
		embedding, err := s.aiService.Embed(chunk.Content)
		if err != nil {
			logger.WithModule("NoteVectorService").Error("笔记向量化失败", "noteID", noteID, "chunk", i, "error", err)
			continue
		}
		if len(embedding) == 0 {
			logger.WithModule("NoteVectorService").Warn("Embed 返回空向量", "noteID", noteID, "chunk", i)
			continue
		}

		// 存储向量
		docID := fmt.Sprintf("note_%d_chunk_%d", noteID, i)
		metadata := map[string]string{
			"note_id":  fmt.Sprintf("%d", noteID),
			"chunk_id": fmt.Sprintf("%d", i),
			"title":    title,
			"type":     "note",
			"entities": entities, // 预提取实体，供图谱直接使用
		}

		if err := s.vectorSvc.AddVector(ctx, collectionName, docID, embedding, chunk.Content, metadata); err != nil {
			logger.WithModule("NoteVectorService").Error("笔记向量存储失败", "noteID", noteID, "chunk", i, "error", err)
		}
	}

	logger.WithModule("NoteVectorService").Info("笔记向量化完成", "noteID", noteID, "chunkCount", len(chunks))
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
	s.entityCache.Delete(fmt.Sprintf("%d:%d", userID, noteID))
	ctx := context.Background()
	collectionName := fmt.Sprintf("user_notes_%d", userID)
	return s.deleteNoteVectors(ctx, collectionName, noteID)
}

func (s *NoteVectorService) deleteNoteVectors(ctx context.Context, collectionName string, noteID uint) error {
	noteIDStr := fmt.Sprintf("%d", noteID)
	filter := map[string]string{"note_id": noteIDStr}
	deleted, err := s.vectorSvc.DeleteByFilter(ctx, collectionName, filter)
	if err != nil {
		return fmt.Errorf("删除笔记 %d 的向量失败: %w", noteID, err)
	}
	logger.WithModule("NoteVectorService").Info("删除笔记向量", "noteID", noteID, "deletedCount", deleted)
	return nil
}

// extractNoteEntities 用 LLM 从笔记内容提取实体名称（人名、项目名、组织名等）。
// 返回去重后的实体列表；LLM 调用失败或无有效结果时降级返回空列表，不阻断主流程。
func extractNoteEntities(aiService *ai.AIService, content string) []string {
	if aiService == nil || len(content) < 10 {
		return nil
	}
	prompt := `从以下笔记内容中提取关键实体（人名、项目名、组织名、产品名等），
仅返回 JSON，形如 {"entities": ["张三", "项目X"]}，最多 15 个实体。
无法确定时返回空数组。

笔记内容：
` + content

	msgs := []ai.Message{{Role: "user", Content: prompt}}
	out, err := aiService.GetCompletion(ai.TaskTypeAnalysis, msgs)
	if err != nil {
		return nil
	}
	var raw struct {
		Entities []string `json:"entities"`
	}
	// 从 LLM 输出中提取 JSON（容忍前后多余文本）
	if idx := strings.Index(out, "{"); idx >= 0 {
		if jdx := strings.Index(out[idx:], "}"); jdx > 0 {
			json.Unmarshal([]byte(out[idx:idx+jdx+1]), &raw)
		}
	}
	if len(raw.Entities) == 0 {
		return nil
	}
	seen := make(map[string]bool)
	var result []string
	for _, e := range raw.Entities {
		e = strings.TrimSpace(e)
		if e != "" && !seen[e] && len(e) <= 20 {
			seen[e] = true
			result = append(result, e)
		}
	}
	return result
}

// BuildNoteGraph 为笔记来源构建实体共现拓扑图（懒加载，不持久化）。
// 每用户独立子图（节点 ID 带 u{userID}: 前缀，避免跨用户实体污染）。
// 实体提取用 LLM，每次查看现算（图谱页切换不频繁，可接受）。
func (s *NoteVectorService) BuildNoteGraph(userID uint, maxNodes int) (*MemoryGraph, error) {
	if s.vectorSvc == nil {
		logger.WithModule("NoteGraph").Warn("vectorSvc 为空，跳过图谱构建")
		return &MemoryGraph{}, nil
	}
	ctx := context.Background()
	collection := fmt.Sprintf("user_notes_%d", userID)
	results, err := s.vectorSvc.GetByCollection(ctx, collection, maxNodes*10)
	if err != nil {
		logger.WithModule("NoteGraph").Warn("GetByCollection 失败", "collection", collection, "error", err)
		return &MemoryGraph{}, nil
	}
	if len(results) == 0 {
		logger.WithModule("NoteGraph").Info("笔记向量为空", "collection", collection)
		return &MemoryGraph{}, nil
	}
	// 按 note_id 分组（一篇笔记可能有多个 chunk）
	type noteGroup struct {
		noteID   string
		title    string
		chunks   []string
		entities string // 逗号分隔的实体列表（从 metadata 预提取）
	}
	groupMap := make(map[string]*noteGroup)
	for _, r := range results {
		noteID := r.Metadata["note_id"]
		if noteID == "" {
			noteID = r.ID
		}
		title := r.Metadata["title"]
		if g, ok := groupMap[noteID]; ok {
			if title == "" {
				title = g.title
			}
			g.chunks = append(g.chunks, r.Content)
			g.title = title
		} else {
			groupMap[noteID] = &noteGroup{
				noteID: noteID,
				title:  title,
				chunks: []string{r.Content},
			}
		}
		// 合并跨 chunk 的实体（预提取方案：实体已存入 metadata["entities"]）
		if ents := r.Metadata["entities"]; ents != "" && groupMap[noteID] != nil {
			existing := groupMap[noteID].entities
			for _, e := range strings.Split(ents, ",") {
				e = strings.TrimSpace(e)
				if e != "" && !strings.Contains(","+existing+",", ","+e+",") {
					if existing != "" {
						existing += ","
					}
					existing += e
				}
			}
			groupMap[noteID].entities = existing
		}
	}
	type noteItem struct {
		noteID string
		title  string
		entities []string
	}
	// 并发提取实体：无预提取 metadata 的笔记需要调 LLM。
	// 先查进程内缓存（同一笔记首次提取后缓存，后续秒读），缓存未命中才调 LLM。
	type llmJob struct {
		noteID string
		chunks []string
	}
	var llmJobs []llmJob
	for _, g := range groupMap {
		if g.entities == "" && len(g.chunks) > 0 {
			// 先查缓存
			cacheKey := fmt.Sprintf("%d:%s", userID, g.noteID)
			if cached, ok := s.entityCache.Load(cacheKey); ok {
				g.entities = cached.(string)
				continue
			}
			llmJobs = append(llmJobs, llmJob{noteID: g.noteID, chunks: g.chunks})
		}
	}
	if len(llmJobs) > 0 && s.aiService != nil {
		var mu sync.Mutex
		var wg sync.WaitGroup
		sem := make(chan struct{}, 5) // 最多 5 路并发，防打爆 AI 服务
		for _, job := range llmJobs {
			wg.Add(1)
			go func(j llmJob) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()
				joined := strings.Join(j.chunks, "\n")
				if len(joined) > 2000 {
					joined = joined[:2000]
				}
				entStrs := extractNoteEntities(s.aiService, joined)
				if len(entStrs) == 0 {
					return
				}
				entStr := strings.Join(entStrs, ",")
				// 写入 groupMap + 缓存
				cacheKey := fmt.Sprintf("%d:%s", userID, j.noteID)
				s.entityCache.Store(cacheKey, entStr)
				mu.Lock()
				if g, ok := groupMap[j.noteID]; ok {
					g.entities = entStr
				}
				mu.Unlock()
			}(job)
		}
		wg.Wait()
	}
	// 读取实体（预提取 + LLM 提取合并后）
	var notes []noteItem
	for _, g := range groupMap {
		var entList []string
		if g.entities != "" {
			entList = strings.Split(g.entities, ",")
		}
		notes = append(notes, noteItem{noteID: g.noteID, title: g.title, entities: entList})
	}
	if len(notes) > maxNodes {
		notes = notes[:maxNodes]
	}
	graph := &MemoryGraph{
		Nodes:    []MemoryGraphNode{},
		Edges:    []MemoryGraphEdge{},
		Memories: []MemoryGraphMemory{},
	}
	nodeIdx := make(map[string]int)
	addNode := func(id, label, typ string, data map[string]string) {
		if _, ok := nodeIdx[id]; ok {
			graph.Nodes[nodeIdx[id]].Count++
		} else {
			nodeIdx[id] = len(graph.Nodes)
			graph.Nodes = append(graph.Nodes, MemoryGraphNode{ID: id, Name: label, Type: typ, Count: 1, Data: data})
		}
	}
	edgeKey := func(a, b string) string {
		if a < b {
			return a + "|" + b
		}
		return b + "|" + a
	}
	edgeW := make(map[string]int)
	for _, n := range notes {
		noteID := fmt.Sprintf("note:%s:%s", fmt.Sprintf("u%d", userID), n.noteID)
		addNode(noteID, n.title, "note", map[string]string{"content": n.title})
		graph.Memories = append(graph.Memories, MemoryGraphMemory{ID: noteID, Content: n.title, Terms: n.entities})
		for _, e := range n.entities {
			eID := fmt.Sprintf("entity:%s:%s", fmt.Sprintf("u%d", userID), e)
			addNode(eID, e, "entity", nil)
			addEdgeKey := edgeKey(noteID, eID)
			edgeW[addEdgeKey]++
		}
	}
	// 实体共现边：同一笔记内多个实体两两配对 co_occurs，与群聊图谱对齐，
	// 使不同笔记共享同一实体时能交叉连线形成网状拓扑（而非孤立星团）。
	entByNote := make(map[string][]string) // noteID → entity names
	for _, n := range notes {
		if len(n.entities) >= 2 {
			entByNote[n.noteID] = n.entities
		}
	}
	for _, ents := range entByNote {
		for i := 0; i < len(ents); i++ {
			for j := i + 1; j < len(ents); j++ {
				aID := fmt.Sprintf("entity:%s:%s", fmt.Sprintf("u%d", userID), ents[i])
				bID := fmt.Sprintf("entity:%s:%s", fmt.Sprintf("u%d", userID), ents[j])
				ek := edgeKey(aID, bID)
				edgeW[ek]++
			}
		}
	}
	for k, w := range edgeW {
		parts := strings.Split(k, "|")
		if len(parts) == 2 {
			graph.Edges = append(graph.Edges, MemoryGraphEdge{Source: parts[0], Target: parts[1], Weight: w})
		}
	}
	return graph, nil
}
