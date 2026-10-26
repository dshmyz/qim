package service

import (
	"fmt"
	"strings"
	"testing"

	"github.com/dshmyz/gracedb/pkg/gracedb"
	"github.com/dshmyz/gracedb/pkg/graph"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/sqlite"
	"gorm.io/gorm"
)

// TestBuildGroupKnowledgeGraph_FromStoredGraph 端到端验证群聊知识图谱拓扑：
//  1. 用与 buildDocumentGraph 完全相同的存储形态（doc:{id} 节点 + entity:{name} 节点 +
//     mentions/co_occurs 边）写入临时 gracedb；
//  2. BuildGroupKnowledgeGraph 从该存储图聚合出知识/实体节点、关系边与实体反查（related），
//     并支持查询节点。
//
// 用临时 gracedb + 内存 sqlite，不依赖外部 AI / 不碰线上库。
func TestBuildGroupKnowledgeGraph_FromStoredGraph(t *testing.T) {
	gdb, err := gracedb.Open(t.TempDir()+"/gracedb", gracedb.WithEmbedder(fakeEmbedder{}))
	if err != nil {
		t.Fatalf("打开临时 gracedb 失败: %v", err)
	}
	defer gdb.Close()

	g := gdb.Graph()
	// 与 buildDocumentGraph 一致：doc 节点 + entity 节点 + mentions/co_occurs 边
	mustUpsertNode(t, g, &graph.GraphNode{ID: "doc:1", Type: "document", Properties: map[string]string{"group_id": "1", "doc_id": "1", "title": "需求文档"}})
	mustUpsertNode(t, g, &graph.GraphNode{ID: "entity:PRD-2024-001", Type: "entity", Properties: map[string]string{"name": "PRD-2024-001"}})
	mustUpsertNode(t, g, &graph.GraphNode{ID: "entity:BUG-123", Type: "entity", Properties: map[string]string{"name": "BUG-123"}})
	mustUpsertEdge(t, g, &graph.GraphEdge{ID: "e1", FromNodeID: "doc:1", ToNodeID: "entity:PRD-2024-001", Type: "mentions", Weight: 1})
	mustUpsertEdge(t, g, &graph.GraphEdge{ID: "e2", FromNodeID: "doc:1", ToNodeID: "entity:BUG-123", Type: "mentions", Weight: 1})
	mustUpsertEdge(t, g, &graph.GraphEdge{ID: "e3", FromNodeID: "entity:PRD-2024-001", ToNodeID: "entity:BUG-123", Type: "co_occurs", Weight: 1})

	// 内存 sqlite 库：群 1 有一份文档（File 预加载出真实标题）
	sdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存 sqlite 失败: %v", err)
	}
	if err := sdb.AutoMigrate(&model.GroupDocument{}, &model.File{}); err != nil {
		t.Fatalf("建表失败: %v", err)
	}
	if err := sdb.Create(&model.File{ID: 1, UserID: 1, Name: "需求文档.pdf", Size: 10, StoragePath: "x"}).Error; err != nil {
		t.Fatalf("插入 File 失败: %v", err)
	}
	if err := sdb.Create(&model.GroupDocument{ID: 1, GroupID: 1, FileID: 1}).Error; err != nil {
		t.Fatalf("插入 GroupDocument 失败: %v", err)
	}

	svc := &GroupDocumentService{db: sdb, gracedbDB: gdb}
	res, err := svc.BuildGroupKnowledgeGraph(1, "", 50, 0)
	if err != nil {
		t.Fatalf("BuildGroupKnowledgeGraph 失败: %v", err)
	}

	// 应有知识(文档)节点 + 实体节点，以及关系边
	if res.TotalNodes == 0 {
		t.Fatalf("期望非空节点，实际 0")
	}
	if res.TotalEdges == 0 {
		t.Fatalf("期望非空边，实际 0")
	}
	if res.KnowledgeCount != 1 {
		t.Errorf("期望 1 个知识(文档)节点，实际 %d", res.KnowledgeCount)
	}

	var entityNode map[string]interface{}
	var knowNode map[string]interface{}
	for _, n := range res.Nodes {
		switch n["type"] {
		case "entity":
			entityNode = n
		case "knowledge":
			knowNode = n
		case "query":
			t.Errorf("query=='' 时不应产出查询节点，实际出现")
		}
	}
	if entityNode == nil {
		t.Fatalf("期望存在 entity 类型节点，实际: %+v", res.Nodes)
	}
	if knowNode == nil {
		t.Fatalf("期望存在 knowledge 类型节点，实际: %+v", res.Nodes)
	}
	// 文档标题应取自 File.Name（预加载）
	if knowNode["label"] != "需求文档.pdf" {
		t.Errorf("知识节点标签应为文档标题，实际 %v", knowNode["label"])
	}
	// 实体反查：related 应包含所在文档标题（对齐分身 memories[].terms）
	data, _ := entityNode["data"].(map[string]interface{})
	related, _ := data["related"].([]string)
	if len(related) == 0 || related[0] != "需求文档.pdf" {
		t.Errorf("实体反查应包含文档标题，实际 %v", related)
	}
}

