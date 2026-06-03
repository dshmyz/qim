package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dshmyz/qim/qim-server/pkg/logger"

	"github.com/liliang-cn/cortexdb/v2/pkg/core"
	"github.com/liliang-cn/cortexdb/v2/pkg/cortexdb"
)

type VectorService struct {
	db *cortexdb.DB
}

func NewVectorService(path string, embedder cortexdb.Embedder) (*VectorService, error) {
	// 确保数据目录存在
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("创建向量数据目录失败: %w", err)
	}

	cfg := cortexdb.DefaultConfig(path)

	var opts []cortexdb.Option
	if embedder != nil {
		opts = append(opts, cortexdb.WithEmbedder(embedder))
	}

	db, err := cortexdb.Open(cfg, opts...)
	if err != nil {
		return nil, fmt.Errorf("打开向量数据库失败: %w", err)
	}

	logger.WithModule("VectorService").Info("向量数据库初始化完成", "path", path)
	return &VectorService{db: db}, nil
}

// AddVector 添加向量到指定集合（自动创建集合）
func (s *VectorService) AddVector(ctx context.Context, collection string, id string, embedding []float32, content string, metadata map[string]string) error {
	if err := s.EnsureCollection(ctx, collection, len(embedding)); err != nil {
		return fmt.Errorf("确保集合存在失败: %w", err)
	}

	emb := &core.Embedding{
		ID:         id,
		Collection: collection,
		Vector:     embedding,
		Content:    content,
		Metadata:   metadata,
	}
	return s.db.Vector().Upsert(ctx, emb)
}

// EnsureCollection 确保集合存在，不存在则创建
func (s *VectorService) EnsureCollection(ctx context.Context, name string, dimensions int) error {
	_, err := s.db.Vector().CreateCollection(ctx, name, dimensions)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			return nil
		}
		return err
	}
	return nil
}

// Search 在集合中搜索相似向量
func (s *VectorService) Search(ctx context.Context, collection string, queryVector []float32, topK int) ([]core.ScoredEmbedding, error) {
	opts := core.SearchOptions{
		Collection: collection,
		TopK:       topK,
	}
	return s.db.Vector().Search(ctx, queryVector, opts)
}

// DeleteByDocID 按文档 ID 删除向量
func (s *VectorService) DeleteByDocID(ctx context.Context, docID string) error {
	return s.db.Vector().DeleteByDocID(ctx, docID)
}

// DeleteByCollection 按集合名删除所有向量
func (s *VectorService) DeleteByCollection(ctx context.Context, collection string) error {
	return s.db.Vector().DeleteCollection(ctx, collection)
}

// GetDB 返回底层 cortexdb.DB，供需要使用高层 API（SaveKnowledge/SaveMemory 等）的服务调用
func (s *VectorService) GetDB() *cortexdb.DB {
	return s.db
}

// GetByCollection 获取指定集合中的所有向量（用于管理界面）
func (s *VectorService) GetByCollection(ctx context.Context, collection string, limit int) ([]core.ScoredEmbedding, error) {
	col, err := s.db.Vector().GetCollection(ctx, collection)
	if err != nil {
		// 集合不存在，返回空列表
		logger.WithModule("VectorService").Info("集合不存在或获取失败", "collection", collection, "error", err)
		return []core.ScoredEmbedding{}, nil
	}

	if col.Dimensions <= 0 {
		logger.WithModule("VectorService").Info("集合维度异常", "collection", collection, "dimensions", col.Dimensions)
		return []core.ScoredEmbedding{}, nil
	}

	zeroVector := make([]float32, col.Dimensions)
	opts := core.SearchOptions{
		Collection: collection,
		TopK:       limit,
	}

	results, err := s.db.Vector().Search(ctx, zeroVector, opts)
	if err != nil {
		logger.WithModule("VectorService").Error("搜索集合失败", "collection", collection, "error", err)
		return []core.ScoredEmbedding{}, nil
	}

	return results, nil
}

// DeleteByFilter 按 metadata 过滤条件删除向量
func (s *VectorService) DeleteByFilter(ctx context.Context, collection string, filter map[string]string) (int, error) {
	if collection == "" {
		return 0, fmt.Errorf("collection 名称不能为空")
	}

	col, err := s.db.Vector().GetCollection(ctx, collection)
	if err != nil {
		return 0, nil
	}

	zeroVector := make([]float32, col.Dimensions)
	opts := core.SearchOptions{
		Collection: collection,
		TopK:       10000,
		Filter:     filter,
	}

	results, err := s.db.Vector().Search(ctx, zeroVector, opts)
	if err != nil {
		return 0, fmt.Errorf("按过滤条件搜索失败: %w", err)
	}

	deletedCount := 0
	for _, result := range results {
		if err := s.db.Vector().DeleteByDocID(ctx, result.DocID); err != nil {
			logger.WithModule("VectorService").Error("删除向量失败", "docID", result.DocID, "error", err)
			continue
		}
		deletedCount++
	}

	return deletedCount, nil
}

// Close 关闭数据库
func (s *VectorService) Close() error {
	return s.db.Close()
}

// SearchResult 通用搜索结果结构
type SearchResult struct {
	Content    string            `json:"content"`
	Score      float64           `json:"score"`
	Metadata   map[string]string `json:"metadata"`
	Collection string            `json:"collection"`
	DocID      string            `json:"docId"`
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