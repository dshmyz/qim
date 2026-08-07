package service

import (
	"fmt"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/pkg/logger"

	"github.com/dshmyz/gracedb/pkg/gracedb"
	"github.com/dshmyz/gracedb/pkg/types"
)

type UnifiedToolBridge struct {
	registry  *ai.ToolRegistry
	db        *gracedb.DB
	aiService *ai.AIService
}

func NewUnifiedToolBridge(registry *ai.ToolRegistry, db *gracedb.DB, aiService *ai.AIService) *UnifiedToolBridge {
	bridge := &UnifiedToolBridge{
		registry:  registry,
		db:        db,
		aiService: aiService,
	}

	bridge.registerKnowledgeTools()
	logger.WithModule("UnifiedToolBridge").Info("工具注册完成",
		"toolCount", len(registry.ListTools()))
	return bridge
}

func (b *UnifiedToolBridge) GetRegistry() *ai.ToolRegistry {
	return b.registry
}

func (b *UnifiedToolBridge) registerKnowledgeTools() {
	b.registry.RegisterTool(&KnowledgeSearchTool{bridge: b})
	b.registry.RegisterTool(&KnowledgeSaveTool{bridge: b})
	b.registry.RegisterTool(&MemorySearchTool{bridge: b})
}

// ── knowledge_search ──

type KnowledgeSearchTool struct{ bridge *UnifiedToolBridge }

func (t *KnowledgeSearchTool) Name() string { return "knowledge_search" }
func (t *KnowledgeSearchTool) Description() string {
	return "在群知识库中语义搜索文档内容，仅在群聊中使用。如果不在群聊中，请使用自身知识直接回答用户"
}
func (t *KnowledgeSearchTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query":      map[string]interface{}{"type": "string", "description": "搜索查询词，用于语义检索群知识库"},
			"collection": map[string]interface{}{"type": "string", "description": "集合名称（如 group_1），不传则自动使用当前群聊 ID"},
			"top_k":      map[string]interface{}{"type": "integer", "description": "返回结果数量，默认5"},
		},
		"required": []string{"query"},
	}
}

func (t *KnowledgeSearchTool) Execute(params map[string]interface{}, ctx *ai.CallerContext) (interface{}, error) {
	query, _ := params["query"].(string)
	collection, _ := params["collection"].(string)
	topK := 5
	if k, ok := params["top_k"].(float64); ok {
		topK = int(k)
	}
	if query == "" {
		return nil, fmt.Errorf("query 不能为空")
	}

	// 如果没传 collection，从 CallerContext 的 GroupID 自动推导
	if collection == "" && ctx != nil && ctx.GroupID > 0 {
		collection = fmt.Sprintf("group_%d", ctx.GroupID)
	}
	if collection == "" {
		return map[string]interface{}{
			"results": []interface{}{},
			"total":   0,
			"note":    "当前不在群聊中，群知识库不可用，请使用自身知识回答用户",
		}, nil
	}

	// 语义 + 词法（FTS）混合召回，同群知识库检索路径保持一致
	fetchK := topK * 3
	if fetchK <= 0 {
		fetchK = 5
	}

	queryVec, err := t.bridge.aiService.Embed(query)
	if err != nil {
		return nil, fmt.Errorf("生成查询向量失败: %v", err)
	}
	var semanticRes, ftsRes []types.ScoredEmbedding
	if sem, se := t.bridge.db.Search(collection, queryVec, types.SearchOptions{TopK: fetchK}); se == nil {
		semanticRes = sem
	}
	if fts, fe := t.bridge.db.SearchFTSWithContent(collection, query, fetchK); fe == nil {
		ftsRes = fts
	}

	var scored []types.ScoredEmbedding
	switch {
	case len(semanticRes) == 0 && len(ftsRes) == 0:
		scored = nil
	case len(semanticRes) == 0:
		scored = hybridDisplayScores(ftsRes, nil)
	case len(ftsRes) == 0:
		scored = semanticRes
	default:
		scored = hybridDisplayScores(mergeRRF(semanticRes, ftsRes, topK), semanticRes)
	}

	type hit struct {
		KnowledgeID string            `json:"knowledge_id"`
		Title       string            `json:"title"`
		Snippet     string            `json:"snippet"`
		Score       float64           `json:"score"`
		Metadata    map[string]string `json:"metadata"`
	}
	hits := make([]hit, 0, len(scored))
	for _, r := range scored {
		metadata := r.Metadata
		if metadata == nil {
			metadata = make(map[string]string)
		}
		hits = append(hits, hit{
			KnowledgeID: r.DocID,
			Title:       metadata["title"],
			Snippet:     r.Content,
			Score:       float64(r.Score),
			Metadata:    metadata,
		})
	}
	if len(hits) == 0 {
		return map[string]interface{}{
			"results": hits,
			"total":   0,
			"note":    "群知识库中未找到相关内容，请使用自身知识回答用户",
		}, nil
	}
	return map[string]interface{}{"results": hits, "total": len(hits)}, nil
}

type KnowledgeSaveTool struct{ bridge *UnifiedToolBridge }

