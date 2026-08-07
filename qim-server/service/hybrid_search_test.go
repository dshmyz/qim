package service

import (
	"testing"

	"github.com/dshmyz/gracedb/pkg/types"
)

func sc(i int, score float32) types.ScoredEmbedding {
	return types.ScoredEmbedding{
		Embedding: types.Embedding{
			ID:      string(rune('a' + i)),
			DocID:   string(rune('a' + i)),
			Content: "content",
			Metadata: map[string]string{
				"title": "doc",
			},
		},
		Score: score,
	}
}

func TestMergeRRF_ReturnsBothSourcesTopK(t *testing.T) {
	semantic := []types.ScoredEmbedding{sc(0, 0.9), sc(1, 0.8), sc(2, 0.7)}
	fts := []types.ScoredEmbedding{sc(3, 1.0), sc(4, 0.5), sc(2, 0.5)}

	merged := mergeRRF(semantic, fts, 4)
	if len(merged) != 4 {
		t.Fatalf("mergeRRF returned %d, want 4", len(merged))
	}
	// 第 2 个文档（ID 'c'）双路命中，RRF 分数最高，应排第一
	if merged[0].DocID != "c" {
		t.Errorf("双路命中文档应排第一，got %q", merged[0].DocID)
	}
}

func TestMergeRRF_CapsTopK(t *testing.T) {
	semantic := []types.ScoredEmbedding{sc(0, 0.9), sc(1, 0.8)}
	fts := []types.ScoredEmbedding{sc(2, 0.9)}
	merged := mergeRRF(semantic, fts, 2)
	if len(merged) != 2 {
		t.Fatalf("mergeRRF should cap at topK=2, got %d", len(merged))
	}
}

func TestHybridDisplayScores_PreservesSemanticKeepsFTSFallback(t *testing.T) {
	semantic := []types.ScoredEmbedding{sc(0, 0.85), sc(1, 0.6)}
	// merged 里 mixed：doc a（语义 0.85）、doc a2 仅 FTS、doc b（语义 0.6）
	merged := []types.ScoredEmbedding{sc(0, 0.0), {Embedding: types.Embedding{ID: "x", DocID: "x"}, Score: 0.0}, sc(1, 0.0)}

	out := hybridDisplayScores(merged, semantic)
	scores := map[string]float32{}
	for _, m := range out {
		scores[m.DocID] = m.Score
	}
	if scores["a"] != 0.85 {
		t.Errorf("语义命中 doc=a 应保留余弦分 0.85, got %v", scores["a"])
	}
	if scores["b"] != 0.6 {
		t.Errorf("语义命中 doc=b 应保留余弦分 0.6, got %v", scores["b"])
	}
	if scores["x"] != 0.5 {
		t.Errorf("仅 FTS 命中 doc=x 应回退到 0.5, got %v", scores["x"])
	}
}
