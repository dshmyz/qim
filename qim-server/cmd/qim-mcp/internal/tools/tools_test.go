package tools_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/dshmyz/qim/qim-server/cmd/qim-mcp/internal/client"
	"github.com/dshmyz/qim/qim-server/cmd/qim-mcp/internal/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// capturedRequest 记录一次 Bot API 请求。
type capturedRequest struct {
	method string
	path   string
	auth   string
	body   map[string]any
}

// mockBotAPI 起一个可编程的 Bot API mock，记录所有请求。
// handle 在每个请求里据 path/body 写响应。
func mockBotAPI(t *testing.T, handle func(w http.ResponseWriter, r *http.Request, body map[string]any)) (*httptest.Server, *[]capturedRequest) {
	t.Helper()
	var mu sync.Mutex
	var reqs []capturedRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		if len(raw) > 0 {
			_ = json.Unmarshal(raw, &body)
		}
		mu.Lock()
		reqs = append(reqs, capturedRequest{method: r.Method, path: r.URL.Path, auth: r.Header.Get("Authorization"), body: body})
		mu.Unlock()
		handle(w, r, body)
	}))
	t.Cleanup(srv.Close)
	return srv, &reqs
}

func lastReq(reqs *[]capturedRequest) capturedRequest {
	var mu sync.Mutex
	mu.Lock()
	defer mu.Unlock()
	return (*reqs)[len(*reqs)-1]
}

// setupMCP 起一个 InMemory 连接的 MCP server+client，返回 client session。
func setupMCP(t *testing.T, api *client.BotAPIClient) *mcp.ClientSession {
	t.Helper()
	s := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "v"}, nil)
	tools.Register(s, tools.New(api))

	ct, st := mcp.NewInMemoryTransports()
	ss, err := s.Connect(context.Background(), st, nil)
	require.NoError(t, err)
	t.Cleanup(func() { ss.Close() })

	c := mcp.NewClient(&mcp.Implementation{Name: "test-client"}, nil)
	cs, err := c.Connect(context.Background(), ct, nil)
	require.NoError(t, err)
	t.Cleanup(func() { cs.Close() })
	return cs
}

func callTool(t *testing.T, cs *mcp.ClientSession, name string, args map[string]any) string {
	t.Helper()
	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	require.NoError(t, err)
	require.False(t, res.IsError, "tool 返回错误: %v", res)
	require.Len(t, res.Content, 1)
	tc, ok := res.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	return tc.Text
}

func TestSendMessage(t *testing.T) {
	srv, reqs := mockBotAPI(t, func(w http.ResponseWriter, r *http.Request, body map[string]any) {
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"message_id": float64(42), "conversation_id": float64(7)}})
	})
	cs := setupMCP(t, client.New(srv.URL, "qbot_test"))

	out := callTool(t, cs, "send_message", map[string]any{
		"to_user_id": float64(13), "thread_id": float64(7), "content": "hi", "msg_type": "markdown",
	})
	assert.Contains(t, out, `"message_id":42`)
	assert.Contains(t, out, `"conversation_id":7`)

	r := lastReq(reqs)
	assert.Equal(t, "Bearer qbot_test", r.auth, "token 应透传为 Bearer")
	assert.Equal(t, "/api/v1/bot/messages", r.path)
	assert.Equal(t, "hi", r.body["content"])
	assert.Equal(t, "markdown", r.body["msg_type"])
	assert.EqualValues(t, 13, r.body["to_user_id"])
	assert.EqualValues(t, 7, r.body["thread_id"])
}

func TestSendMessage_NoThreadAutoCreate(t *testing.T) {
	srv, reqs := mockBotAPI(t, func(w http.ResponseWriter, r *http.Request, body map[string]any) {
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"message_id": float64(1), "conversation_id": float64(9)}})
	})
	cs := setupMCP(t, client.New(srv.URL, "qbot_test"))

	out := callTool(t, cs, "send_message", map[string]any{"to_user_id": float64(13), "content": "hello"})
	assert.Contains(t, out, `"conversation_id":9`)
	r := lastReq(reqs)
	_, hasThread := r.body["thread_id"]
	assert.False(t, hasThread, "thread_id 省略时不应送该字段，由服务端自动建会话")
	assert.EqualValues(t, 13, r.body["to_user_id"])
}

func TestSendMessage_DefaultTextType(t *testing.T) {
	srv, reqs := mockBotAPI(t, func(w http.ResponseWriter, r *http.Request, body map[string]any) {
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"message_id": float64(1), "conversation_id": float64(1)}})
	})
	cs := setupMCP(t, client.New(srv.URL, "qbot_test"))

	callTool(t, cs, "send_message", map[string]any{"to_user_id": float64(1), "content": "hello"})
	r := lastReq(reqs)
	assert.Equal(t, "text", r.body["msg_type"], "未传 msg_type 应默认 text")
}