func (t *KnowledgeSaveTool) Name() string { return "knowledge_save" }
func (t *KnowledgeSaveTool) Description() string {
	return "将文档内容存入知识库并自动向量化+切片"
}
func (t *KnowledgeSaveTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"knowledge_id": map[string]interface{}{"type": "string", "description": "知识ID（唯一标识）", "required": true},
		"title":        map[string]interface{}{"type": "string", "description": "标题", "required": true},
		"content":      map[string]interface{}{"type": "string", "description": "文本内容", "required": true},
		"collection":   map[string]interface{}{"type": "string", "description": "集合名称（如 group_1）", "required": false},
		"chunk_size":   map[string]interface{}{"type": "integer", "description": "切片大小（字符数）", "required": false, "default": 800},
	}
}

func (t *KnowledgeSaveTool) Execute(params map[string]interface{}, ctx *ai.CallerContext) (interface{}, error) {
	knowledgeID, _ := params["knowledge_id"].(string)
	title, _ := params["title"].(string)
	content, _ := params["content"].(string)
	collection, _ := params["collection"].(string)
	chunkSize := 800
	if cs, ok := params["chunk_size"].(float64); ok {
		chunkSize = int(cs)
	}
	if knowledgeID == "" || content == "" {
		return nil, fmt.Errorf("knowledge_id 和 content 不能为空")
	}
	if collection == "" {
		return nil, fmt.Errorf("collection 不能为空")
	}

	// 确保集合存在（gracedb Upsert 不会自动建集合）
	if err := ensureGracedbCollection(t.bridge.db, collection); err != nil {
		return nil, fmt.Errorf("创建知识库集合失败: %v", err)
	}

	chunks := ChunkDocument(content, chunkSize)
	if len(chunks) == 0 {
		chunks = []Chunk{{Content: content, Title: title}}
	}

	// 逐块向量化（无批量 embedding API，串行调用），收集后一次性批量写入
	type pendingChunk struct {
		docID string
		vec   []float32
		text  string
	}
	var pending []pendingChunk
	for i, chunk := range chunks {
		if len(chunk.Content) < 10 {
			continue
		}
		embedding, err := t.bridge.aiService.Embed(chunk.Content)
		if err != nil {
			return nil, fmt.Errorf("切片向量化失败: %v", err)
		}
		pending = append(pending, pendingChunk{
			docID: fmt.Sprintf("%s_chunk_%d", knowledgeID, i),
			vec:   embedding,
			text:  chunk.Content,
		})
	}

	if len(pending) > 0 {
		vectors := make([][]float32, 0, len(pending))
		contents := make([]string, 0, len(pending))
		docIDs := make([]string, 0, len(pending))
		metas := make([]map[string]string, 0, len(pending))
		for _, p := range pending {
			vectors = append(vectors, p.vec)
			contents = append(contents, p.text)
			docIDs = append(docIDs, p.docID)
			metas = append(metas, map[string]string{"title": title})
		}
		if err := t.bridge.db.UpsertBatch(collection, vectors, contents, docIDs, metas); err != nil {
			return nil, fmt.Errorf("批量存储失败: %v", err)
		}
	}

	return map[string]interface{}{
		"knowledge_id": knowledgeID,
		"chunk_count":  len(pending),
	}, nil
}

// ── memory_search ──

type MemorySearchTool struct{ bridge *UnifiedToolBridge }

func (t *MemorySearchTool) Name() string { return "memory_search" }
func (t *MemorySearchTool) Description() string {
	return "在用户长期记忆中语义搜索，用于分身回忆之前的对话和偏好"
}
func (t *MemorySearchTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query":   map[string]interface{}{"type": "string", "description": "搜索查询词，用于语义检索用户记忆"},
			"user_id": map[string]interface{}{"type": "string", "description": "用户ID，不传则自动使用当前调用者ID"},
			"top_k":   map[string]interface{}{"type": "integer", "description": "返回结果数量，默认3"},
		},
		"required": []string{"query"},
	}
}

func (t *MemorySearchTool) Execute(params map[string]interface{}, ctx *ai.CallerContext) (interface{}, error) {
	query, _ := params["query"].(string)
	userIDStr, _ := params["user_id"].(string)
	topK := 3
	if k, ok := params["top_k"].(float64); ok {
		topK = int(k)
	}
	if query == "" {
		return nil, fmt.Errorf("query 不能为空")
	}

	// 如果没传 user_id，从 CallerContext 自动填充
	if userIDStr == "" && ctx != nil && ctx.UserID > 0 {
		userIDStr = fmt.Sprintf("%d", ctx.UserID)
	}
	if userIDStr == "" {
		return nil, fmt.Errorf("user_id 不能为空，请确保 CallerContext 中有用户ID")
	}

	resp, err := t.bridge.db.SearchMemory(types.MemorySearchRequest{
		Query:     query,
		UserID:    userIDStr,
		Scope:     "user",
		Namespace: "avatar",
		TopK:      topK,
	})
	if err != nil {
		return nil, err
	}

	type hit struct {
		Content string  `json:"content"`
		Score   float64 `json:"score"`
	}
	hits := make([]hit, 0, len(resp.Results))
	for _, r := range resp.Results {
		hits = append(hits, hit{
			Content: r.Memory.Content,
			Score:   r.Score,
		})
	}
	return map[string]interface{}{"results": hits, "total": len(hits)}, nil
}
