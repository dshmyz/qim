package service

import (
	"testing"
)

func TestBuildMemoryGraphFromRecords(t *testing.T) {
	records := []MemoryRecord{
		{DocID: "m1", Content: "小明喜欢喝咖啡",
			Entities: []string{"小明"}, Themes: []string{"咖啡", "偏好"}},
		{DocID: "m2", Content: "小明在互联网公司上班",
			Entities: []string{"小明"}, Themes: []string{"工作", "互联网公司"}},
		{DocID: "m3", Content: "她们一起喝咖啡聊工作",
			Entities: []string{}, Themes: []string{"咖啡", "工作"}},
	}

	g := buildMemoryGraphFromRecords(records)

	// 节点：小明(2·实体), 咖啡(2·主题), 工作(2·主题), 偏好(1·主题), 互联网公司(1·主题)
	if len(g.Nodes) != 5 {
		t.Fatalf("nodes = %d, want 5", len(g.Nodes))
	}
	countByName := map[string]int{}
	typeByName := map[string]string{}
	for _, n := range g.Nodes {
		countByName[n.Name] = n.Count
		typeByName[n.Name] = n.Type
		if n.ID != n.Name {
			t.Errorf("node ID %q != Name %q", n.ID, n.Name)
		}
	}
	if countByName["小明"] != 2 {
		t.Errorf("小明 count = %d, want 2", countByName["小明"])
	}
	if countByName["咖啡"] != 2 {
		t.Errorf("咖啡 count = %d, want 2", countByName["咖啡"])
	}
	if countByName["工作"] != 2 {
		t.Errorf("工作 count = %d, want 2", countByName["工作"])
	}
	if typeByName["小明"] != "entity" {
		t.Errorf("小明 type = %q, want entity", typeByName["小明"])
	}
	if typeByName["咖啡"] != "theme" {
		t.Errorf("咖啡 type = %q, want theme", typeByName["咖啡"])
	}

	// 边：共现对去重、weight=共同出现的记忆数。
	// 记忆1:{小明,咖啡,偏好} 记忆2:{小明,工作,互联网公司} 记忆3:{咖啡,工作}
	// 共现组合：小明-咖啡(1), 小明-偏好(1), 咖啡-偏好(1), 小明-工作(1),
	//           小明-互联网公司(1), 工作-互联网公司(1), 咖啡-工作(1) → 共 7 条，均 weight=1
	if len(g.Edges) != 7 {
		t.Fatalf("edges = %d, want 7\n%+v", len(g.Edges), g.Edges)
	}
	// 端点是排序后的 key（无向边），按 source/target 组合断言存在
	hasEdge := func(a, b string) bool {
		for _, e := range g.Edges {
			if (e.Source == a && e.Target == b) || (e.Source == b && e.Target == a) {
				return true
			}
		}
		return false
	}
	if !hasEdge("小明", "咖啡") {
		t.Errorf("missing edge 小明-咖啡")
	}
	if !hasEdge("咖啡", "工作") {
		t.Errorf("missing edge 咖啡-工作")
	}
	for _, e := range g.Edges {
		if e.Weight != 1 {
			t.Errorf("edge %s-%s weight = %d, want 1", e.Source, e.Target, e.Weight)
		}
	}

	// Memories 应带 Terms，供点节点回查"包含此名词的记忆"
	if len(g.Memories) != 3 {
		t.Fatalf("memories = %d, want 3", len(g.Memories))
	}
	if len(g.Memories[0].Terms) != 3 {
		t.Errorf("memory m1 terms = %v, want 3 terms", g.Memories[0].Terms)
	}
}

func TestBuildMemoryGraphEmptyAndBlank(t *testing.T) {
	// 空输入 -> 空图
	g := buildMemoryGraphFromRecords(nil)
	if len(g.Nodes) != 0 || len(g.Edges) != 0 || len(g.Memories) != 0 {
		t.Fatalf("empty input should yield empty graph, got n=%d e=%d m=%d",
			len(g.Nodes), len(g.Edges), len(g.Memories))
	}

	// 纯空白实体/主题应被剥离，不产生空名节点
	g2 := buildMemoryGraphFromRecords([]MemoryRecord{
		{DocID: "m1", Content: "内容", Entities: []string{"  ", ""}, Themes: []string{"\t", ""}},
	})
	if len(g2.Nodes) != 0 {
		t.Fatalf("blank terms should be dropped, got %d nodes", len(g2.Nodes))
	}

	// 单条记忆内的重复名词只算一个节点，Count 为 1（同一记忆内 addNode 去重）
	g3 := buildMemoryGraphFromRecords([]MemoryRecord{
		{DocID: "m1", Content: "c", Entities: []string{"A"}, Themes: []string{"A"}},
	})
	if len(g3.Nodes) != 1 {
		t.Fatalf("dedup within one memory should collapse to 1 node, got %d", len(g3.Nodes))
	}
	if g3.Nodes[0].Count != 1 {
		t.Errorf("single memory node count = %d, want 1", g3.Nodes[0].Count)
	}
	if len(g3.Edges) != 0 {
		t.Errorf("single term memory should have 0 edges, got %d", len(g3.Edges))
	}
}

// TestAvatarMemoryService_ConflictThreshold
// 冲突检测门槛默认 0.7；注入阈值服务后经 GetFloat 读取（nil DB 回退默认值 0.7）。
func TestAvatarMemoryService_ConflictThreshold(t *testing.T) {
	svc := &AvatarMemoryService{}
	if got := svc.conflictThreshold(); got != 0.7 {
		t.Errorf("默认冲突门槛应为 0.7，got %v", got)
	}

	// 注入 nil DB 的阈值服务：GetFloat 无配置时回退传入默认 0.7，与硬编码一致。
	svc.SetThresholdService(NewAiThresholdService(nil))
	if got := svc.conflictThreshold(); got != 0.7 {
		t.Errorf("注入阈值服务后应读回默认 0.7，got %v", got)
	}
}