// TestBuildGroupKnowledgeGraph_WithQuery 带查询词时产出 query 节点并连到知识节点。
func TestBuildGroupKnowledgeGraph_WithQuery(t *testing.T) {
	gdb, err := gracedb.Open(t.TempDir()+"/gracedb", gracedb.WithEmbedder(fakeEmbedder{}))
	if err != nil {
		t.Fatalf("打开临时 gracedb 失败: %v", err)
	}
	defer gdb.Close()

	g := gdb.Graph()
	mustUpsertNode(t, g, &graph.GraphNode{ID: "doc:1", Type: "document", Properties: map[string]string{"group_id": "1", "doc_id": "1", "title": "需求文档"}})
	mustUpsertNode(t, g, &graph.GraphNode{ID: "entity:PRD-2024-001", Type: "entity", Properties: map[string]string{"name": "PRD-2024-001"}})
	mustUpsertEdge(t, g, &graph.GraphEdge{ID: "e1", FromNodeID: "doc:1", ToNodeID: "entity:PRD-2024-001", Type: "mentions", Weight: 1})

	sdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存 sqlite 失败: %v", err)
	}
	_ = sdb.AutoMigrate(&model.GroupDocument{}, &model.File{})
	_ = sdb.Create(&model.File{ID: 1, UserID: 1, Name: "需求文档.pdf", Size: 10, StoragePath: "x"})
	_ = sdb.Create(&model.GroupDocument{ID: 1, GroupID: 1, FileID: 1})

	svc := &GroupDocumentService{db: sdb, gracedbDB: gdb}
	res, err := svc.BuildGroupKnowledgeGraph(1, "PRD", 50, 0)
	if err != nil {
		t.Fatalf("BuildGroupKnowledgeGraph 失败: %v", err)
	}

	foundQuery := false
	for _, n := range res.Nodes {
		if n["type"] == "query" {
			foundQuery = true
		}
	}
	if !foundQuery {
		t.Fatalf("带 query 时应产出 query 节点，实际: %+v", res.Nodes)
	}
}

// TestBuildGroupKnowledgeGraph_EmptyGroup 无文档/图谱数据的群返回空结构而非报错。
func TestBuildGroupKnowledgeGraph_EmptyGroup(t *testing.T) {
	gdb, err := gracedb.Open(t.TempDir()+"/gracedb", gracedb.WithEmbedder(fakeEmbedder{}))
	if err != nil {
		t.Fatalf("打开临时 gracedb 失败: %v", err)
	}
	defer gdb.Close()

	sdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存 sqlite 失败: %v", err)
	}
	_ = sdb.AutoMigrate(&model.GroupDocument{}, &model.File{})

	svc := &GroupDocumentService{db: sdb, gracedbDB: gdb}
	res, err := svc.BuildGroupKnowledgeGraph(99, "", 50, 0)
	if err != nil {
		t.Fatalf("空群不应报错，实际: %v", err)
	}
	if res.TotalNodes != 0 || len(res.Nodes) != 0 {
		t.Fatalf("空群应返回空节点，实际 %+v", res.Nodes)
	}
}

