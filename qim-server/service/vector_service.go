package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dshmyz/qim/qim-server/pkg/logger"

	"github.com/dshmyz/gracedb/pkg/gracedb"
	"github.com/dshmyz/gracedb/pkg/types"
)

type VectorService struct {
	db *gracedb.DB
}

func NewVectorService(path string, embedder types.Embedder) (*VectorService, error) {
	// 确保数据目录存在
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("创建向量数据目录失败: %w", err)
	}

	var opts []gracedb.Option
	if embedder != nil {
		opts = append(opts, gracedb.WithEmbedder(embedder))
	}

	db, err := gracedb.Open(path, opts...)
	if err != nil {
		return nil, fmt.Errorf("打开向量数据库失败: %w", err)
	}

	logger.WithModule("VectorService").Info("向量数据库初始化完成", "path", path)
	return &VectorService{db: db}, nil
}

// ensureGracedbCollection 确保 gracedb 集合存在。
// gracedb 的 Upsert 不会自动创建集合（集合不存在时返回 "gracedb: not found"），
// 因此写入前必须确保集合已创建；已存在集合（ErrCollectionExists）视为成功。
func ensureGracedbCollection(db *gracedb.DB, name string) error {
	if name == "" {
		return fmt.Errorf("collection 名称不能为空")
	}
	_, err := db.CreateCollection(name)
	if err == nil {
		return nil
	}
	if errors.Is(err, types.ErrCollectionExists) {
		return nil
	}
	return err
}

// AddVector 添加向量到指定集合（确保集合存在后再 Upsert）
func (s *VectorService) AddVector(ctx context.Context, collection string, id string, embedding []float32, content string, metadata map[string]string) error {
	if err := ensureGracedbCollection(s.db, collection); err != nil {
		return err
	}
	_, err := s.db.Upsert(collection, id, embedding, content, metadata, nil)
	return err
}

// EnsureCollection 确保集合存在（gracedb 的 Upsert 不会自动建集合，这里显式创建）
func (s *VectorService) EnsureCollection(ctx context.Context, name string, dimensions int) error {
	return ensureGracedbCollection(s.db, name)
}

// Search 在集合中搜索相似向量
func (s *VectorService) Search(ctx context.Context, collection string, queryVector []float32, topK int) ([]types.ScoredEmbedding, error) {
	opts := types.SearchOptions{
		TopK: topK,
	}
	return s.db.Search(collection, queryVector, opts)
}

// DeleteByDocID 按文档 ID 删除向量
func (s *VectorService) DeleteByDocID(ctx context.Context, collection string, docID string) error {
	return s.db.DeleteByDocID(collection, docID)
}

// DeleteByCollection 按集合名删除所有向量
func (s *VectorService) DeleteByCollection(ctx context.Context, collection string) error {
	return s.db.DeleteCollection(collection)
}

// GetDB 返回底层 gracedb.DB，供需要使用高层 API（SaveMemory/SearchMemory 等）的服务调用
func (s *VectorService) GetDB() *gracedb.DB {
	return s.db
}

// GetByCollection 获取指定集合中的所有向量（用于管理界面）
func (s *VectorService) GetByCollection(ctx context.Context, collection string, limit int) ([]types.ScoredEmbedding, error) {
	if _, err := s.db.GetCollection(collection); err != nil {
		// 集合不存在，返回空列表
		logger.WithModule("VectorService").Info("集合不存在或获取失败", "collection", collection, "error", err)
		return []types.ScoredEmbedding{}, nil
	}

	ids, err := s.db.ListEmbeddingIDs(collection)
	if err != nil {
		logger.WithModule("VectorService").Error("列出集合向量失败", "collection", collection, "error", err)
		return []types.ScoredEmbedding{}, nil
	}

	results := make([]types.ScoredEmbedding, 0, len(ids))
	for _, id := range ids {
		if limit > 0 && len(results) >= limit {
			break
		}
		emb, err := s.db.GetEmbedding(collection, id, true)
		if err != nil {
			logger.WithModule("VectorService").Error("读取向量失败", "collection", collection, "embID", id, "error", err)
			continue
		}
		if emb == nil {
			continue
		}
		results = append(results, types.ScoredEmbedding{
			Embedding: types.Embedding{
				ID:         emb.ID,
				Collection: emb.Collection,
				Content:    emb.Content,
				DocID:      emb.DocID,
				Metadata:   emb.Metadata,
			},
		})
	}

	return results, nil
}

// DeleteByFilter 按 metadata 过滤条件删除向量（枚举集合内向量，逐条匹配后删除）
func (s *VectorService) DeleteByFilter(ctx context.Context, collection string, filter map[string]string) (int, error) {
	if collection == "" {
		return 0, fmt.Errorf("collection 名称不能为空")
	}

	if _, err := s.db.GetCollection(collection); err != nil {
		return 0, nil
	}

	ids, err := s.db.ListEmbeddingIDs(collection)
	if err != nil {
		return 0, fmt.Errorf("列出集合向量失败: %w", err)
	}

	// 收集命中的 embedding ID（ListEmbeddingIDs 返回的是随机 UUID，而非 DocID）。
	// 删除必须按 embedding ID 走 DeleteEmbeddingBatch —— 若误把 UUID 当 docID 传给
	// DeleteByDocID，会因 DocID 恒不相等而静默删不掉（DeleteByDocID 只匹配 emb.DocID）。
	toDelete := make([]string, 0, len(ids))
	for _, id := range ids {
		emb, err := s.db.GetEmbedding(collection, id, true)
		if err != nil || emb == nil {
			continue
		}
		if matchesFilter(emb.Metadata, filter) {
			toDelete = append(toDelete, id)
		}
	}

	if len(toDelete) == 0 {
		return 0, nil
	}
	if err := s.db.DeleteEmbeddingBatch(collection, toDelete); err != nil {
		return 0, fmt.Errorf("批量删除向量失败: %w", err)
	}
	return len(toDelete), nil
}

// matchesFilter 判断 metadata 是否满足全部过滤条件（key=value 精确匹配）
func matchesFilter(metadata map[string]string, filter map[string]string) bool {
	for k, v := range filter {
		if metadata[k] != v {
			return false
		}
	}
	return true
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

// ScoredEmbeddingToSearchResult 将 gracedb 结果转换为通用结构
func ScoredEmbeddingToSearchResult(se types.ScoredEmbedding) SearchResult {
	return SearchResult{
		Content:    se.Content,
		Score:      float64(se.Score),
		Metadata:   se.Metadata,
		Collection: se.Collection,
		DocID:      se.DocID,
	}
}
