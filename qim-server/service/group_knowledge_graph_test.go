package service

import (
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
	res, err := svc.BuildGroupKnowledgeGraph(1, "", 50)
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
	res, err := svc.BuildGroupKnowledgeGraph(1, "PRD", 50)
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
	res, err := svc.BuildGroupKnowledgeGraph(99, "", 50)
	if err != nil {
		t.Fatalf("空群不应报错，实际: %v", err)
	}
	if res.TotalNodes != 0 || len(res.Nodes) != 0 {
		t.Fatalf("空群应返回空节点，实际 %+v", res.Nodes)
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