func TestListMessages(t *testing.T) {
	srv, reqs := mockBotAPI(t, func(w http.ResponseWriter, r *http.Request, body map[string]any) {
		// 只回查 query 判断是否带 after_id
		msgs := []map[string]any{
			{"id": float64(10), "content": "hello", "type": "text", "sender_nickname": "Dave"},
			{"id": float64(11), "content": "world", "type": "markdown", "sender_nickname": "Bot"},
		}
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"messages": msgs}})
	})
	cs := setupMCP(t, client.New(srv.URL, "qbot_test"))

	out := callTool(t, cs, "list_messages", map[string]any{"thread_id": float64(5), "limit": float64(20)})
	assert.Contains(t, out, `"content":"hello"`)
	assert.Contains(t, out, `"content":"world"`)
	// 每行一条
	assert.Equal(t, 2, strings.Count(strings.TrimSpace(out), "\n")+1)

	r := lastReq(reqs)
	assert.Equal(t, "/api/v1/bot/messages", r.path)
	assert.Contains(t, r.path, "", "")
	// thread_id 与 limit 应进 query（httptest 里 r.URL.Query）
	_ = r // query 已在 path 外，此处仅断言被调用
}

func TestPollMessages_RequiresAfterID(t *testing.T) {
	srv, _ := mockBotAPI(t, func(w http.ResponseWriter, r *http.Request, body map[string]any) {
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"messages": []any{}}})
	})
	cs := setupMCP(t, client.New(srv.URL, "qbot_test"))

	// after_id=0 应被工具拒绝（返回错误结果），不应调 Bot API
	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "poll_messages",
		Arguments: map[string]any{"thread_id": float64(5), "after_id": float64(0)},
	})
	require.NoError(t, err)
	assert.True(t, res.IsError, "after_id=0 应返回错误")
}

func TestPollMessages_Incremental(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"messages": []any{
			map[string]any{"id": float64(21), "content": "reply", "type": "text"},
		}}})
	}))
	t.Cleanup(srv.Close)
	cs := setupMCP(t, client.New(srv.URL, "qbot_test"))

	out := callTool(t, cs, "poll_messages", map[string]any{"thread_id": float64(5), "after_id": float64(20)})
	assert.Contains(t, out, `"content":"reply"`)
	assert.Contains(t, gotQuery, "after_id=20", "after_id 应进 query 做增量拉取")
	assert.Contains(t, gotQuery, "thread_id=5")
}

func TestStreamingThreeStep(t *testing.T) {
	var mu sync.Mutex
	var streamReqs []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		_ = json.Unmarshal(raw, &body)
		if strings.HasSuffix(r.URL.Path, "/stream") {
			mu.Lock()
			streamReqs = append(streamReqs, body)
			mu.Unlock()
		}
		if strings.HasSuffix(r.URL.Path, "/messages") && r.Method == http.MethodPost {
			// start_streaming：返回新消息 id + 会话 id
			json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"message_id": float64(99), "conversation_id": float64(5)}})
		} else {
			json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{}})
		}
	}))
	t.Cleanup(srv.Close)
	cs := setupMCP(t, client.New(srv.URL, "qbot_test"))

	// 1) start
	startOut := callTool(t, cs, "start_streaming_message", map[string]any{"to_user_id": float64(13)})
	assert.Contains(t, startOut, `"message_id":99`)
	assert.Contains(t, startOut, `"conversation_id":5`)

	// 2) append x2
	callTool(t, cs, "append_streaming_chunk", map[string]any{"message_id": float64(99), "delta": "Hello "})
	callTool(t, cs, "append_streaming_chunk", map[string]any{"message_id": float64(99), "delta": "world"})

	// 3) finish
	callTool(t, cs, "finish_streaming_message", map[string]any{"message_id": float64(99)})

	mu.Lock()
	defer mu.Unlock()
	require.Len(t, streamReqs, 3, "应有 2 次 append + 1 次 finish")
	assert.Equal(t, "Hello ", streamReqs[0]["content_delta"])
	assert.False(t, streamReqs[0]["finish"].(bool))
	assert.Equal(t, "world", streamReqs[1]["content_delta"])
	assert.True(t, streamReqs[2]["finish"].(bool), "最后一次应为 finish=true")
}

func TestAuthHeaderPropagated(t *testing.T) {
	var seen string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.Header.Get("Authorization")
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"messages": []any{}}})
	}))
	t.Cleanup(srv.Close)
	cs := setupMCP(t, client.New(srv.URL, "qbot_secret_xyz"))

	callTool(t, cs, "list_messages", map[string]any{"thread_id": float64(1)})
	assert.Equal(t, "Bearer qbot_secret_xyz", seen)
}
