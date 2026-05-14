package service

import (
	"context"
	"fmt"
	"log"

	"qim-server/ai"

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
	log.Printf("[UnifiedMCPBridge] 桥接完成：QIM 工具 %d 个 → 已添加 cortexdb 知识工具",
		len(qimMCPSrv.ListTools()))
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
func (t *KnowledgeSearchTool) Description() string  { return "在群知识库中语义搜索文档内容，支持向量+图谱增强检索" }
func (t *KnowledgeSearchTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"query":      map[string]interface{}{"type": "string", "description": "搜索查询", "required": true},
		"collection": map[string]interface{}{"type": "string", "description": "集合名称（如 group_1）", "required": true},
		"top_k":      map[string]interface{}{"type": "integer", "description": "返回结果数", "required": false, "default": 5},
	}
}

func (t *KnowledgeSearchTool) Execute(params map[string]interface{}) (interface{}, error) {
	query, _ := params["query"].(string)
	collection, _ := params["collection"].(string)
	topK := 5
	if k, ok := params["top_k"].(float64); ok {
		topK = int(k)
	}
	if query == "" || collection == "" {
		return nil, fmt.Errorf("query 和 collection 不能为空")
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
	return map[string]interface{}{"results": hits, "total": len(hits)}, nil
}

// ── knowledge_save ──

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

func (t *KnowledgeSaveTool) Execute(params map[string]interface{}) (interface{}, error) {
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
		"knowledge_id":  resp.Knowledge.ID,
		"chunk_count":   len(resp.Knowledge.ChunkIDs),
		"entity_count":  len(resp.EntityNodeIDs),
		"relation_count": len(resp.RelationEdgeIDs),
	}, nil
}

// ── memory_search ──

type MemorySearchTool struct{ bridge *UnifiedMCPBridge }

func (t *MemorySearchTool) Name() string        { return "memory_search" }
func (t *MemorySearchTool) Description() string  { return "在用户长期记忆中语义搜索，用于分身回忆之前的对话和偏好" }
func (t *MemorySearchTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"query":  map[string]interface{}{"type": "string", "description": "搜索查询", "required": true},
		"user_id": map[string]interface{}{"type": "string", "description": "用户ID", "required": true},
		"top_k":  map[string]interface{}{"type": "integer", "description": "返回结果数", "required": false, "default": 3},
	}
}

func (t *MemorySearchTool) Execute(params map[string]interface{}) (interface{}, error) {
	query, _ := params["query"].(string)
	userID, _ := params["user_id"].(string)
	topK := 3
	if k, ok := params["top_k"].(float64); ok {
		topK = int(k)
	}
	if query == "" || userID == "" {
		return nil, fmt.Errorf("query 和 user_id 不能为空")
	}

	resp, err := t.bridge.cortexDB.SearchMemory(context.Background(), cortexdb.MemorySearchRequest{
		Query:     query,
		UserID:    userID,
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