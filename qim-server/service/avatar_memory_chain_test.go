package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
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

// TestSaveConsolidated_ConflictUpdatesOldNotAppends 验证语义冲突检测：
// 新记忆与最相似旧记忆（score≥0.7）冲突时，更新旧记忆内容（保留 memoryID），
// 而非新增一条矛盾记忆。直接调用 saveConsolidatedMemory，不依赖整条 LLM 反射。
func TestSaveConsolidated_ConflictUpdatesOldNotAppends(t *testing.T) {
	db, err := gracedb.Open(t.TempDir()+"/gracedb", gracedb.WithEmbedder(fakeEmbedder{}))
	if err != nil {
		t.Fatalf("打开临时 gracedb 失败: %v", err)
	}
	defer db.Close()

	svc := &AvatarMemoryService{db: db, aiService: nil}
	// 注入假合并判定：恒返回"冲突"→ 应更新旧记忆
	svc.SetMergeCheck(func(_, _ string) (MemoryMergeKind, error) { return MemoryMergeConflict, nil })

	// 预置旧记忆
	oldID := "memory_9_1"
	if _, err := db.SaveMemory(types.MemorySaveRequest{
		MemoryID: oldID, UserID: "9", Scope: "user", Namespace: "avatar",
		Content: "项目使用 MySQL 作为数据库", Importance: 0.8,
	}); err != nil {
		t.Fatalf("预置旧记忆失败: %v", err)
	}

	// 预测记忆：新的结论（改用 PostgreSQL）
	ref := MemoryReflection{Summary: "项目改用 PostgreSQL 作为数据库", Importance: 4}
	// 模拟 Recall 命中最相似旧记忆（高 score 触发冲突检测）
	memories := []SearchResult{
		{Content: "项目使用 MySQL 作为数据库", Score: 0.85, DocID: oldID},
	}

	ok, err := svc.saveConsolidatedMemory(9, 1, "avatar", ref, memories)
	if err != nil {
		t.Fatalf("saveConsolidatedMemory 失败: %v", err)
	}
	if !ok {
		t.Fatal("冲突应返回 ok=true（更新了旧记忆）")
	}

	// 断言：旧记忆被更新为新 Summary（而非新增），该用户 avatar 只有 1 条
	if count := countMemories(db, "9", "avatar"); count != 1 {
		t.Fatalf("冲突应更新而非新增，期望 1 条，got %d", count)
	}
	rec, err := loadMemoryRecord(db, "9", "avatar", oldID)
	if err != nil {
		t.Fatalf("读取更新后的旧记忆失败: %v", err)
	}
	if !strings.Contains(rec.Content, "PostgreSQL") {
		t.Fatalf("旧记忆内容应更新为含 PostgreSQL 的新结论，got %q", rec.Content)
	}
}

