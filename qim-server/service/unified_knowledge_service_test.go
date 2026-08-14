package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBuildContextWithSources_SingleRetrieval
// BuildContextWithSources 应一次检索同时产出（注入提示词的上下文串, 命中的知识来源），
// 来源为最小展示结构（标题/相关度/命中摘要），携带 snippet 供前端悬停/点击查看正文摘要。
func TestBuildContextWithSources_SingleRetrieval(t *testing.T) {
	svc := NewUnifiedKnowledgeService(nil, &LegacyKnowledgeFallback{
		SearchFunc: func(_ string, _ uint, _ int) []KnowledgeSnippet {
			return []KnowledgeSnippet{
				{Title: "Q3 规划", Score: 0.92, Content: "敏感正文", DocID: "doc_123"},
				{Title: "", Score: 0.8, Content: "无标题正文"},
			}
		},
	}, nil, nil)

	ctx, sources := svc.BuildContextWithSources("问题", 1, 3)
	assert.Contains(t, ctx, "Q3 规划", "上下文串应包含命中标题")
	assert.Contains(t, ctx, "相关度: 92.0%", "提示词相关度标签应为 score*100，与前端徽章一致")
	require.Len(t, sources, 2)
	assert.Equal(t, "Q3 规划", sources[0].Title)
	assert.Equal(t, 0.92, sources[0].Score)
	assert.Equal(t, "knowledge", sources[0].Source, "知识库命中应标记 source=knowledge")
	assert.Equal(t, "doc_123", sources[0].ID, "应填充文档ID供前端点击跳转")
	assert.Equal(t, "未命名", sources[1].Title, "空标题回退为「未命名」")
	// B 方案：携带 snippet（命中正文摘要）供前端悬停/点击展示，不再主动去掉。
	raw, err := json.Marshal(sources[0])
	require.NoError(t, err)
	assert.JSONEq(t, `{"title":"Q3 规划","score":0.92,"source":"knowledge","id":"doc_123","snippet":"敏感正文"}`, string(raw))
}

// TestBuildContextWithSources_EmptyWhenNoHit
// 无命中时返回空串与 nil 来源，调用方据此不写 knowledge_sources，前端不展示徽章。
func TestBuildContextWithSources_EmptyWhenNoHit(t *testing.T) {
	svc := NewUnifiedKnowledgeService(nil, &LegacyKnowledgeFallback{
		SearchFunc: func(_ string, _ uint, _ int) []KnowledgeSnippet { return nil },
	}, nil, nil)
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
	}, nil, nil)

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
		{Content: "张三负责后端开发工作", Score: 0.75, DocID: "mem_2"},
	}
	sources := memoryResultsToSources(results, 0.6)
	require.Len(t, sources, 2)
	assert.Equal(t, "项目排期", sources[0].Title, "有 title 时用 title")
	assert.Equal(t, "memory", sources[0].Source)
	assert.Equal(t, "mem_1", sources[0].ID)
	assert.Equal(t, "张三负责后端开发工作", sources[1].Title, "无 title 时用 content 作为标题")
	assert.Equal(t, "memory", sources[1].Source)
	assert.Equal(t, "mem_2", sources[1].ID)
}

// TestMemoryResultsToSources_Empty
// 空记忆结果应返回空切片（非 nil），前端据此不渲染额外条目。
func TestMemoryResultsToSources_Empty(t *testing.T) {
	sources := memoryResultsToSources(nil, 0.6)
	assert.Empty(t, sources)
}

