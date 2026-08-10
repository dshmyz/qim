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
	assert.Contains(t, ctx, "相关度: 92.0%", "提示词相关度标签应为 score*100，与前端徽章一致")
	require.Len(t, sources, 2)
	assert.Equal(t, "Q3 规划", sources[0].Title)
	assert.Equal(t, 0.92, sources[0].Score)
	assert.Equal(t, "knowledge", sources[0].Source, "知识库命中应标记 source=knowledge")
	assert.Equal(t, "未命名", sources[1].Title, "空标题回退为「未命名」")
	// 只暴露最小结构，不携带正文：序列化后仅含 title/score/source 三个键
	raw, err := json.Marshal(sources[0])
	require.NoError(t, err)
	assert.JSONEq(t, `{"title":"Q3 规划","score":0.92,"source":"knowledge"}`, string(raw))
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
// 上下文串与「知识来源」徽章统一按标题去重：同文档只保留得分最高的那块
// 正文注入提示词（避免浪费 context window），徽章只展示一条。
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

	// 上下文串只保留每个文档得分最高的一块正文（不浪费 context window）
	assert.Contains(t, ctx, "第一段内容", "应保留最高分块（0.92）的正文")
	assert.Contains(t, ctx, "规范内容")
	assert.NotContains(t, ctx, "第二段内容", "同文档低分块不应注入提示词")
	assert.NotContains(t, ctx, "第三段内容", "同文档低分块不应注入提示词")
}

// TestMemoryResultsToSources
// 群记忆 Recall 结果应转为 KnowledgeSource，保留 score 并标记 source=memory。
func TestMemoryResultsToSources(t *testing.T) {
	results := []SearchResult{
		{Content: "项目截止日期 3 月 15 日", Score: 0.87, DocID: "mem_1", Metadata: map[string]string{"title": "项目排期"}},
		{Content: "张三负责后端", Score: 0.65, DocID: "mem_2"},
	}
	sources := memoryResultsToSources(results)
	require.Len(t, sources, 2)
	assert.Equal(t, "项目排期", sources[0].Title)
	assert.Equal(t, 0.87, sources[0].Score)
	assert.Equal(t, "memory", sources[0].Source)
	assert.Equal(t, "未命名", sources[1].Title, "无 title 时回退为未命名")
	assert.Equal(t, "memory", sources[1].Source)
}

// TestMemoryResultsToSources_Empty
// 空记忆结果应返回空切片（非 nil），前端据此不渲染额外条目。
func TestMemoryResultsToSources_Empty(t *testing.T) {
	sources := memoryResultsToSources(nil)
	assert.Empty(t, sources)
}