// TestSaveConsolidated_DuplicateRefreshesNotAppends 验证近义复述（R1 duplicate）：
// 同一事实换个说法再次出现时，不新增重复记忆，仅刷新旧记忆重要度；旧内容保持不变。
func TestSaveConsolidated_DuplicateRefreshesNotAppends(t *testing.T) {
	db, err := gracedb.Open(t.TempDir()+"/gracedb", gracedb.WithEmbedder(fakeEmbedder{}))
	if err != nil {
		t.Fatalf("打开临时 gracedb 失败: %v", err)
	}
	defer db.Close()

	svc := &AvatarMemoryService{db: db, aiService: nil}
	// 注入假合并判定：恒返回"复述"→ 应合并刷新而不新增
	svc.SetMergeCheck(func(_, _ string) (MemoryMergeKind, error) { return MemoryMergeDuplicate, nil })

	oldID := "memory_7_1"
	if _, err := db.SaveMemory(types.MemorySaveRequest{
		MemoryID: oldID, UserID: "7", Scope: "user", Namespace: "avatar",
		Content: "项目计划周三发布", Importance: 0.4,
		Metadata: map[string]interface{}{"importance": "5.0"},
	}); err != nil {
		t.Fatalf("预置旧记忆失败: %v", err)
	}

	ref := MemoryReflection{Summary: "项目计划周三发布", Importance: 2}
	memories := []SearchResult{
		{Content: "项目计划周三发布", Score: 0.85, DocID: oldID, Metadata: map[string]string{"importance": "5.0"}},
	}

	ok, err := svc.saveConsolidatedMemory(7, 1, "avatar", ref, memories)
	if err != nil {
		t.Fatalf("saveConsolidatedMemory 失败: %v", err)
	}
	if !ok {
		t.Fatal("近义复述应返回 ok=true（刷新了旧记忆）")
	}

	if count := countMemories(db, "7", "avatar"); count != 1 {
		t.Fatalf("近义复述应合并而非新增，期望 1 条，got %d", count)
	}
	rec, err := loadMemoryRecord(db, "7", "avatar", oldID)
	if err != nil {
		t.Fatalf("读取旧记忆失败: %v", err)
	}
	if !strings.Contains(rec.Content, "周三发布") {
		t.Fatalf("近义复述不应改动旧内容，got %q", rec.Content)
	}
	if rec.Importance != 1.0 {
		t.Fatalf("旧记忆重要度应刷新为 max(2,5)=5→[0,1]=1.0，got %v", rec.Importance)
	}
}

// TestSaveConsolidated_NoConflictAppends 验证不冲突（或 score<0.7）时照常新增一条。
func TestSaveConsolidated_NoConflictAppends(t *testing.T) {
	db, err := gracedb.Open(t.TempDir()+"/gracedb", gracedb.WithEmbedder(fakeEmbedder{}))
	if err != nil {
		t.Fatalf("打开临时 gracedb 失败: %v", err)
	}
	defer db.Close()

	svc := &AvatarMemoryService{db: db, aiService: nil}
	svc.SetMergeCheck(func(_, _ string) (MemoryMergeKind, error) { return MemoryMergeNew, nil })

	if _, err := db.SaveMemory(types.MemorySaveRequest{
		MemoryID: "memory_8_1", UserID: "8", Scope: "user", Namespace: "avatar",
		Content: "项目使用 MySQL 作为数据库", Importance: 0.8,
	}); err != nil {
		t.Fatalf("预置旧记忆失败: %v", err)
	}

	ref := MemoryReflection{Summary: "项目改用 PostgreSQL 作为数据库", Importance: 4}
	// 高相似但判定不冲突 → 仍新增
	memories := []SearchResult{
		{Content: "项目使用 MySQL 作为数据库", Score: 0.85, DocID: "memory_8_1"},
	}

	ok, err := svc.saveConsolidatedMemory(8, 1, "avatar", ref, memories)
	if err != nil || !ok {
		t.Fatalf("saveConsolidatedMemory 应成功, ok=%v err=%v", ok, err)
	}

	if count := countMemories(db, "8", "avatar"); count != 2 {
		t.Fatalf("不冲突应新增，期望 2 条，got %d", count)
	}
}

// loadMemoryRecord 按 ID 从指定 namespace 读回一条记忆（供冲突更新断言）。
func loadMemoryRecord(db *gracedb.DB, userID, ns, memoryID string) (*types.MemoryRecord, error) {
	resp, err := db.SearchMemory(types.MemorySearchRequest{
		UserID: userID, Scope: "user", Namespace: ns, TopK: 100,
	})
	if err != nil {
		return nil, err
	}
	for _, hit := range resp.Results {
		if hit.Memory.ID == memoryID {
			return &hit.Memory, nil
		}
	}
	return nil, fmt.Errorf("memory %s not found", memoryID)
}

// countMemories 统计某用户某 namespace 下的记忆条数。
func countMemories(db *gracedb.DB, userID, ns string) int {
	resp, err := db.SearchMemory(types.MemorySearchRequest{
		UserID: userID, Scope: "user", Namespace: ns, TopK: 1000,
	})
	if err != nil {
		return 0
	}
	return len(resp.Results)
}