// TestValidGraphEntity 实体过滤：编号类 token 保留（大小写均可），自然语言连字符词与
// 通用技术编码视为噪音。
func TestValidGraphEntity(t *testing.T) {
	cases := map[string]bool{
		"PRD-2024-001": true,
		"BUG-123":      true,
		"bug-123":      true,
		"utf-8":        false,
		"sha-256":      false,
		"co-occurs":    false,
		"e-mail":       false,
		"":             false,
		"AB":           false, // 太短
	}
	for name, want := range cases {
		if got := validGraphEntity(name); got != want {
			t.Errorf("validGraphEntity(%q)=%v want %v", name, got, want)
		}
	}
}

func newGraphTestService(t *testing.T) (*GroupDocumentService, *gracedb.DB, *gorm.DB) {
	t.Helper()
	gdb, err := gracedb.Open(t.TempDir()+"/gracedb", gracedb.WithEmbedder(fakeEmbedder{}))
	if err != nil {
		t.Fatalf("打开临时 gracedb 失败: %v", err)
	}
	t.Cleanup(func() { _ = gdb.Close() })

	sdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存 sqlite 失败: %v", err)
	}
	if err := sdb.AutoMigrate(&model.GroupDocument{}, &model.File{}); err != nil {
		t.Fatalf("建表失败: %v", err)
	}
	return &GroupDocumentService{db: sdb, gracedbDB: gdb}, gdb, sdb
}

func seedDoc(t *testing.T, sdb *gorm.DB, docID, groupID uint, name string) {
	t.Helper()
	if err := sdb.Create(&model.File{ID: docID, UserID: 1, Name: name, Size: 10, StoragePath: "x"}).Error; err != nil {
		t.Fatalf("插入 File 失败: %v", err)
	}
	if err := sdb.Create(&model.GroupDocument{ID: docID, GroupID: groupID, FileID: docID}).Error; err != nil {
		t.Fatalf("插入 GroupDocument 失败: %v", err)
	}
}

// TestBuildGroupKnowledgeGraph_CrossDocCoOccurrence 只有跨文档共现的实体对才建 co_occurs 边，
// 单文档内不再产生两两全连接。
func TestBuildGroupKnowledgeGraph_CrossDocCoOccurrence(t *testing.T) {
	svc, _, sdb := newGraphTestService(t)
	seedDoc(t, sdb, 1, 1, "A.pdf")
	seedDoc(t, sdb, 2, 1, "B.pdf")

	if err := svc.buildDocumentGraph(1, 1, "A.pdf", "PRD-2024-001 BUG-123"); err != nil {
		t.Fatalf("build doc1 失败: %v", err)
	}
	if err := svc.buildDocumentGraph(1, 2, "B.pdf", "PRD-2024-001 BUG-123"); err != nil {
		t.Fatalf("build doc2 失败: %v", err)
	}

	res, err := svc.BuildGroupKnowledgeGraph(1, "", 50, 0)
	if err != nil {
		t.Fatalf("构建图谱失败: %v", err)
	}
	co := 0
	for _, e := range res.Edges {
		if e["type"] == "co_occurs" {
			co++
		}
	}
	if co != 1 {
		t.Fatalf("跨两个文档共现的实体对应有 1 条 co_occurs 边，实际 %d（edges=%+v）", co, res.Edges)
	}
}

// TestBuildGroupKnowledgeGraph_NoiseEntityFiltered 历史存储里的小写噪音实体会在构建期被过滤。
func TestBuildGroupKnowledgeGraph_NoiseEntityFiltered(t *testing.T) {
	svc, gdb, sdb := newGraphTestService(t)
	seedDoc(t, sdb, 1, 1, "A.pdf")

	g := gdb.Graph()
	mustUpsertNode(t, g, &graph.GraphNode{ID: "doc:1", Type: "document", Properties: map[string]string{"group_id": "1"}})
	for _, name := range []string{"PRD-2024-001", "utf-8", "co-occurs"} {
		mustUpsertNode(t, g, &graph.GraphNode{ID: "entity:" + name, Type: "entity", Properties: map[string]string{"name": name}})
		mustUpsertEdge(t, g, &graph.GraphEdge{ID: "m-" + name, FromNodeID: "doc:1", ToNodeID: "entity:" + name, Type: "mentions", Weight: 1})
	}

	res, err := svc.BuildGroupKnowledgeGraph(1, "", 50, 0)
	if err != nil {
		t.Fatalf("构建图谱失败: %v", err)
	}
	for _, n := range res.Nodes {
		if n["type"] != "entity" {
			continue
		}
		label, _ := n["label"].(string)
		if label == "utf-8" || label == "co-occurs" {
			t.Fatalf("噪音实体不应进入图谱: %v", label)
		}
	}
}