// TestMemoryResultsToSources_SortDescAndHardFloor
// 记忆来源按分数降序输出；门槛为硬下限（拦纯噪音），低分但高于下限的真实命中保留，
// 与群助手 selectTopKnowledge 语义一致。
func TestMemoryResultsToSources_SortDescAndHardFloor(t *testing.T) {
	results := []SearchResult{
		{Content: "低分但真实（0.35）", Score: 0.35, DocID: "m3"},
		{Content: "最高分（0.9）", Score: 0.9, DocID: "m1"},
		{Content: "中分（0.5）", Score: 0.5, DocID: "m2"},
		{Content: "纯噪音（0.1）", Score: 0.1, DocID: "m4"},
	}
	// 硬下限 0.3：0.35/0.5/0.9 保留（旧绝对门槛 0.6 会误杀 0.35），0.1 被拦
	sources := memoryResultsToSources(results, 0.3)
	require.Len(t, sources, 3)
	assert.Equal(t, 0.9, sources[0].Score, "降序：最高分在前")
	assert.Equal(t, 0.5, sources[1].Score)
	assert.Equal(t, 0.35, sources[2].Score, "高于硬下限的真实低分记忆保留")
	assert.NotContains(t, sourceIDs(sources), "m4", "纯噪音被硬下限拦住")
}

func sourceIDs(sources []KnowledgeSource) []string {
	out := make([]string, 0, len(sources))
	for _, s := range sources {
		out = append(out, s.ID)
	}
	return out
}

// TestClipSnippet
// clipSnippet 超过 maxSnippetLen 截断并加省略号，空串原样返回不改语义。
func TestClipSnippet(t *testing.T) {
	assert.Equal(t, "", clipSnippet(""), "空串原样返回")
	assert.Equal(t, "短摘要", clipSnippet("短摘要"), "未超长时原样返回")
	// 构造刚好超长的字符串（121 runes）
	long := strings.Repeat("字", maxSnippetLen+1)
	got := clipSnippet(long)
	assert.Equal(t, maxSnippetLen+1, len([]rune(got)), "截断后长度为 maxSnippetLen 字符 + 省略号")
	assert.True(t, strings.HasSuffix(got, "…"), "超长应以省略号结尾")
	// 刚好等于 maxSnippetLen 的字符串不截断
	exact := strings.Repeat("字", maxSnippetLen)
	assert.Equal(t, exact, clipSnippet(exact), "刚好等于 maxSnippetLen 时不截断")
}

// TestBuildContextWithSources_HardFloor
// 硬下限（默认 0.3）只拦纯噪音：低于下限的召回不注入也不进徽章；
// 高于下限的（含 0.55 这类此前被绝对门槛 0.6 误杀的唯一命中）按分数降序保留进 top-N。
func TestBuildContextWithSources_HardFloor(t *testing.T) {
	svc := NewUnifiedKnowledgeService(nil, &LegacyKnowledgeFallback{
		SearchFunc: func(_ string, _ uint, _ int) []KnowledgeSnippet {
			return []KnowledgeSnippet{
				{Title: "相关文档", Score: 0.85, Content: "高分内容", DocID: "doc_1"},
				{Title: "唯一命中文档", Score: 0.55, Content: "低分但唯一命中内容", DocID: "doc_2"}, // 旧版会被 0.6 门槛误杀
				{Title: "纯噪音文档", Score: 0.2, Content: "低于硬下限", DocID: "doc_3"},       // < 0.3 硬下限
			}
		},
	}, nil, nil)

	ctx, sources := svc.BuildContextWithSources("问题", 1, 5)

	// 高分优先排序；0.55 高于硬下限因此被保留；0.2 低于硬下限被丢弃
	require.Len(t, sources, 2)
	assert.Equal(t, "相关文档", sources[0].Title)
	assert.Equal(t, "唯一命中文档", sources[1].Title)

	assert.Contains(t, ctx, "高分内容")
	assert.Contains(t, ctx, "低分但唯一命中内容")
	assert.NotContains(t, ctx, "低于硬下限")
}