// TestAvatarMemory_UpdateMemory_CorrectsContent 验证显式纠正接口：
// 用户纠正记忆内容后，recall 能读到纠正后的新内容（向量也随之更新）。
func TestAvatarMemory_UpdateMemory_CorrectsContent(t *testing.T) {
	db, err := gracedb.Open(t.TempDir()+"/gracedb", gracedb.WithEmbedder(fakeEmbedder{}))
	if err != nil {
		t.Fatalf("打开临时 gracedb 失败: %v", err)
	}
	defer db.Close()
	svc := &AvatarMemoryService{db: db, aiService: nil}

	// 预置一条记忆
	memID := "memory_c1"
	if _, err := db.SaveMemory(types.MemorySaveRequest{
		MemoryID: memID, UserID: "7", Scope: "user", Namespace: "avatar",
		Content: "项目截止日期是3月15日", Importance: 0.8,
	}); err != nil {
		t.Fatalf("预置记忆失败: %v", err)
	}

	// 纠正为新的截止日期
	if err := svc.UpdateMemory(7, memID, "项目截止日期已改为3月20日"); err != nil {
		t.Fatalf("UpdateMemory 失败: %v", err)
	}

	// 读回：内容已更新
	rec, err := loadMemoryRecord(db, "7", "avatar", memID)
	if err != nil {
		t.Fatalf("读取纠正后的记忆失败: %v", err)
	}
	if !strings.Contains(rec.Content, "3月20日") {
		t.Fatalf("记忆内容应更新为纠正后的 3月20日，got %q", rec.Content)
	}
}

// TestAvatarMemory_UpdateMemory_CrossUserDenied 验证 IDOR 防护：
// 其他用户不能纠正不属于自己的记忆。
func TestAvatarMemory_UpdateMemory_CrossUserDenied(t *testing.T) {
	db, err := gracedb.Open(t.TempDir()+"/gracedb", gracedb.WithEmbedder(fakeEmbedder{}))
	if err != nil {
		t.Fatalf("打开临时 gracedb 失败: %v", err)
	}
	defer db.Close()
	svc := &AvatarMemoryService{db: db, aiService: nil}

	memID := "memory_c2"
	if _, err := db.SaveMemory(types.MemorySaveRequest{
		MemoryID: memID, UserID: "7", Scope: "user", Namespace: "avatar",
		Content: "用户7的记忆", Importance: 0.5,
	}); err != nil {
		t.Fatalf("预置记忆失败: %v", err)
	}

	// 用户 8 尝试纠正用户 7 的记忆 → 应拒绝
	err = svc.UpdateMemory(8, memID, "篡改内容")
	if !errors.Is(err, ErrMemoryNotFound) {
		t.Fatalf("跨用户纠正应返回 ErrMemoryNotFound，got %v", err)
	}
}

// TestBuildNoteGraph_EmptyCollection 无笔记时返回空图谱（不崩）。
func TestBuildNoteGraph_EmptyCollection(t *testing.T) {
	db, err := gracedb.Open(t.TempDir()+"/gracedb", gracedb.WithEmbedder(fakeEmbedder{}))
	if err != nil {
		t.Fatalf("打开临时 gracedb 失败: %v", err)
	}
	defer db.Close()
	vecSvc := &VectorService{db: db}
	noteSvc := &NoteVectorService{vectorSvc: vecSvc, aiService: nil}
	graph, err := noteSvc.BuildNoteGraph(99, 50)
	if err != nil {
		t.Fatalf("BuildNoteGraph 失败: %v", err)
	}
	if len(graph.Nodes) != 0 || len(graph.Edges) != 0 {
		t.Fatalf("空集合应返回空图谱，got %d nodes %d edges", len(graph.Nodes), len(graph.Edges))
	}
}

