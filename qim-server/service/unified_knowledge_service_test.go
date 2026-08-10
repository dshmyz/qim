package service

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBuildContextWithSources_SingleRetrieval
// BuildContextWithSources 应一次检索同时产出（注入提示词的上下文串, 命中的知识来源），
// 来源为最小展示结构（标题/相关度），不携带文档正文。
func TestBuildContextWithSources_SingleRetrieval(t *testing.T) {
	svc := NewUnifiedKnowledgeService(nil, &LegacyKnowledgeFallback{
		SearchFunc: func(_ string, _ uint, _ int) []KnowledgeSnippet {
			return []KnowledgeSnippet{
				{Title: "Q3 规划", Score: 0.92, Content: "敏感正文"},
				{Title: "", Score: 0.8, Content: "无标题正文"},
			}
		},
	})

	ctx, sources := svc.BuildContextWithSources("问题", 1, 3)
	assert.Contains(t, ctx, "Q3 规划", "上下文串应包含命中标题")
	require.Len(t, sources, 2)
	assert.Equal(t, "Q3 规划", sources[0].Title)
	assert.Equal(t, 0.92, sources[0].Score)
	assert.Equal(t, "未命名", sources[1].Title, "空标题回退为「未命名」")
	// 只暴露最小结构，不携带正文：序列化后仅含 title/score 两个键
	raw, err := json.Marshal(sources[0])
	require.NoError(t, err)
	assert.JSONEq(t, `{"title":"Q3 规划","score":0.92}`, string(raw))
}

// TestBuildContextWithSources_EmptyWhenNoHit
// 无命中时返回空串与 nil 来源，调用方据此不写 knowledge_sources，前端不展示徽章。
func TestBuildContextWithSources_EmptyWhenNoHit(t *testing.T) {
	svc := NewUnifiedKnowledgeService(nil, &LegacyKnowledgeFallback{
		SearchFunc: func(_ string, _ uint, _ int) []KnowledgeSnippet { return nil },
	})
	ctx, sources := svc.BuildContextWithSources("问题", 1, 3)
	assert.Empty(t, ctx)
	assert.Nil(t, sources)
}

// TestBuildContextWithSources_NilServiceSafe
// nil 接收者安全返回空，避免 prepareInput nil 崩。
func TestBuildContextWithSources_NilServiceSafe(t *testing.T) {
	var svc *UnifiedKnowledgeService
	ctx, sources := svc.BuildContextWithSources("问题", 1, 3)
	assert.Empty(t, ctx)
	assert.Nil(t, sources)
}

// TestBuildContextWithSources_DedupSameDocChunks
// 文档按 800 字分块入库，同一文档的多个块可能同时命中检索。
// 「知识来源」徽章应按标题去重，同文档只保留得分最高的一条；
// 但上下文串仍保留各块不同正文（供模型参考文档不同段落）。
func TestBuildContextWithSources_DedupSameDocChunks(t *testing.T) {
	svc := NewUnifiedKnowledgeService(nil, &LegacyKnowledgeFallback{
		SearchFunc: func(_ string, _ uint, _ int) []KnowledgeSnippet {
			return []KnowledgeSnippet{
				{Title: "产品需求文档", Score: 0.92, Content: "第一段内容"},
				{Title: "产品需求文档", Score: 0.85, Content: "第二段内容"},
				{Title: "产品需求文档", Score: 0.78, Content: "第三段内容"},
				{Title: "设计规范", Score: 0.80, Content: "规范内容"},
			}
		},
	})

	ctx, sources := svc.BuildContextWithSources("问题", 1, 3)

	// 同标题文档去重后只剩 2 条
	require.Len(t, sources, 2)
	assert.Equal(t, "产品需求文档", sources[0].Title)
	assert.Equal(t, 0.92, sources[0].Score, "同文档多块应保留最高分")
	assert.Equal(t, "设计规范", sources[1].Title)

	// 上下文串仍包含所有命中块的不同正文（供模型参考不同段落）
	assert.Contains(t, ctx, "第一段内容")
	assert.Contains(t, ctx, "第二段内容")
	assert.Contains(t, ctx, "第三段内容")
}