// TestSelectTopKnowledge
// selectTopKnowledge 纯函数：硬下限去噪 → 同标题去重（保留最高分块）→ 分数降序 → 截取前 limit 条。
func TestSelectTopKnowledge(t *testing.T) {
	svc := &UnifiedKnowledgeService{}
	t.Run("去重取最高分并排序截取", func(t *testing.T) {
		picked := svc.selectTopKnowledge(0.3, 2, []KnowledgeSnippet{
			{Title: "A", Score: 0.4, Content: "低"},
			{Title: "B", Score: 0.8, Content: "高"},
			{Title: "A", Score: 0.7, Content: "同文档更高分块"},
			{Title: "C", Score: 0.9, Content: "最高"},
		})
		// 同标题 A 只保留 0.7 那块；排序后 C(0.9) > B(0.8) > A(0.7)；取前 2 → C, B
		require.Len(t, picked, 2)
		assert.Equal(t, "C", picked[0].Title)
		assert.Equal(t, "B", picked[1].Title)
	})

	t.Run("低于硬下限被丢弃", func(t *testing.T) {
		picked := svc.selectTopKnowledge(0.3, 5, []KnowledgeSnippet{
			{Title: "A", Score: 0.2, Content: "噪音"},
			{Title: "B", Score: 0.6, Content: "相关"},
		})
		require.Len(t, picked, 1)
		assert.Equal(t, "B", picked[0].Title)
	})

	t.Run("limit 大于数量时全收", func(t *testing.T) {
		picked := svc.selectTopKnowledge(0.0, 10, []KnowledgeSnippet{
			{Title: "A", Score: 0.1, Content: "a"},
			{Title: "B", Score: 0.2, Content: "b"},
		})
		require.Len(t, picked, 2)
	})

	// 回归：按 DocID 去重，不同文档同标题应都保留
	t.Run("不同文档同标题应保留两个", func(t *testing.T) {
		picked := svc.selectTopKnowledge(0.3, 10, []KnowledgeSnippet{
			{DocID: "doc_1", Title: "背景", Score: 0.8, Content: "文档1内容"},
			{DocID: "doc_2", Title: "背景", Score: 0.7, Content: "文档2内容"},
		})
		require.Len(t, picked, 2, "两个不同 DocID 的同标题文档都应保留")
		assert.Equal(t, "doc_1", picked[0].DocID)
		assert.Equal(t, "doc_2", picked[1].DocID)
	})

	// DocID 为空时回退标题去重（同标题同空 DocID 视为同一文档多块）
	t.Run("DocID 为空时标题去重仍生效", func(t *testing.T) {
		picked := svc.selectTopKnowledge(0.3, 10, []KnowledgeSnippet{
			{DocID: "", Title: "A", Score: 0.5, Content: "a"},
			{DocID: "", Title: "A", Score: 0.9, Content: "A更高分"},
		})
		require.Len(t, picked, 1, "同标题且无 DocID 应去重保留最高分")
		assert.Equal(t, "A更高分", picked[0].Content)
	})
}

// mockReranker 测试用相关性判定器，按标题可编程返回三态判定。
type mockReranker struct {
	// verdict 标题 → 注入判定；未命中默认"拿不准"（confident=false，保留）
	verdict map[string]mockVerdict
	calls   int
}

type mockVerdict struct {
	relevant  bool
	confident bool
	err       error
}

func (m *mockReranker) Relevant(_ context.Context, _ string, snip KnowledgeSnippet) (bool, bool, error) {
	m.calls++
	if v, ok := m.verdict[snip.Title]; ok {
		return v.relevant, v.confident, v.err
	}
	// 默认"拿不准"：调用方应保留
	return false, false, nil
}

// TestBuildContext_OnlyConfidentIrrelevantFiltered
// 只有"明确不相关"（relevant=false && confident=true）才被过滤；
// "拿不准"（confident=false）与"相关"均保留，避免误杀。
func TestBuildContext_OnlyConfidentIrrelevantFiltered(t *testing.T) {
	svc := NewUnifiedKnowledgeService(nil, &LegacyKnowledgeFallback{
		SearchFunc: func(_ string, _ uint, _ int) []KnowledgeSnippet {
			return []KnowledgeSnippet{
				{Title: "相关文档", Score: 0.85, Content: "相关内容", DocID: "d0"},
				{Title: "马克思主义论文", Score: 0.80, Content: "不相关内容", DocID: "d1"},
				{Title: "拿不准文档", Score: 0.75, Content: "模糊内容", DocID: "d2"},
			}
		},
	}, nil, nil)
	svc.SetReranker(&mockReranker{verdict: map[string]mockVerdict{
		"相关文档":    {relevant: true, confident: true},
		"马克思主义论文": {relevant: false, confident: true}, // 明确不相关
		// 拿不准文档未配置 → 默认自信=false，应保留
	}})

	ctx, sources := svc.BuildContextWithSources("用户问题", 1, 3)

	// 相关文档保留了，马克思主义论文被过滤，拿不准文档保留
	require.Len(t, sources, 2)
	assert.Equal(t, "相关文档", sources[0].Title)
	assert.Equal(t, "拿不准文档", sources[1].Title)
	assert.Contains(t, ctx, "相关内容")
	assert.Contains(t, ctx, "模糊内容")
	assert.NotContains(t, ctx, "不相关内容", "明确不相关的内容不应注入 prompt")
}

