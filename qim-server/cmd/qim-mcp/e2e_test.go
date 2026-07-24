//go:build mcp_e2e

// 手动端到端测试：spawn 真 qim-mcp 子进程（stdio），连真 qim-server，验证 6 个工具。
//
// 跑法（server 已在 :8096，且 DB 已 seed bot+token）：
//
//	go test -tags mcp_e2e ./cmd/qim-mcp/ -run TestE2E -v -args -binary /tmp/qim-mcp-e2e
package main_test

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"net/http"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	serverURL = flag.String("server", "http://127.0.0.1:8096", "QIM server URL")
	token     = flag.String("token", "qbot_mcp_e2e_test_token", "bot token")
	binary    = flag.String("binary", "/tmp/qim-mcp-e2e", "qim-mcp binary path")
	toUser    = flag.Uint64("to-user", 13, "目标人类用户 ID（seed 的 Dave）")
)

func dial(t *testing.T) *mcp.ClientSession {
	t.Helper()
	cmd := exec.Command(*binary, "--server", *serverURL, "--token", *token)
	t.Logf("spawn: %s %v", cmd.Path, cmd.Args[1:])
	transport := &mcp.CommandTransport{Command: cmd}
	c := mcp.NewClient(&mcp.Implementation{Name: "e2e-test"}, nil)
	cs, err := c.Connect(context.Background(), transport, nil)
	require.NoError(t, err)
	t.Cleanup(func() { cs.Close() })
	return cs
}

func toolText(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()
	require.False(t, res.IsError, "tool 返回错误: %v", res)
	require.Len(t, res.Content, 1)
	tc, ok := res.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	return tc.Text
}

func parseFieldUint(t *testing.T, jsonStr, field string) uint64 {
	t.Helper()
	var m map[string]any
	require.NoError(t, json.Unmarshal([]byte(jsonStr), &m))
	v, ok := m[field]
	require.True(t, ok, "响应缺少 %s: %s", field, jsonStr)
	f, ok := v.(float64)
	require.True(t, ok)
	return uint64(f)
}

func TestE2E_ToolsList(t *testing.T) {
	cs := dial(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	res, err := cs.ListTools(ctx, nil)
	require.NoError(t, err)
	names := map[string]bool{}
	for _, tl := range res.Tools {
		names[tl.Name] = true
	}
	for _, want := range []string{"list_messages", "poll_messages", "send_message", "start_streaming_message", "append_streaming_chunk", "finish_streaming_message"} {
		assert.True(t, names[want], "缺少工具 %s", want)
	}
}

func TestE2E_SendThenList(t *testing.T) {
	cs := dial(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	res, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name:      "send_message",
		Arguments: map[string]any{"to_user_id": *toUser, "content": "e2e hello from mcp", "msg_type": "markdown"},
	})
	require.NoError(t, err)
	txt := toolText(t, res)
	t.Logf("send_message => %s", txt)
	require.Contains(t, txt, "message_id")
	convID := parseFieldUint(t, txt, "conversation_id")
	require.NotZero(t, convID)

	res2, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name:      "list_messages",
		Arguments: map[string]any{"thread_id": convID, "limit": 10},
	})
	require.NoError(t, err)
	txt2 := toolText(t, res2)
	t.Logf("list_messages => %s", txt2)
	assert.Contains(t, txt2, "e2e hello from mcp")
}

func TestE2E_Streaming(t *testing.T) {
	cs := dial(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	res, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name:      "start_streaming_message",
		Arguments: map[string]any{"to_user_id": *toUser},
	})
	require.NoError(t, err)
	startTxt := toolText(t, res)
	t.Logf("start_streaming => %s", startTxt)
	msgID := parseFieldUint(t, startTxt, "message_id")
	convID := parseFieldUint(t, startTxt, "conversation_id")
	require.NotZero(t, msgID)

	for _, delta := range []string{"流式 ", "测试"} {
		_, err := cs.CallTool(ctx, &mcp.CallToolParams{
			Name:      "append_streaming_chunk",
			Arguments: map[string]any{"message_id": msgID, "delta": delta},
		})
		require.NoError(t, err)
	}
	_, err = cs.CallTool(ctx, &mcp.CallToolParams{
		Name:      "finish_streaming_message",
		Arguments: map[string]any{"message_id": msgID},
	})
	require.NoError(t, err)

	res2, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name:      "list_messages",
		Arguments: map[string]any{"thread_id": convID, "limit": 5},
	})
	require.NoError(t, err)
	txt := toolText(t, res2)
	t.Logf("streaming final => %s", txt)
	assert.Contains(t, txt, "流式 测试", "流式分段应合并为最终 markdown")
}

