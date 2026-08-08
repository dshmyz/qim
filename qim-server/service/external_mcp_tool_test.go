package service

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// startTestMCPServer 起一个进程内 MCP server（streamable HTTP），暴露一个
// calculator 工具，供 client 侧测试真实走一遍 CallTool 链路（非纯 mock）。
func startTestMCPServer(t *testing.T) *httptest.Server {
	t.Helper()

	s := mcp.NewServer(&mcp.Implementation{Name: "test-server", Version: "0.0.1"}, nil)

	type calcParams struct {
		Expr string `json:"expr" jsonschema_description:"四则运算表达式"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "calculator",
		Description: "计算四则运算表达式",
	}, func(_ context.Context, _ *mcp.CallToolRequest, p calcParams) (*mcp.CallToolResult, any, error) {
		if p.Expr == "" {
			return nil, nil, fmt.Errorf("expr 不能为空")
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("%s = 24", p.Expr)}},
		}, nil, nil
	})

	handler := mcp.NewStreamableHTTPHandler(
		func(r *http.Request) *mcp.Server { return s },
		&mcp.StreamableHTTPOptions{Stateless: true},
	)

	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)
	return ts
}

// connectTestSession 建立到测试 server 的 client 会话。
func connectTestSession(t *testing.T, url string) *mcp.ClientSession {
	t.Helper()
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "0.0.1"}, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: url}, nil)
	require.NoError(t, err)
	t.Cleanup(func() { session.Close() })
	return session
}

func TestExternalMCPTool_ExecuteProxiesRemoteCall(t *testing.T) {
	ts := startTestMCPServer(t)
	session := connectTestSession(t, ts.URL)

	tool := NewExternalMCPTool("demo", "calculator", map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"expr": map[string]interface{}{"type": "string"},
		},
		"required": []interface{}{"expr"},
	}, session)

	// 命名空间：mcp_<conn>_<tool>
	assert.Equal(t, "mcp_demo_calculator", tool.Name())
	assert.Contains(t, tool.Description(), "demo")

	got, err := tool.Execute(map[string]interface{}{"expr": "4*6"}, &ai.CallerContext{})
	require.NoError(t, err)
	assert.Contains(t, fmt.Sprintf("%v", got), "24", "应返回远程 server 的计算文本")
}

// TestExternalMCPTool_ExecuteNoSession 验证未连接（降级态）时不 panic、直接报错，
// 供 ReAct 把工具错误回喂 LLM 而非中断主路径。
func TestExternalMCPTool_ExecuteNoSession(t *testing.T) {
	tool := NewExternalMCPTool("demo", "calculator", nil, nil)
	assert.Equal(t, "mcp_demo_calculator", tool.Name())

	_, err := tool.Execute(map[string]interface{}{"expr": "1+1"}, &ai.CallerContext{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "不可用")
}