// TestBuildContext_RerankerErrorKeepsAll
// 判定器返回错误时应全部保留（宁可留不可错杀），不阻塞检索流程。
func TestBuildContext_RerankerErrorKeepsAll(t *testing.T) {
	svc := NewUnifiedKnowledgeService(nil, &LegacyKnowledgeFallback{
		SearchFunc: func(_ string, _ uint, _ int) []KnowledgeSnippet {
			return []KnowledgeSnippet{
				{Title: "文档A", Score: 0.85, Content: "内容A", DocID: "a"},
				{Title: "文档B", Score: 0.80, Content: "内容B", DocID: "b"},
			}
		},
	}, nil, nil)
	svc.SetReranker(&mockReranker{verdict: map[string]mockVerdict{
		"文档A": {relevant: false, confident: true, err: fmt.Errorf("网络超时")},
	}})

	ctx, sources := svc.BuildContextWithSources("问题", 1, 3)

	// 出错那条也保留
	require.Len(t, sources, 2)
	assert.Contains(t, ctx, "内容A")
	assert.Contains(t, ctx, "内容B")
}

// TestBuildContext_NilRerankerSkipsJudgment
// 未注入判定器（纯装配路径）时不调用判定，全部保留。等价于启动时未启用 LLM。
func TestBuildContext_NilRerankerSkipsJudgment(t *testing.T) {
	svc := NewUnifiedKnowledgeService(nil, &LegacyKnowledgeFallback{
		SearchFunc: func(_ string, _ uint, _ int) []KnowledgeSnippet {
			return []KnowledgeSnippet{
				{Title: "任意文档", Score: 0.80, Content: "内容", DocID: "x"},
			}
		},
	}, nil, nil)
	// 不 SetReranker → nil，走纯装配

	ctx, sources := svc.BuildContextWithSources("问题", 1, 3)

	require.Len(t, sources, 1)
	assert.Contains(t, ctx, "内容")
}

// TestFilterByReranker_RegardingOrder
// filterByReranker 应保持原序稳定输出，且仅过滤明确不相关的条目。
func TestFilterByReranker_RegardingOrder(t *testing.T) {
	svc := NewUnifiedKnowledgeService(nil, &LegacyKnowledgeFallback{
		SearchFunc: func(_ string, _ uint, _ int) []KnowledgeSnippet {
			return []KnowledgeSnippet{
				{Title: "A", Score: 0.9, Content: "cA"},
				{Title: "B", Score: 0.8, Content: "cB"},
				{Title: "C", Score: 0.7, Content: "cC"},
			}
		},
	}, nil, nil)
	mr := &mockReranker{verdict: map[string]mockVerdict{
		"A": {relevant: true, confident: true},
		"B": {relevant: false, confident: true}, // 明确不相关，应过滤
		"C": {relevant: true, confident: true},
	}}
	svc.SetReranker(mr)

	// 直接调 filterByReranker，绕开阈值/去重，验证判定器行为与顺序
	snips := []KnowledgeSnippet{
		{Title: "A", Score: 0.9, Content: "cA"},
		{Title: "B", Score: 0.8, Content: "cB"},
		{Title: "C", Score: 0.7, Content: "cC"},
	}
	filtered := svc.filterByReranker("q", snips)

	require.Len(t, filtered, 2)
	assert.Equal(t, "A", filtered[0].Title)
	assert.Equal(t, "C", filtered[1].Title, "保持原序，B 被过滤")
	assert.Equal(t, 3, mr.calls, "逐条判定调用 3 次")
}