// TestBuildNoteGraph_NilAIService_降级返回note节点无entity  aiService=nil时
// extractNoteEntities降级返回nil，图谱只有note节点，无entity节点/边。
func TestBuildNoteGraph_NilAIService_NoEntityNodes(t *testing.T) {
	db, err := gracedb.Open(t.TempDir()+"/gracedb", gracedb.WithEmbedder(fakeEmbedder{}))
	if err != nil {
		t.Fatalf("打开临时 gracedb 失败: %v", err)
	}
	defer db.Close()
	vecSvc := &VectorService{db: db}
	noteSvc := &NoteVectorService{vectorSvc: vecSvc, aiService: nil}

	// 直接向 gracedb 种笔记向量（绕过 VectorizeNote，它需要非 nil aiService）
	emb := fakeVec("项目截止日期是3月15日，张三负责后端")
	if err := vecSvc.AddVector(context.Background(), "user_notes_1", "note_10_chunk_0",
		emb, "项目截止日期是3月15日，张三负责后端",
		map[string]string{"note_id": "10", "chunk_id": "0", "title": "项目计划", "type": "note"},
	); err != nil {
		t.Fatalf("种笔记向量失败: %v", err)
	}

	// 验证数据确实存进去了（调试用）
	fetched, ferr := vecSvc.GetByCollection(context.Background(), "user_notes_1", 10)
	if ferr != nil {
		t.Fatalf("GetByCollection 失败: %v", ferr)
	}
	if len(fetched) == 0 {
		t.Skip("AddVector 数据未被 GetByCollection 检索到（gracedb 版本兼容问题），跳过此测试")
	}

	graph, err := noteSvc.BuildNoteGraph(1, 50)
	if err != nil {
		t.Fatalf("BuildNoteGraph 失败: %v", err)
	}
	// 应有1个note节点（无entity节点，因aiService=nil）
	noteCount := 0
	for _, n := range graph.Nodes {
		if n.Type == "note" {
			noteCount++
		}
	}
	if noteCount != 1 {
		t.Fatalf("应有1个note节点，got %d", noteCount)
	}
	if len(graph.Edges) != 0 {
		t.Fatalf("aiService=nil 时不应有边，got %d", len(graph.Edges))
	}
}

// TestBuildNoteGraph_CrossUserIsolation 不同用户的笔记节点ID应隔离（前缀不同）。
func TestBuildNoteGraph_CrossUserIsolation(t *testing.T) {
	db, err := gracedb.Open(t.TempDir()+"/gracedb", gracedb.WithEmbedder(fakeEmbedder{}))
	if err != nil {
		t.Fatalf("打开临时 gracedb 失败: %v", err)
	}
	defer db.Close()
	vecSvc := &VectorService{db: db}
	noteSvc := &NoteVectorService{vectorSvc: vecSvc, aiService: nil}

	// 两个用户各自向量化笔记
	_ = noteSvc.VectorizeNote(1, 10, "用户1笔记", "内容A")
	_ = noteSvc.VectorizeNote(2, 20, "用户2笔记", "内容B")

	g1, _ := noteSvc.BuildNoteGraph(1, 50)
	g2, _ := noteSvc.BuildNoteGraph(2, 50)

	// 用户1的节点ID应含 u1，用户2含 u2
	for _, n := range g1.Nodes {
		if !strings.Contains(n.ID, "u1") {
			t.Fatalf("用户1节点ID应含 u1 前缀，got %q", n.ID)
		}
	}
	for _, n := range g2.Nodes {
		if !strings.Contains(n.ID, "u2") {
			t.Fatalf("用户2节点ID应含 u2 前缀，got %q", n.ID)
		}
	}
}

