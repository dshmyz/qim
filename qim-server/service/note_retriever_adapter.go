package service

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/schema"
)

type NoteRetriever struct {
	noteSvc *NoteVectorService
	userID  uint
	topK    int
}

func NewNoteRetriever(noteSvc *NoteVectorService, userID uint, topK int) *NoteRetriever {
	return &NoteRetriever{
		noteSvc: noteSvc,
		userID:  userID,
		topK:    topK,
	}
}

func (r *NoteRetriever) Retrieve(ctx context.Context, query string, opts ...retriever.Option) ([]*schema.Document, error) {
	topK := r.topK
	opt := retriever.GetCommonOptions(&retriever.Options{}, opts...)
	if opt.TopK != nil {
		topK = *opt.TopK
	}

	results, err := r.noteSvc.SearchNotes(r.userID, query, topK)
	if err != nil {
		return nil, fmt.Errorf("搜索笔记失败: %w", err)
	}

	docs := make([]*schema.Document, len(results))
	for i, r := range results {
		meta := map[string]any{
			"score": r.Score,
			"type":  "note",
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
