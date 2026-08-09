package ai

import (
	"strings"
	"testing"
)

// BuildCapabilityPrompt 依据 allowed 工具集动态注入能力自述。
func TestToolRegistry_BuildCapabilityPrompt(t *testing.T) {
	newRegistry := func() *ToolRegistry {
		r := NewToolRegistry(nil)
		r.RegisterTool(staticTool{name: "search_knowledge"})
		r.RegisterTool(staticTool{name: "create_user_task"})
		r.RegisterTool(staticTool{name: "disabled_tool"})
		return r
	}

	t.Run("无工具allowed时仅注入静态能力且不含工具段", func(t *testing.T) {
		r := newRegistry()
		out := r.BuildCapabilityPrompt(nil)
		// 静态能力应恒在
		if !strings.Contains(out, "你具备以下能力") {
			t.Fatalf("应包含静态能力段: %q", out)
		}
		// 不应出现任何工具名
		if strings.Contains(out, "search_knowledge") || strings.Contains(out, "disabled_tool") {
			t.Fatalf("allowed 为空不应注入工具段: %q", out)
		}
	})

	t.Run("allowed过滤只列白名单内工具", func(t *testing.T) {
		r := newRegistry()
		out := r.BuildCapabilityPrompt([]string{"search_knowledge"})
		if !strings.Contains(out, "search_knowledge") {
			t.Fatalf("应包含白名单内工具: %q", out)
		}
		if strings.Contains(out, "create_user_task") {
			t.Fatalf("不白名单外工具 create_user_task 不应出现: %q", out)
		}
	})

	t.Run("禁用工具不注入", func(t *testing.T) {
		r := newRegistry()
		_ = r.DisableTool("disabled_tool")
		out := r.BuildCapabilityPrompt([]string{"disabled_tool", "create_user_task"})
		if strings.Contains(out, "disabled_tool") {
			t.Fatalf("禁用工具不应出现: %q", out)
		}
		if !strings.Contains(out, "create_user_task") {
			t.Fatalf("启用工具应出现: %q", out)
		}
	})

	t.Run("nil注册表安全降级（仅静态能力）", func(t *testing.T) {
		var r *ToolRegistry
		out := r.BuildCapabilityPrompt([]string{"search_knowledge"})
		if !strings.Contains(out, "你具备以下能力") {
			t.Fatalf("nil 注册表应仍注入静态能力: %q", out)
		}
		if strings.Contains(out, "search_knowledge") {
			t.Fatalf("nil 注册表不应注入工具段: %q", out)
		}
	})

	t.Run("静态能力段恒定非空", func(t *testing.T) {
		if len(StaticCapabilities) == 0 {
			t.Fatal("StaticCapabilities 不应为空")
		}
		if !strings.Contains(BuildStaticCapabilitiesPrompt(), "你具备以下能力") {
			t.Fatal("BuildStaticCapabilitiesPrompt 应返回静态能力段")
		}
	})
}