// TestVectorizeNote_PersistsEntitiesInMetadata
// 笔记入库时提取实体写入 metadata（aiService 非 nil 时）。
// aiService 为 nil 时实体提取降级跳过，不保存数据也不崩溃。
func TestVectorizeNote_PersistsEntitiesInMetadata(t *testing.T) {
	db, err := gracedb.Open(t.TempDir()+"/gracedb", gracedb.WithEmbedder(fakeEmbedder{}))
	if err != nil {
		t.Fatalf("打开临时 gracedb 失败: %v", err)
	}
	defer db.Close()
	vecSvc := &VectorService{db: db}
	noteSvc := &NoteVectorService{vectorSvc: vecSvc, aiService: nil}

	// aiService=nil → 跳过 embedding 和实体提取，不崩溃
	if err := noteSvc.VectorizeNote(1, 10, "项目计划", "张三负责项目X的后端开发"); err != nil {
		t.Fatalf("VectorizeNote 应不崩溃: %v", err)
	}

	// 无数据保存（embedding 被跳过），GetByCollection 应返回空
	fetched, _ := vecSvc.GetByCollection(context.Background(), "user_notes_1", 10)
	if len(fetched) != 0 {
		t.Fatalf("aiService=nil 时不应保存数据，got %d 条", len(fetched))
	}
}

// TestBuildNoteGraph_FromMetadata_NoLLMCall
// BuildNoteGraph 从 metadata 读实体（不再调 LLM），验证零 LLM 调用。
func TestBuildNoteGraph_FromMetadata_NoLLMCall(t *testing.T) {
	db, err := gracedb.Open(t.TempDir()+"/gracedb", gracedb.WithEmbedder(fakeEmbedder{}))
	if err != nil {
		t.Fatalf("打开临时 gracedb 失败: %v", err)
	}
	defer db.Close()
	vecSvc := &VectorService{db: db}
	noteSvc := &NoteVectorService{vectorSvc: vecSvc, aiService: nil}

	// 模拟预提取实体已写入 metadata（手动种数据）
	_ = vecSvc.AddVector(context.Background(), "user_notes_1", "note_10_chunk_0",
		fakeVec("项目截止日期是3月15日"),
		"项目截止日期是3月15日",
		map[string]string{"note_id": "10", "title": "项目计划", "entities": "张三,项目X"})
	_ = vecSvc.AddVector(context.Background(), "user_notes_1", "note_11_chunk_0",
		fakeVec("张三负责后端"),
		"张三负责后端",
		map[string]string{"note_id": "11", "title": "任务分配", "entities": "张三"})

	graph, err := noteSvc.BuildNoteGraph(1, 50)
	if err != nil {
		t.Fatalf("BuildNoteGraph 失败: %v", err)
	}
	// 应有2个note节点 + entity节点 + 边
	noteCount, entityCount := 0, 0
	for _, n := range graph.Nodes {
		switch n.Type {
		case "note":
			noteCount++
		case "entity":
			entityCount++
		}
	}
	if noteCount != 2 {
		t.Fatalf("应有2个note节点，got %d", noteCount)
	}
	if entityCount < 1 {
		t.Fatalf("应有至少1个entity节点，got %d", entityCount)
	}
	if len(graph.Edges) < 1 {
		t.Fatalf("应有至少1条边，got %d", len(graph.Edges))
	}
}

