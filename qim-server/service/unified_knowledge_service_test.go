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
