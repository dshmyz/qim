package service

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/schema"
	"github.com/cloudwego/eino/components/retriever"
)

type GroupDocRetriever struct {
	groupDocSvc *GroupDocumentService
	groupID     uint
	topK        int
}

func NewGroupDocRetriever(groupDocSvc *GroupDocumentService, groupID uint, topK int) *GroupDocRetriever {
	return &GroupDocRetriever{
		groupDocSvc: groupDocSvc,
		groupID:     groupID,
		topK:        topK,
	}
}

func (r *GroupDocRetriever) Retrieve(ctx context.Context, query string, opts ...retriever.Option) ([]*schema.Document, error) {
	topK := r.topK
	opt := retriever.GetCommonOptions(&retriever.Options{}, opts...)
	if opt.TopK != nil {
		topK = *opt.TopK
	}

	results, err := r.groupDocSvc.SearchKnowledge(r.groupID, query, topK)
	if err != nil {
		return nil, fmt.Errorf("搜索群文档知识失败: %w", err)
	}

	docs := make([]*schema.Document, len(results))
	for i, r := range results {
		meta := map[string]any{
			"score": r.Score,
			"type":  "group_document",
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
