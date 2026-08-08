package service

import (
	"context"
	"testing"

	"github.com/dshmyz/gracedb/pkg/gracedb"
	"github.com/dshmyz/gracedb/pkg/types"
)

// fakeEmbedder 用确定性伪向量模拟 embedding，让 gracedb 在不依赖外部 AI 的前提下
// 也能 SaveMemory/SearchMemory，从而端到端验证「记忆反射落库 → 枚举 → 聚合图谱」链路。
type fakeEmbedder struct{}

func (fakeEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	return fakeVec(text), nil
}

func (fakeEmbedder) EmbedBatch(_ context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, len(texts))
	for i, t := range texts {
		out[i] = fakeVec(t)
	}
	return out, nil
}

func (fakeEmbedder) Dimension() int { return 8 }

// fakeVec 根据文本内容生成确定性伪向量（同内容同向量，便于搜索命中）。
func fakeVec(text string) []float32 {
	v := make([]float32, 8)
	for i, r := range text {
		v[int(r)%8] += float32(i+1) * 0.1
	}
	return v
}

// TestMemoryGraphChain_SeededLikeReflection 端到端验证「记忆来源」知识图谱链路：
//  1. 用与 ConsolidateMessage 完全相同的 SaveMemory（带 knowledge_memory_entities/themes 数组
//     metadata）写入若干条记忆；
//  2. GetUserMemories 把这些 []interface{} 元数据读回为 []string；
//  3. buildMemoryGraphFromRecords 聚合出非空 nodes/edges/memories（含 terms）。
//
// 用临时 gracedb + fakeEmbedder，不依赖外部 AI / 不碰线上库；这模拟记住了「谁去哪里旅游、
// 谁喜欢喝咖啡聊工作」等值得画图的实体/主题记忆。
func TestMemoryGraphChain_SeededLikeReflection(t *testing.T) {
	db, err := gracedb.Open(t.TempDir()+"/gracedb", gracedb.WithEmbedder(fakeEmbedder{}))
	if err != nil {
		t.Fatalf("打开临时 gracedb 失败: %v", err)
	}
	defer db.Close()

	svc := &AvatarMemoryService{db: db, aiService: nil}

	seeds := []types.MemorySaveRequest{
		{
			MemoryID: "memory_3_1", UserID: "3", Scope: "user", Namespace: "avatar",
			Content: "小明决定周三开项目例会", Importance: 0.8,
			Metadata: map[string]any{
				"knowledge_memory_themes":   []string{"项目", "例会"},
				"knowledge_memory_entities": []string{"小明"},
			},
		},
		{
			MemoryID: "memory_3_2", UserID: "3", Scope: "user", Namespace: "avatar",
			Content: "小明喜欢喝咖啡，常和同事聊工作", Importance: 0.6,
			Metadata: map[string]any{
				"knowledge_memory_themes":   []string{"咖啡", "工作"},
				"knowledge_memory_entities": []string{"小明"},
			},
		},
		{
			MemoryID: "memory_3_3", UserID: "3", Scope: "user", Namespace: "avatar",
			Content: "团队周五下午开例会", Importance: 0.7,
			Metadata: map[string]any{
				"knowledge_memory_themes":   []string{"例会"},
				"knowledge_memory_entities": []string{"团队"},
			},
		},
	}
	for i := range seeds {
		if _, err := db.SaveMemory(seeds[i]); err != nil {
			t.Fatalf("seed #%d SaveMemory 失败: %v", i, err)
		}
	}

	// 关键：走生产枚举路径，验证 []interface{} 元数据正确回读为实体/主题
	records, err := svc.GetUserMemories(3, 100)
	if err != nil {
		t.Fatalf("GetUserMemories 失败: %v", err)
	}
	if len(records) != 3 {
		t.Fatalf("records = %d, want 3", len(records))
	}

	graph := buildMemoryGraphFromRecords(records)

	// 节点：小明(2·实体), 团队(1·实体), 项目(1·主题), 例会(2·主题), 咖啡(1·主题), 工作(1·主题)
	if len(graph.Nodes) != 6 {
		t.Fatalf("nodes = %d, want 6 (entities/themes read back from metadata)", len(graph.Nodes))
	}
	countByName := map[string]int{}
	typeByName := map[string]string{}
	for _, n := range graph.Nodes {
		countByName[n.Name] = n.Count
		typeByName[n.Name] = n.Type
	}
	if countByName["小明"] != 2 || typeByName["小明"] != "entity" {
		t.Errorf("小明 count/type = %d/%s, want 2/entity", countByName["小明"], typeByName["小明"])
	}
	if countByName["例会"] != 2 || typeByName["例会"] != "theme" {
		t.Errorf("例会 count/type = %d/%s, want 2/theme", countByName["例会"], typeByName["例会"])
	}

	// 边：有共现对（小明-例会 等），weight >= 1
	if len(graph.Edges) == 0 {
		t.Error("存在同一条记忆内的多名词共现，edges 不应为空")
	}
	for _, e := range graph.Edges {
		if e.Weight < 1 {
			t.Errorf("edge %s-%s weight %d < 1", e.Source, e.Target, e.Weight)
		}
	}

	// memories 应带 Terms，供前端点节点回查
	for _, m := range graph.Memories {
		if len(m.Terms) == 0 {
			t.Errorf("memory %s 应有 terms（实体/主题回读不丢失）", m.ID)
		}
	}

	// 端到端：聚合出的 graph 应真实非空（前端会据此画实体网）
	if len(graph.Nodes) == 0 || len(graph.Memories) == 0 {
		t.Error("graph 应为非空（这是「记忆」来源能画出实体网的判定）")
	}
}

// fakeVec 的确定性需保证同文本同向量，否则 SearchMemory 顶多检索不中。GetUserMemories
// 走空查询的内存桶列表路径（不依赖语义召回排序），此处确认该枚举可被后续聚合，独立于排序。
func TestFakeVec_Deterministic(t *testing.T) {
	a, b, c := fakeVec("小明"), fakeVec("小明"), fakeVec("团队")
	if len(a) != 8 || len(b) != 8 || len(c) != 8 {
		t.Fatalf("fakeVec 应有 8 维，got %d/%d/%d", len(a), len(b), len(c))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("同文本向量应一致，dim %d: %v vs %v", i, a[i], b[i])
		}
	}
	diff := false
	for i := range a {
		if a[i] != c[i] {
			diff = true
		}
	}
	if !diff {
		t.Error("不同文本向量应不同")
	}
}
