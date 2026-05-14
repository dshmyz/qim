package service

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/schema"
	"github.com/cloudwego/eino/components/retriever"
)

type MemoryRetriever struct {
	memorySvc *AvatarMemoryService
	userID    uint
	topK      int
}

func NewMemoryRetriever(memorySvc *AvatarMemoryService, userID uint, topK int) *MemoryRetriever {
	return &MemoryRetriever{
		memorySvc: memorySvc,
		userID:    userID,
		topK:      topK,
	}
}

func (r *MemoryRetriever) Retrieve(ctx context.Context, query string, opts ...retriever.Option) ([]*schema.Document, error) {
	topK := r.topK
	opt := retriever.GetCommonOptions(&retriever.Options{}, opts...)
	if opt.TopK != nil {
		topK = *opt.TopK
	}

	results, err := r.memorySvc.Recall(r.userID, query, topK)
	if err != nil {
		return nil, fmt.Errorf("检索记忆失败: %w", err)
	}

	docs := make([]*schema.Document, len(results))
	for i, r := range results {
		meta := map[string]any{
			"score": r.Score,
			"type":  "memory",
		}
		for k, v := range r.Metadata {
			meta[k] = v
		}
		docs[i] = &schema.Document{
			ID:       r.DocID,
			Content:  r.Content,
			MetaData: meta,
		}
	}
	return docs, nil
}
