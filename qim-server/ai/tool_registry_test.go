package ai

import (
	"strings"
	"testing"
)

// TestToolRegistryRejectsCaseVariantCollision 锁定 RegisterTool 的大小写碰撞守卫：
// 两个工具名仅大小写不同（如 send_message vs Send_Message）会让 canonicalKey 的大小写
// 不敏感回退扫描（map 迭代顺序不确定）产生不确定结果，注册期必须拒绝后者，杜绝该状态。
// 同一名字重复注册仍允许（幂等覆盖，外部 MCP 重连会重注册同名工具）。
func TestToolRegistryRejectsCaseVariantCollision(t *testing.T) {
	registry := NewToolRegistry(nil)

	if err := registry.RegisterTool(staticTool{name: "send_message"}); err != nil {
		t.Fatalf("首次注册 send_message 不应失败: %v", err)
	}
	// 同一名字重复注册：幂等覆盖，允许。
	if err := registry.RegisterTool(staticTool{name: "send_message"}); err != nil {
		t.Fatalf("同名重复注册应幂等允许: %v", err)
	}
	// 仅大小写不同：拒绝，且被拒工具不进入注册表（注册表仅保留原 send_message 一个键）。
	err := registry.RegisterTool(staticTool{name: "Send_Message"})
	if err == nil {
		t.Fatal("大小写不同的工具注册应被拒绝")
	}
	if !strings.Contains(err.Error(), "Send_Message") {
		t.Fatalf("错误应指明冲突工具名，实际: %v", err)
	}
	if len(registry.tools) != 1 {
		t.Fatalf("被拒绝的工具不应残留到注册表，实际键数 %d: %v", len(registry.tools), registry.tools)
	}
	if _, ok := registry.tools["Send_Message"]; ok {
		t.Fatal("被拒绝的 Send_Message 不应出现在注册表原始键中")
	}

	// 反向注册顺序：先大小写变体，原名字后到同样被拒。
	registry2 := NewToolRegistry(nil)
	if err := registry2.RegisterTool(staticTool{name: "SEND_MESSAGE"}); err != nil {
		t.Fatalf("首次注册 SEND_MESSAGE 不应失败: %v", err)
	}
	if err := registry2.RegisterTool(staticTool{name: "send_message"}); err == nil {
		t.Fatal("与既有工具大小写不同的后注册应被拒绝")
	}
}

// TestToolRegistryCanonicalKeyDeterministic 单键时大小写不敏感查找确定命中原注册键。
// 守卫保证注册表内不存在仅大小写不同的键，canonicalKey 回退扫描不再有歧义。
func TestToolRegistryCanonicalKeyDeterministic(t *testing.T) {
	registry := NewToolRegistry(nil)
	if err := registry.RegisterTool(staticTool{name: "send_message"}); err != nil {
		t.Fatalf("register: %v", err)
	}

	for _, probe := range []string{"send_message", "Send_Message", "SEND_MESSAGE", "SEND_message"} {
		tool, ok := registry.GetTool(probe)
		if !ok {
			t.Fatalf("GetTool(%q) 应命中", probe)
		}
		if tool.Name() != "send_message" {
			t.Fatalf("GetTool(%q) 应返回原注册键 send_message，实际 %q", probe, tool.Name())
		}
	}
}