// TestFilterByReranker_AllConfidentIrrelevant
// 全部"明确不相关"时应过滤所有片段，返回空结果。
func TestFilterByReranker_AllConfidentIrrelevant(t *testing.T) {
	svc := NewUnifiedKnowledgeService(nil, &LegacyKnowledgeFallback{
		SearchFunc: func(_ string, _ uint, _ int) []KnowledgeSnippet {
			return []KnowledgeSnippet{
				{Title: "马克思主义论文", Score: 0.80, Content: "完全不相关"},
			}
		},
	}, nil, nil)
	svc.SetReranker(&mockReranker{verdict: map[string]mockVerdict{
		"马克思主义论文": {relevant: false, confident: true},
	}})

	ctx, sources := svc.BuildContextWithSources("SSH key 问题", 1, 3)

	assert.Empty(t, ctx)
	assert.Nil(t, sources)
}

// TestExtractBoolField
// extractBoolField 应正确解析各种格式的布尔字段，包括多行、带后缀文字。
func TestExtractBoolField(t *testing.T) {
	tests := []struct {
		name string
		in   string
		key  string
		want bool
	}{
		{"正常true", `{"relevant": true, "confident": true}`, "relevant", true},
		{"正常false", `{"relevant": false, "confident": true}`, "relevant", false},
		{"confident false", `{"relevant": true, "confident": false}`, "confident", false},
		{"带空格", `{"relevant" :  true}`, "relevant", true},
		{"markdown代码块", "```json\n{\"confident\": true}\n```", "confident", true},
		{"多行", "{\n  \"relevant\": false,\n  \"confident\": true\n}", "relevant", false},
		{"无该字段", `{"other": true}`, "relevant", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, extractBoolField(tt.in, tt.key))
		})
	}
}

// TestNewUnifiedKnowledgeService_DefaultLLMReranker
// 传入非 nil aiService 时应构建默认 LLM 判定器；传 nil 时应为 nil（纯装配路径）。
func TestNewUnifiedKnowledgeService_DefaultLLMReranker(t *testing.T) {
	withAI := NewUnifiedKnowledgeService(nil, nil, nil, &ai.AIService{})
	require.NotNil(t, withAI.reranker, "有 aiService 时应默认构造 LLM 判定器")

	withoutAI := NewUnifiedKnowledgeService(nil, nil, nil, nil)
	assert.Nil(t, withoutAI.reranker, "无 aiService 时判定器应为 nil")
}

// TestParseVerdict_RequiresBothKeys
// parseVerdict 只有在 relevant/used 与 confident 两个布尔都显式解析出时才 ok=true；
// 缺任一字段返回"拿不准"（ok=false），避免因字段缺失被当成 false 而误杀。
func TestParseVerdict_RequiresBothKeys(t *testing.T) {
	r := NewLLMReranker(nil)
	tests := []struct {
		name string
		in   string
		ok   bool
		want bool // relevant 期望值（仅 ok=true 时检查）
	}{
		{"两键俱在true", `{"relevant": true, "confident": true}`, true, true},
		{"两键俱在false", `{"used": false, "confident": true}`, true, false},
		{"confident无", `{"relevant": true}`, false, false},
		{"relevant缺", `{"confident": true}`, false, false},
		{"非JSON", "完全不是json", false, false},
		{"带围栏", "```json\n{\"relevant\": false, \"confident\": true}\n```", true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rel, confident, ok := r.parseVerdict(tt.in)
			assert.Equal(t, tt.ok, ok)
			if tt.ok {
				assert.Equal(t, tt.want, rel)
				assert.True(t, confident)
			} else {
				// 解析失败/缺字段 → ok=false，confident 按拿不准（false）
				assert.False(t, confident)
			}
		})
	}
}
