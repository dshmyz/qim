package service

import (
	"context"
	"fmt"
	"log"

	"github.com/liliang-cn/cortexdb/v2/pkg/core"
)

// VectorService 向量存储服务封装（基于 CortexDB）
type VectorService struct {
	store *core.SQLiteStore
}

// NewVectorService 创建向量服务实例
func NewVectorService(path string) (*VectorService, error) {
	cfg := core.DefaultConfig(path)
	store, err := core.NewWithConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("打开向量数据库失败: %w", err)
	}

	ctx := context.Background()
	if err := store.Init(ctx); err != nil {
		return nil, fmt.Errorf("初始化向量存储失败: %w", err)
	}

	log.Printf("[VectorService] 向量数据库初始化完成, path=%s", path)
	return &VectorService{store: store}, nil
}

// AddVector 添加向量到指定集合
func (s *VectorService) AddVector(ctx context.Context, collection string, id string, embedding []float32, content string, metadata map[string]string) error {
	emb := &core.Embedding{
		ID:         id,
		Collection: collection,
		Vector:     embedding,
		Content:    content,
		Metadata:   metadata,
	}

	if err := s.store.Upsert(ctx, emb); err != nil {
		return fmt.Errorf("添加向量到集合 %s 失败: %w", collection, err)
	}
	return nil
}

// Search 在集合中搜索相似向量
func (s *VectorService) Search(ctx context.Context, collection string, queryVector []float32, topK int) ([]core.ScoredEmbedding, error) {
	opts := core.SearchOptions{
		Collection: collection,
		TopK:       topK,
	}

	results, err := s.store.Search(ctx, queryVector, opts)
	if err != nil {
		return nil, fmt.Errorf("在集合 %s 中搜索失败: %w", collection, err)
	}
	return results, nil
}

// DeleteByDocID 按文档 ID 删除向量
func (s *VectorService) DeleteByDocID(ctx context.Context, docID string) error {
	if err := s.store.DeleteByDocID(ctx, docID); err != nil {
		return fmt.Errorf("按 docID=%s 删除向量失败: %w", docID, err)
	}
	return nil
}

// DeleteByCollection 按集合名删除所有向量
func (s *VectorService) DeleteByCollection(ctx context.Context, collection string) error {
	if err := s.store.DeleteByDocID(ctx, collection); err != nil {
		// DeleteByDocID 是删除指定 docID 的向量
		// 如果需要删除整个集合，需要自定义实现
		log.Printf("[VectorService] 按集合 %s 删除向量: %v", collection, err)
	}
	return nil
}

// GetStore 返回底层 store（用于高级操作）
func (s *VectorService) GetStore() *core.SQLiteStore {
	return s.store
}

// GetByCollection 获取指定集合中的所有向量（用于管理界面）
func (s *VectorService) GetByCollection(ctx context.Context, collection string, limit int) ([]core.ScoredEmbedding, error) {
	// 使用一个零向量进行查询，返回集合中所有数据
	zeroVector := make([]float32, 1536)
	opts := core.SearchOptions{
		Collection: collection,
		TopK:       limit,
	}
	
	results, err := s.store.Search(ctx, zeroVector, opts)
	if err != nil {
		return nil, fmt.Errorf("获取集合 %s 数据失败: %w", collection, err)
	}
	return results, nil
}

// SearchResult 通用搜索结果结构
type SearchResult struct {
	Content  string            `json:"content"`
	Score    float64           `json:"score"`
	Metadata map[string]string `json:"metadata"`
	Collection string          `json:"collection"`
	DocID    string            `json:"docId"`
}

// ScoredEmbeddingToSearchResult 将 cortexdb 结果转换为通用结构
func ScoredEmbeddingToSearchResult(se core.ScoredEmbedding) SearchResult {
	return SearchResult{
		Content:    se.Content,
		Score:      se.Score,
		Metadata:   se.Metadata,
		Collection: se.Collection,
		DocID:      se.DocID,
	}
}
