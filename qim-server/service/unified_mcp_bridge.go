package service

import (
	"context"
	"fmt"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/pkg/logger"

	"github.com/liliang-cn/cortexdb/v2/pkg/cortexdb"
)

type UnifiedMCPBridge struct {
	qimMCPSrv *ai.MCPServer
	cortexDB  *cortexdb.DB
}

func NewUnifiedMCPBridge(qimMCPSrv *ai.MCPServer, cortexDB *cortexdb.DB) *UnifiedMCPBridge {
	bridge := &UnifiedMCPBridge{
		qimMCPSrv: qimMCPSrv,
		cortexDB:  cortexDB,
	}

	bridge.registerKnowledgeTools()
	logger.WithModule("UnifiedMCPBridge").Info("桥接完成",
		"toolCount", len(qimMCPSrv.ListTools()))
	return bridge
}

func (b *UnifiedMCPBridge) GetQIMServer() *ai.MCPServer {
	return b.qimMCPSrv
}

func (b *UnifiedMCPBridge) registerKnowledgeTools() {
	b.qimMCPSrv.RegisterTool(&KnowledgeSearchTool{bridge: b})
	b.qimMCPSrv.RegisterTool(&KnowledgeSaveTool{bridge: b})
	b.qimMCPSrv.RegisterTool(&MemorySearchTool{bridge: b})
}

// ── knowledge_search ──

type KnowledgeSearchTool struct{ bridge *UnifiedMCPBridge }

func (t *KnowledgeSearchTool) Name() string        { return "knowledge_search" }
func (t *KnowledgeSearchTool) Description() string  { return "在群知识库中语义搜索文档内容，仅在群聊中使用。如果不在群聊中，请使用自身知识直接回答用户" }
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

	resp, err := t.bridge.cortexDB.SearchKnowledge(context.Background(), cortexdb.KnowledgeSearchRequest{
		Query:      query,
		Collection: collection,
		TopK:       topK,
	})
	if err != nil {
		return nil, err
	}

	type hit struct {
		KnowledgeID string            `json:"knowledge_id"`
		Title       string            `json:"title"`
		Snippet     string            `json:"snippet"`
		Score       float64           `json:"score"`
		Metadata    map[string]string `json:"metadata"`
	}
	hits := make([]hit, 0, len(resp.Results))
	for _, r := range resp.Results {
		hits = append(hits, hit{
			KnowledgeID: r.KnowledgeID,
			Title:       r.Title,
			Snippet:     r.Snippet,
			Score:       r.Score,
			Metadata:    r.Metadata,
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

type KnowledgeSaveTool struct{ bridge *UnifiedMCPBridge }

func (t *KnowledgeSaveTool) Name() string        { return "knowledge_save" }
func (t *KnowledgeSaveTool) Description() string  { return "将文档内容存入知识库并自动向量化+切片，可选实体关系抽取" }
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

	resp, err := t.bridge.cortexDB.SaveKnowledge(context.Background(), cortexdb.KnowledgeSaveRequest{
		KnowledgeID: knowledgeID,
		Title:       title,
		Content:     content,
		Collection:  collection,
		ChunkSize:   chunkSize,
	})
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"knowledge_id":   resp.Knowledge.ID,
		"chunk_count":    len(resp.Knowledge.ChunkIDs),
		"entity_count":   len(resp.EntityNodeIDs),
		"relation_count": len(resp.RelationEdgeIDs),
	}, nil
}

// ── memory_search ──

type MemorySearchTool struct{ bridge *UnifiedMCPBridge }

func (t *MemorySearchTool) Name() string        { return "memory_search" }
func (t *MemorySearchTool) Description() string  { return "在用户长期记忆中语义搜索，用于分身回忆之前的对话和偏好" }
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

	resp, err := t.bridge.cortexDB.SearchMemory(context.Background(), cortexdb.MemorySearchRequest{
		Query:     query,
		UserID:    userIDStr,
		Scope:     cortexdb.MemoryScopeUser,
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