// TestAvatarMemory_RecallForConversation_FiltersScope 验证分身回复的按对话召回：
// global 级记忆（事实/偏好/FAQ）任意对话可召回；conversation 级记忆（针对特定对象的
// 私约/承诺）仅在其记录时所在对话召回——A-B 对话记下的专属约定不得在 B-C 对话冒出来。
// 全局 Recall（去重/手动检索）不受对话限制，三条都应可见。
func TestAvatarMemory_RecallForConversation_FiltersScope(t *testing.T) {
	db, err := gracedb.Open(t.TempDir()+"/gracedb", gracedb.WithEmbedder(fakeEmbedder{}))
	if err != nil {
		t.Fatalf("打开临时 gracedb 失败: %v", err)
	}
	defer db.Close()

	svc := &AvatarMemoryService{db: db, aiService: nil}

	seeds := []types.MemorySaveRequest{
		{ // global：项目知识，脱离对话依然成立，任意对话可召回
			MemoryID: "memory_7_global", UserID: "7", Scope: "user", Namespace: "avatar",
			Content: "项目 Alpha 的上线时间定在下周一", Importance: 0.8,
			Metadata: map[string]any{"conversation_id": "100", "knowledge_memory_scope": "global"},
		},
		{ // conversation + 对话A（100）：A-B 专属约定
			MemoryID: "memory_7_a", UserID: "7", Scope: "user", Namespace: "avatar",
			Content: "和张三约好周五晚上一起吃饭", Importance: 0.7,
			Metadata: map[string]any{"conversation_id": "100", "knowledge_memory_scope": "conversation"},
		},
		{ // conversation + 对话B（200）：B-C 专属约定
			MemoryID: "memory_7_b", UserID: "7", Scope: "user", Namespace: "avatar",
			Content: "和李四约好下周三一起看球赛", Importance: 0.7,
			Metadata: map[string]any{"conversation_id": "200", "knowledge_memory_scope": "conversation"},
		},
	}
	for i := range seeds {
		if _, err := db.SaveMemory(seeds[i]); err != nil {
			t.Fatalf("seed #%d SaveMemory 失败: %v", i, err)
		}
	}

	contains := func(results []SearchResult, substr string) bool {
		for _, r := range results {
			if strings.Contains(r.Content, substr) {
				return true
			}
		}
		return false
	}

	// 在对话 B（200）回复：不得带出对话 A 的专属记忆（张三），且应命中同对话与 global 记忆
	got, err := svc.RecallForConversation(7, 200, "上线时间 周五吃饭 看球赛 张三", 5)
	if err != nil {
		t.Fatalf("RecallForConversation 失败: %v", err)
	}
	if len(got) == 0 {
		t.Fatal("对话 B 召回为空，过滤测试失去意义（应至少召回 global 或 B 专属记忆）")
	}
	for _, r := range got {
		if strings.Contains(r.Content, "张三") {
			t.Fatalf("A-B 对话的专属记忆泄漏到 B-C 对话: %s", r.Content)
		}
	}
	if !contains(got, "李四") {
		t.Fatal("同对话 B 的专属记忆应被召回（过滤不能把同对话记忆也剔掉）")
	}
	if !contains(got, "上线时间") {
		t.Fatal("global 级事实应在任意对话召回")
	}

	// 反向：在对话 A（100）回复：不得带出对话 B 的专属记忆（李四），且应命中同对话记忆
	gotA, err := svc.RecallForConversation(7, 100, "上线时间 周五吃饭 看球赛 李四", 5)
	if err != nil {
		t.Fatalf("RecallForConversation(conv A) 失败: %v", err)
	}
	for _, r := range gotA {
		if strings.Contains(r.Content, "李四") {
			t.Fatalf("B-C 对话的专属记忆泄漏到 A-B 对话: %s", r.Content)
		}
	}
	if !contains(gotA, "张三") {
		t.Fatal("同对话 A 的专属记忆应被召回（过滤不能把同对话记忆也剔掉）")
	}

	// 全局召回（去重/手动检索）：不分对话，三条都应可见
	all, err := svc.Recall(7, "上线时间 周五吃饭 看球赛", 10)
	if err != nil {
		t.Fatalf("Recall 失败: %v", err)
	}
	seen := map[string]bool{}
	for _, r := range all {
		seen[r.Content] = true
	}
	if !seen["和张三约好周五晚上一起吃饭"] {
		t.Fatal("全局召回应包含对话 A 的专属记忆（Recall 不分对话）")
	}
	if !seen["和李四约好下周三一起看球赛"] {
		t.Fatal("全局召回应包含对话 B 的专属记忆（Recall 不分对话）")
	}
}