// TestBuildGroupKnowledgeGraph_EdgeBudget 边预算生效且返回截断前真实总数。
func TestBuildGroupKnowledgeGraph_EdgeBudget(t *testing.T) {
	svc, _, sdb := newGraphTestService(t)
	seedDoc(t, sdb, 1, 1, "A.pdf")
	seedDoc(t, sdb, 2, 1, "B.pdf")

	text := ""
	for i := 1; i <= 20; i++ {
		text += " T-" + fmt.Sprintf("%02d", i)
	}
	if err := svc.buildDocumentGraph(1, 1, "A.pdf", text); err != nil {
		t.Fatalf("build doc1 失败: %v", err)
	}
	if err := svc.buildDocumentGraph(1, 2, "B.pdf", text); err != nil {
		t.Fatalf("build doc2 失败: %v", err)
	}

	res, err := svc.BuildGroupKnowledgeGraph(1, "", 50, 10)
	if err != nil {
		t.Fatalf("构建图谱失败: %v", err)
	}
	if len(res.Edges) != 10 {
		t.Fatalf("边预算 10 应恰好返回 10 条，实际 %d", len(res.Edges))
	}
	if !res.Truncated {
		t.Fatalf("超出预算时应标记 truncated")
	}
	if res.TotalEdges <= len(res.Edges) {
		t.Fatalf("TotalEdges 应为截断前总数，实际 %d", res.TotalEdges)
	}
}

// TestRebuildGroupKnowledgeGraph_MigratesLegacy 重建把旧版全局节点迁移为群内命名空间，
// 并清掉旧的全连接 co_occurs 边。
func TestRebuildGroupKnowledgeGraph_MigratesLegacy(t *testing.T) {
	svc, gdb, sdb := newGraphTestService(t)
	seedDoc(t, sdb, 1, 1, "A.pdf")

	g := gdb.Graph()
	mustUpsertNode(t, g, &graph.GraphNode{ID: "doc:1", Type: "document", Properties: map[string]string{"group_id": "1", "title": "A"}})
	mustUpsertNode(t, g, &graph.GraphNode{ID: "entity:PRD-2024-001", Type: "entity", Properties: map[string]string{"name": "PRD-2024-001"}})
	mustUpsertNode(t, g, &graph.GraphNode{ID: "entity:BUG-123", Type: "entity", Properties: map[string]string{"name": "BUG-123"}})
	mustUpsertEdge(t, g, &graph.GraphEdge{ID: "m1", FromNodeID: "doc:1", ToNodeID: "entity:PRD-2024-001", Type: "mentions", Weight: 1})
	mustUpsertEdge(t, g, &graph.GraphEdge{ID: "m2", FromNodeID: "doc:1", ToNodeID: "entity:BUG-123", Type: "mentions", Weight: 1})
	mustUpsertEdge(t, g, &graph.GraphEdge{ID: "co1", FromNodeID: "entity:PRD-2024-001", ToNodeID: "entity:BUG-123", Type: "co_occurs", Weight: 1})

	if _, err := svc.RebuildGroupKnowledgeGraph(1); err != nil {
		t.Fatalf("重建失败: %v", err)
	}

	if _, err := g.GetNode("doc:1"); err == nil {
		t.Errorf("旧版文档节点应被清理")
	}
	if _, err := g.GetNode("entity:PRD-2024-001"); err == nil {
		t.Errorf("旧版实体节点应被清理")
	}
	n, err := g.GetNode("g1:doc:1")
	if err != nil || n == nil {
		t.Fatalf("新文档节点应存在: %v", err)
	}
	if n.Properties["entities"] == "" {
		t.Errorf("新文档节点应带 entities 属性")
	}
}