// --- HTTP (StreamableHTTP) e2e ---

// dialHTTP starts qim-mcp in HTTP mode and returns a connected ClientSession.
func dialHTTP(t *testing.T) *mcp.ClientSession {
	t.Helper()
	cmd := exec.Command(*binary, "--transport", "http", "--addr", ":0", "--server", *serverURL)
	stderr, err := cmd.StderrPipe()
	require.NoError(t, err)
	require.NoError(t, cmd.Start())

	// Read stderr until we find the startup log line with the assigned port.
	scanner := bufio.NewScanner(stderr)
	var addr string
	for scanner.Scan() {
		line := scanner.Text()
		t.Logf("stderr: %s", line)
		if strings.Contains(line, "qim-mcp HTTP") && strings.Contains(line, "addr=") {
			// slog default format: ... addr=:PORTN ...
			for _, part := range strings.Fields(line) {
				if strings.HasPrefix(part, "addr=") {
					addr = strings.TrimPrefix(part, "addr=")
					break
				}
			}
			break
		}
	}
	require.NotEmpty(t, addr, "未捕获到 HTTP 启动地址")

	// HTTP client that injects Authorization: Bearer <token>
	httpClient := &http.Client{
		Transport: &authTransport{
			base:  http.DefaultTransport,
			token: *token,
		},
	}

	transport := &mcp.StreamableClientTransport{
		Endpoint:   "http://" + addr + "/mcp",
		HTTPClient: httpClient,
	}
	c := mcp.NewClient(&mcp.Implementation{Name: "e2e-http-test"}, nil)
	cs, err := c.Connect(context.Background(), transport, nil)
	require.NoError(t, err)
	t.Cleanup(func() {
		cs.Close()
		cmd.Process.Kill()
	})
	return cs
}

type authTransport struct {
	base  http.RoundTripper
	token string
}

func (a *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+a.token)
	return a.base.RoundTrip(req)
}

func TestE2E_HTTP_ToolsList(t *testing.T) {
	cs := dialHTTP(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name:      "list_messages",
		Arguments: map[string]any{"thread_id": 1, "limit": 3},
	})
	require.NoError(t, err)
	t.Logf("HTTP list_messages => %v", res)
}

func TestE2E_HTTP_AuthReject(t *testing.T) {
	// Start binary in HTTP mode, connect WITHOUT auth token → should fail.
	cmd := exec.Command(*binary, "--transport", "http", "--addr", ":0", "--server", *serverURL)
	stderr, err := cmd.StderrPipe()
	require.NoError(t, err)
	require.NoError(t, cmd.Start())
	defer cmd.Process.Kill()

	scanner := bufio.NewScanner(stderr)
	var addr string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "addr=") {
			for _, part := range strings.Fields(line) {
				if strings.HasPrefix(part, "addr=") {
					addr = strings.TrimPrefix(part, "addr=")
					break
				}
			}
			break
		}
	}
	require.NotEmpty(t, addr)

	// Connect without Authorization header
	transport := &mcp.StreamableClientTransport{
		Endpoint:   "http://" + addr + "/mcp",
		HTTPClient: http.DefaultClient,
	}
	c := mcp.NewClient(&mcp.Implementation{Name: "e2e-noauth"}, nil)
	_, err = c.Connect(context.Background(), transport, nil)
	assert.Error(t, err, "无 token 连接应失败")
}