// TestDeleteDocumentGraph_KeepsSharedEntities 删除文档只回收不再被提及的实体。
func TestDeleteDocumentGraph_KeepsSharedEntities(t *testing.T) {
	svc, gdb, sdb := newGraphTestService(t)
	seedDoc(t, sdb, 1, 1, "A.pdf")
	seedDoc(t, sdb, 2, 1, "B.pdf")

	if err := svc.buildDocumentGraph(1, 1, "A.pdf", "SHARED-1 UNIQUE-1"); err != nil {
		t.Fatalf("build doc1 失败: %v", err)
	}
	if err := svc.buildDocumentGraph(1, 2, "B.pdf", "SHARED-1"); err != nil {
		t.Fatalf("build doc2 失败: %v", err)
	}
	if err := svc.deleteDocumentGraph(1, 1); err != nil {
		t.Fatalf("删除文档图谱失败: %v", err)
	}

	g := gdb.Graph()
	if _, err := g.GetNode("g1:entity:UNIQUE-1"); err == nil {
		t.Errorf("只被已删文档引用的实体应被回收")
	}
	if _, err := g.GetNode("g1:entity:SHARED-1"); err != nil {
		t.Errorf("仍被其它文档引用的实体应保留: %v", err)
	}
}

// TestBuildGroupKnowledgeGraph_DeterministicUnderEdgeBudget 边预算触顶时，同一份数据
// 两次构建必须输出完全相同的边（回归：co_occurs 候选曾按 map 遍历顺序 append，同权重
// 边在预算边界上的取舍随机，刷新一次拓扑就变）。
func TestBuildGroupKnowledgeGraph_DeterministicUnderEdgeBudget(t *testing.T) {
	svc, _, sdb := newGraphTestService(t)
	const text = "E-01 E-02 E-03 E-04 E-05"
	for docID := uint(1); docID <= 3; docID++ {
		seedDoc(t, sdb, docID, 1, fmt.Sprintf("doc%d.pdf", docID))
		if err := svc.buildDocumentGraph(1, docID, fmt.Sprintf("doc%d.pdf", docID), text); err != nil {
			t.Fatalf("build doc%d 失败: %v", docID, err)
		}
	}

	// 5 个实体都在 3 篇里出现 → 10 条 co_occurs 边权重全为 3（同权重），
	// maxEdges=20 < 15 条 mentions + 10 条 co_occurs，必然在预算边界上丢弃 5 条。
	snapshot := func() string {
		res, err := svc.BuildGroupKnowledgeGraph(1, "", 50, 20)
		if err != nil {
			t.Fatalf("构建图谱失败: %v", err)
		}
		if len(res.Edges) != 20 {
			t.Fatalf("边预算 20 应恰好返回 20 条，实际 %d", len(res.Edges))
		}
		parts := make([]string, 0, len(res.Edges))
		for _, e := range res.Edges {
			parts = append(parts, fmt.Sprintf("%v>%v:%v:%v", e["source"], e["target"], e["type"], e["weight"]))
		}
		return strings.Join(parts, "|")
	}

	want := snapshot()
	for i := 0; i < 5; i++ {
		if got := snapshot(); got != want {
			t.Fatalf("第 %d 次构建的边集与首次不一致（同权重 co_occurs 截断不确定）\n首次: %s\n本次: %s", i+2, want, got)
		}
	}
}

// TestDeleteDocumentVectors_MissingCollectionIsNoop 群集合不存在（文档从未处理过）时
// 解绑应视为"无可删"，不能报错——否则未处理过的文档永远解绑不掉。
func TestDeleteDocumentVectors_MissingCollectionIsNoop(t *testing.T) {
	svc, _, _ := newGraphTestService(t)
	if err := svc.deleteDocumentVectors("group_never_processed", 1); err != nil {
		t.Fatalf("集合不存在应返回 nil，实际: %v", err)
	}
}

func mustUpsertNode(t *testing.T, g *graph.GraphStore, n *graph.GraphNode) {
	t.Helper()
	if err := g.UpsertNode(n); err != nil {
		t.Fatalf("UpsertNode %s 失败: %v", n.ID, err)
	}
}

func mustUpsertEdge(t *testing.T, g *graph.GraphStore, e *graph.GraphEdge) {
	t.Helper()
	if err := g.UpsertEdge(e); err != nil {
		t.Fatalf("UpsertEdge %s 失败: %v", e.ID, err)
	}
}
