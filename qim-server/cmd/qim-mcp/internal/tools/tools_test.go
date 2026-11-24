package tools_test

import (
	"context"
	"encoding/json"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/dshmyz/qim/qim-server/cmd/qim-mcp/internal/client"
	"github.com/dshmyz/qim/qim-server/cmd/qim-mcp/internal/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// capturedRequest 记录一次 API 请求。
type capturedRequest struct {
	method      string
	path        string
	query       url.Values
	auth        string
	contentType string
	body        map[string]any
	rawBody     []byte
}

// reqLog 是并发安全的请求记录器。
type reqLog struct {
	mu   sync.Mutex
	reqs []capturedRequest
}

func (l *reqLog) add(r capturedRequest) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.reqs = append(l.reqs, r)
}

func (l *reqLog) last() capturedRequest {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.reqs[len(l.reqs)-1]
}

func (l *reqLog) all() []capturedRequest {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]capturedRequest, len(l.reqs))
	copy(out, l.reqs)
	return out
}

// mockBotAPI 起一个可编程的 API mock，记录所有请求。
func mockBotAPI(t *testing.T, handle func(w http.ResponseWriter, r *http.Request, body map[string]any)) (*httptest.Server, *reqLog) {
	t.Helper()
	log := &reqLog{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		if len(raw) > 0 && strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			_ = json.Unmarshal(raw, &body)
		}
		log.add(capturedRequest{
			method:      r.Method,
			path:        r.URL.Path,
			query:       r.URL.Query(),
			auth:        r.Header.Get("Authorization"),
			contentType: r.Header.Get("Content-Type"),
			body:        body,
			rawBody:     raw,
		})
		handle(w, r, body)
	}))
	t.Cleanup(srv.Close)
	return srv, log
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

func listTools(t *testing.T, cs *mcp.ClientSession) []*mcp.Tool {
	t.Helper()
	res, err := cs.ListTools(context.Background(), &mcp.ListToolsParams{})
	require.NoError(t, err)
	return res.Tools
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

func callToolErr(t *testing.T, cs *mcp.ClientSession, name string, args map[string]any) string {
	t.Helper()
	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	require.NoError(t, err)
	require.True(t, res.IsError, "期望 tool 返回错误")
	require.NotEmpty(t, res.Content)
	tc, ok := res.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	return tc.Text
}

// --- 发消息 ---

func TestSendMessage(t *testing.T) {
	srv, log := mockBotAPI(t, func(w http.ResponseWriter, r *http.Request, body map[string]any) {
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"message_id": float64(42), "conversation_id": float64(7)}})
	})
	cs := setupMCP(t, client.New(srv.URL, "qbot_test", ""))

	out := callTool(t, cs, "send_message", map[string]any{
		"to": "13", "content": "hi", "type": "markdown",
	})
	assert.Contains(t, out, `"message_id":42`)
	assert.Contains(t, out, `"conversation_id":7`)

	r := log.last()
	assert.Equal(t, "Bearer qbot_test", r.auth, "token 应透传为 Bearer")
	assert.Equal(t, "/api/v1/bot/messages", r.path)
	assert.Equal(t, "hi", r.body["content"])
	assert.Equal(t, "markdown", r.body["msg_type"])
	assert.EqualValues(t, 13, r.body["to_user_id"])
}

func TestSendMessage_ConversationPriority(t *testing.T) {
	srv, log := mockBotAPI(t, func(w http.ResponseWriter, r *http.Request, body map[string]any) {
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"message_id": float64(42), "conversation_id": float64(7)}})
	})
	cs := setupMCP(t, client.New(srv.URL, "qbot_test", ""))

	callTool(t, cs, "send_message", map[string]any{"to": "13", "conversation": "7", "content": "hi"})
	r := log.last()
	assert.EqualValues(t, 7, r.body["conversation_id"], "conversation 应优先于 to")
	_, hasTo := r.body["to_user_id"]
	assert.False(t, hasTo, "会话发送不应带 to_user_id")
}

func TestSendMessage_ByName(t *testing.T) {
	srv, log := mockBotAPI(t, func(w http.ResponseWriter, r *http.Request, body map[string]any) {
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"message_id": float64(1), "conversation_id": float64(9)}})
	})
	cs := setupMCP(t, client.New(srv.URL, "qbot_test", ""))

	callTool(t, cs, "send_message", map[string]any{"to": "alice", "content": "hi"})
	r := log.last()
	assert.Equal(t, "alice", r.body["to_user_name"], "非数字 to 应走 to_user_name")
	_, hasID := r.body["to_user_id"]
	assert.False(t, hasID)
}

func TestSendMessage_ConversationByName(t *testing.T) {
	srv, log := mockBotAPI(t, func(w http.ResponseWriter, r *http.Request, body map[string]any) {
		if r.URL.Path == "/api/v1/bot/conversations" {
			json.NewEncoder(w).Encode(map[string]any{"data": []any{
				map[string]any{"conversation_id": float64(98), "conv_type": "group", "name": "产品群"},
			}})
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"message_id": float64(1), "conversation_id": float64(98)}})
	})
	cs := setupMCP(t, client.New(srv.URL, "qbot_test", ""))

	callTool(t, cs, "send_message", map[string]any{"conversation": "产品群", "content": "hi"})
	reqs := log.all()
	require.Len(t, reqs, 2)
	assert.Equal(t, "/api/v1/bot/conversations", reqs[0].path)
	assert.Equal(t, "/api/v1/bot/messages", reqs[1].path)
	assert.EqualValues(t, 98, reqs[1].body["conversation_id"], "名称应解析为会话 ID")
}

func TestSendMessage_ReplyTo(t *testing.T) {
	srv, log := mockBotAPI(t, func(w http.ResponseWriter, r *http.Request, body map[string]any) {
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"message_id": float64(1), "conversation_id": float64(1)}})
	})
	cs := setupMCP(t, client.New(srv.URL, "qbot_test", ""))

	callTool(t, cs, "send_message", map[string]any{"to": "13", "content": "hi", "reply_to": float64(99)})
	assert.EqualValues(t, 99, log.last().body["reply_to_id"], "reply_to 应映射为 Bot API 的 reply_to_id")
}

func TestSendMessage_DefaultTextType(t *testing.T) {
	srv, log := mockBotAPI(t, func(w http.ResponseWriter, r *http.Request, body map[string]any) {
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"message_id": float64(1), "conversation_id": float64(1)}})
	})
	cs := setupMCP(t, client.New(srv.URL, "qbot_test", ""))

	callTool(t, cs, "send_message", map[string]any{"to": "1", "content": "hello"})
	assert.Equal(t, "text", log.last().body["msg_type"], "未传 type 应默认 text")
}

func TestSendMessage_NoThreadAutoCreate(t *testing.T) {
	srv, log := mockBotAPI(t, func(w http.ResponseWriter, r *http.Request, body map[string]any) {
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"message_id": float64(1), "conversation_id": float64(9)}})
	})
	cs := setupMCP(t, client.New(srv.URL, "qbot_test", ""))

	callTool(t, cs, "send_message", map[string]any{"to": "13", "content": "hello"})
	_, hasThread := log.last().body["thread_id"]
	assert.False(t, hasThread, "thread 省略时不应送该字段，由服务端自动建会话")
}

func TestSendMessage_ByConversation(t *testing.T) {
	srv, log := mockBotAPI(t, func(w http.ResponseWriter, r *http.Request, body map[string]any) {
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"message_id": float64(55), "conversation_id": float64(5)}})
	})
	cs := setupMCP(t, client.New(srv.URL, "qbot_test", ""))

	out := callTool(t, cs, "send_message", map[string]any{"conversation": "5", "content": "群公告"})
	assert.Contains(t, out, `"conversation_id":5`)
	r := log.last()
	assert.EqualValues(t, 5, r.body["conversation_id"])
	_, hasToUser := r.body["to_user_id"]
	assert.False(t, hasToUser, "conversation 路径不应送 to_user_id")
}

func TestSendMessage_AsUser(t *testing.T) {
	srv, log := mockBotAPI(t, func(w http.ResponseWriter, r *http.Request, body map[string]any) {
		switch {
		case r.URL.Path == "/api/v1/conversations":
			json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"id": float64(77)}})
		case strings.HasSuffix(r.URL.Path, "/messages"):
			json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"id": float64(88)}})
		default:
			json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{}})
		}
	})
	cs := setupMCP(t, client.New(srv.URL, "qbot_test", "qusr_test"))

	out := callTool(t, cs, "send_message", map[string]any{
		"to": "13", "content": "hi", "as_user": true, "reply_to": float64(5),
	})
	assert.Contains(t, out, `"message_id":88`)
	assert.Contains(t, out, `"conversation_id":77`)

	reqs := log.all()
	require.Len(t, reqs, 2)
	assert.Equal(t, "/api/v1/conversations", reqs[0].path)
	assert.EqualValues(t, 13, reqs[0].body["user_id"], "数字 to 应直接作为 user_id")
	assert.Equal(t, "Bearer qusr_test", reqs[0].auth, "as_user 应使用用户令牌")
	assert.Equal(t, "/api/v1/conversations/77/messages", reqs[1].path)
	assert.Equal(t, "hi", reqs[1].body["content"])
	assert.EqualValues(t, 5, reqs[1].body["quoted_message_id"], "reply_to 应映射为 quoted_message_id")
	assert.Equal(t, "Bearer qusr_test", reqs[1].auth)
}

func TestSendMessage_AsUserByName(t *testing.T) {
	srv, log := mockBotAPI(t, func(w http.ResponseWriter, r *http.Request, body map[string]any) {
		switch {
		case r.URL.Path == "/api/v1/users/search":
			json.NewEncoder(w).Encode(map[string]any{"data": []any{
				map[string]any{"id": float64(13), "username": "alice"},
			}})
		case r.URL.Path == "/api/v1/conversations":
			json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"id": float64(77)}})
		default:
			json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"id": float64(88)}})
		}
	})
	cs := setupMCP(t, client.New(srv.URL, "qbot_test", "qusr_test"))

	callTool(t, cs, "send_message", map[string]any{"to": "alice", "content": "hi", "as_user": true})
	reqs := log.all()
	require.Len(t, reqs, 3)
	assert.Equal(t, "/api/v1/users/search", reqs[0].path)
	assert.Equal(t, "alice", reqs[0].query.Get("q"))
	assert.EqualValues(t, 13, reqs[1].body["user_id"])
}

func TestSendMessage_File(t *testing.T) {
	srv, log := mockBotAPI(t, func(w http.ResponseWriter, r *http.Request, body map[string]any) {
		switch {
		case r.URL.Path == "/api/v1/upload":
			json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{
				"url": "uploads/2026/11/report.pdf", "name": "report.pdf",
			}})
		default:
			json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"message_id": float64(3), "conversation_id": float64(4)}})
		}
	})
	cs := setupMCP(t, client.New(srv.URL, "qbot_test", "qusr_test"))

	path := t.TempDir() + "/report.pdf"
	require.NoError(t, writeFile(path, "pdf-bytes"))

	out := callTool(t, cs, "send_message", map[string]any{"to": "13", "content": "见附件", "file": path})
	assert.Contains(t, out, `"message_id":3`)

	reqs := log.all()
	require.Len(t, reqs, 2)
	up := reqs[0]
	assert.Equal(t, "/api/v1/upload", up.path)
	assert.Equal(t, "Bearer qusr_test", up.auth, "上传应使用用户令牌")
	assert.Contains(t, up.contentType, "multipart/form-data")
	assertMultipartFile(t, up, "file", "report.pdf", "pdf-bytes")

	send := reqs[1]
	content, _ := send.body["content"].(string)
	assert.Contains(t, content, "见附件")
	assert.Contains(t, content, "📎 [report.pdf](uploads/2026/11/report.pdf)")
	assert.Equal(t, "markdown", send.body["msg_type"], "带文件时 text 应升级为 markdown")
}

func TestSendMessage_FileMissingUserToken(t *testing.T) {
	srv, _ := mockBotAPI(t, func(w http.ResponseWriter, r *http.Request, body map[string]any) {
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{}})
	})
	cs := setupMCP(t, client.New(srv.URL, "qbot_test", ""))

	path := t.TempDir() + "/a.txt"
	require.NoError(t, writeFile(path, "x"))
	msg := callToolErr(t, cs, "send_message", map[string]any{"to": "1", "file": path})
	assert.Contains(t, msg, "userToken")
}

// --- 读消息 ---

func TestListMessages(t *testing.T) {
	srv, log := mockBotAPI(t, func(w http.ResponseWriter, r *http.Request, body map[string]any) {
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"messages": []any{
			map[string]any{"id": float64(10), "content": "hello", "type": "text", "sender_nickname": "Dave"},
			map[string]any{"id": float64(11), "content": "world", "type": "markdown", "sender_nickname": "Bot"},
		}}})
	})
	cs := setupMCP(t, client.New(srv.URL, "qbot_test", ""))

	out := callTool(t, cs, "list_messages", map[string]any{"thread": "5", "limit": float64(20)})
	assert.Contains(t, out, `"content":"hello"`)
	assert.Contains(t, out, `"content":"world"`)
	assert.Equal(t, 2, strings.Count(strings.TrimSpace(out), "\n")+1)

	r := log.last()
	assert.Equal(t, "/api/v1/bot/messages", r.path)
	assert.Equal(t, "5", r.query.Get("thread_id"))
	assert.Equal(t, "20", r.query.Get("limit"))
}

func TestListMessages_ByName(t *testing.T) {
	srv, log := mockBotAPI(t, func(w http.ResponseWriter, r *http.Request, body map[string]any) {
		if r.URL.Path == "/api/v1/bot/conversations" {
			json.NewEncoder(w).Encode(map[string]any{"data": []any{
				map[string]any{"conversation_id": float64(98), "conv_type": "group", "name": "产品群"},
			}})
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"messages": []any{}}})
	})
	cs := setupMCP(t, client.New(srv.URL, "qbot_test", ""))

	callTool(t, cs, "list_messages", map[string]any{"conversation": "产品群"})
	assert.Equal(t, "98", log.last().query.Get("thread_id"), "名称应解析为会话 ID")
}

func TestListMessages_ThreadAlias(t *testing.T) {
	srv, log := mockBotAPI(t, func(w http.ResponseWriter, r *http.Request, body map[string]any) {
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"messages": []any{}}})
	})
	cs := setupMCP(t, client.New(srv.URL, "qbot_test", ""))

	callTool(t, cs, "list_messages", map[string]any{"thread": "5"})
	assert.Equal(t, "5", log.last().query.Get("thread_id"), "thread 应作为弃用别名继续可用")
}

func TestListMessages_UsernameFallback(t *testing.T) {
	srv, log := mockBotAPI(t, func(w http.ResponseWriter, r *http.Request, body map[string]any) {
		if r.URL.Path == "/api/v1/bot/conversations" {
			json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"messages": []any{}}})
	})
	cs := setupMCP(t, client.New(srv.URL, "qbot_test", ""))

	callTool(t, cs, "list_messages", map[string]any{"conversation": "alice"})
	assert.Equal(t, "alice", log.last().query.Get("thread_name"), "会话名匹配不到时应回退服务端 thread_name")
}

func TestSendMessage_ThreadOnlyUsernameFallback(t *testing.T) {
	srv, log := mockBotAPI(t, func(w http.ResponseWriter, r *http.Request, body map[string]any) {
		if r.URL.Path == "/api/v1/bot/conversations" {
			json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"message_id": float64(1), "conversation_id": float64(9)}})
	})
	cs := setupMCP(t, client.New(srv.URL, "qbot_test", ""))

	callTool(t, cs, "send_message", map[string]any{"thread": "alice", "content": "hi"})
	assert.Equal(t, "alice", log.last().body["to_user_name"], "thread-only 名称应回退为 to_user_name")
}

func TestListMessages_TypeFilter(t *testing.T) {
	srv, _ := mockBotAPI(t, func(w http.ResponseWriter, r *http.Request, body map[string]any) {
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"messages": []any{
			map[string]any{"id": float64(1), "content": "a", "type": "text"},
			map[string]any{"id": float64(2), "content": "b", "type": "card_action"},
		}}})
	})
	cs := setupMCP(t, client.New(srv.URL, "qbot_test", ""))

	out := callTool(t, cs, "list_messages", map[string]any{"thread": "5", "type": "card_action"})
	assert.Contains(t, out, `"content":"b"`)
	assert.NotContains(t, out, `"content":"a"`)
}

func TestPollMessages_RequiresAfterID(t *testing.T) {
	srv, _ := mockBotAPI(t, func(w http.ResponseWriter, r *http.Request, body map[string]any) {
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"messages": []any{}}})
	})
	cs := setupMCP(t, client.New(srv.URL, "qbot_test", ""))

	callToolErr(t, cs, "poll_messages", map[string]any{"thread": "5", "after_id": float64(0)})
}

func TestPollMessages_Incremental(t *testing.T) {
	srv, log := mockBotAPI(t, func(w http.ResponseWriter, r *http.Request, body map[string]any) {
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"messages": []any{
			map[string]any{"id": float64(21), "content": "reply", "type": "text"},
		}}})
	})
	cs := setupMCP(t, client.New(srv.URL, "qbot_test", ""))

	out := callTool(t, cs, "poll_messages", map[string]any{"thread": "5", "after_id": float64(20)})
	assert.Contains(t, out, `"content":"reply"`)
	assert.Contains(t, out, `"after_id":21`, "应返回本批最大消息 id 作为新水位线")
	assert.Equal(t, "20", log.last().query.Get("after_id"))
	assert.Equal(t, "5", log.last().query.Get("thread_id"))
}

func TestPollMessages_TypeFilterKeepsWatermark(t *testing.T) {
	srv, _ := mockBotAPI(t, func(w http.ResponseWriter, r *http.Request, body map[string]any) {
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"messages": []any{
			map[string]any{"id": float64(30), "content": "noise", "type": "text"},
			map[string]any{"id": float64(31), "content": "click", "type": "card_action"},
		}}})
	})
	cs := setupMCP(t, client.New(srv.URL, "qbot_test", ""))

	out := callTool(t, cs, "poll_messages", map[string]any{"thread": "5", "after_id": float64(20), "type": "card_action"})
	assert.Contains(t, out, `"after_id":31`, "水位线应按整批推进，不能被过滤拖住")
	assert.Contains(t, out, `"content":"click"`)
	assert.NotContains(t, out, `"content":"noise"`)
}

func TestListActions(t *testing.T) {
	srv, log := mockBotAPI(t, func(w http.ResponseWriter, r *http.Request, body map[string]any) {
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"messages": []any{
			map[string]any{"id": float64(1), "content": "a", "type": "text"},
			map[string]any{"id": float64(2), "content": "clicked", "type": "card_action", "action_id": "confirm"},
		}}})
	})
	cs := setupMCP(t, client.New(srv.URL, "qbot_test", ""))

	out := callTool(t, cs, "list_actions", map[string]any{"thread": "5"})
	assert.Contains(t, out, `"content":"clicked"`)
	assert.NotContains(t, out, `"content":"a"`)
	assert.Equal(t, "5", log.last().query.Get("thread_id"))
}

// --- 流式 ---

func TestStreamingStartAndStream(t *testing.T) {
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
			json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"message_id": float64(99), "conversation_id": float64(5)}})
		} else {
			json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{}})
		}
	}))
	t.Cleanup(srv.Close)
	cs := setupMCP(t, client.New(srv.URL, "qbot_test", ""))

	startOut := callTool(t, cs, "start_streaming_message", map[string]any{"to": "13"})
	assert.Contains(t, startOut, `"message_id":99`)
	assert.Contains(t, startOut, `"conversation_id":5`)

	callTool(t, cs, "stream_message", map[string]any{"message_id": float64(99), "delta": "Hello "})
	callTool(t, cs, "stream_message", map[string]any{"message_id": float64(99), "delta": "world", "finish": true})

	mu.Lock()
	defer mu.Unlock()
	require.Len(t, streamReqs, 2, "应有 1 次 delta + 1 次 finish")
	assert.Equal(t, "Hello ", streamReqs[0]["content_delta"])
	assert.False(t, streamReqs[0]["finish"].(bool))
	assert.Equal(t, "world", streamReqs[1]["content_delta"])
	assert.True(t, streamReqs[1]["finish"].(bool), "finish=true 应透传")
}

func TestStreamAbort(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &got)
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{}})
	}))
	t.Cleanup(srv.Close)
	cs := setupMCP(t, client.New(srv.URL, "qbot_test", ""))

	callTool(t, cs, "stream_message", map[string]any{"message_id": float64(99), "abort": true})
	assert.Equal(t, true, got["abort"])
}

func TestStreamRequiresSomething(t *testing.T) {
	srv, _ := mockBotAPI(t, func(w http.ResponseWriter, r *http.Request, body map[string]any) {
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{}})
	})
	cs := setupMCP(t, client.New(srv.URL, "qbot_test", ""))
	callToolErr(t, cs, "stream_message", map[string]any{"message_id": float64(1)})
}

// --- 鉴权 / 会话发现 ---

func TestAuthHeaderPropagated(t *testing.T) {
	var seen string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.Header.Get("Authorization")
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"messages": []any{}}})
	}))
	t.Cleanup(srv.Close)
	cs := setupMCP(t, client.New(srv.URL, "qbot_secret_xyz", ""))

	callTool(t, cs, "list_messages", map[string]any{"thread": "1"})
	assert.Equal(t, "Bearer qbot_secret_xyz", seen)
}

func TestListGroupConversations(t *testing.T) {
	srv, log := mockBotAPI(t, func(w http.ResponseWriter, r *http.Request, body map[string]any) {
		json.NewEncoder(w).Encode(map[string]any{"data": []any{
			map[string]any{"conversation_id": float64(5), "group_name": "项目组A"},
			map[string]any{"conversation_id": float64(9), "group_name": "运维群"},
		}})
	})
	cs := setupMCP(t, client.New(srv.URL, "qbot_test", ""))

	out := callTool(t, cs, "list_group_conversations", map[string]any{})
	assert.Contains(t, out, `"conversation_id":5`)
	assert.Contains(t, out, "项目组A")

	r := log.last()
	assert.Equal(t, "GET", r.method)
	assert.Equal(t, "/api/v1/bot/groups", r.path)
}

func TestListBotConversations(t *testing.T) {
	srv, log := mockBotAPI(t, func(w http.ResponseWriter, r *http.Request, body map[string]any) {
		json.NewEncoder(w).Encode(map[string]any{"data": []any{
			map[string]any{"conversation_id": float64(5), "conv_type": "group", "name": "项目组A"},
			map[string]any{"conversation_id": float64(42), "conv_type": "bot", "name": "爱丽丝"},
		}})
	})
	cs := setupMCP(t, client.New(srv.URL, "qbot_test", ""))

	out := callTool(t, cs, "list_bot_conversations", map[string]any{})
	assert.Contains(t, out, `"conversation_id":5`)
	assert.Contains(t, out, `"conversation_id":42`)
	assert.Equal(t, "/api/v1/bot/conversations", log.last().path)
}

func TestListConversations_User(t *testing.T) {
	srv, log := mockBotAPI(t, func(w http.ResponseWriter, r *http.Request, body map[string]any) {
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"list": []any{
			map[string]any{"id": float64(5), "type": "group", "name": "项目组A", "unread_count": float64(3)},
			map[string]any{"id": float64(42), "type": "single", "other_member_name": "爱丽丝",
				"last_message": map[string]any{"content": "在吗"}},
		}}})
	})
	cs := setupMCP(t, client.New(srv.URL, "qbot_test", "qusr_test"))

	out := callTool(t, cs, "list_conversations", map[string]any{"limit": float64(30)})
	assert.Contains(t, out, `"conversation_id":5`)
	assert.Contains(t, out, "项目组A")
	assert.Contains(t, out, `"name":"爱丽丝"`, "单聊应回退到 other_member_name")
	assert.Contains(t, out, "在吗")

	r := log.last()
	assert.Equal(t, "/api/v1/conversations", r.path)
	assert.Equal(t, "30", r.query.Get("page_size"))
	assert.Equal(t, "Bearer qusr_test", r.auth)
}

func TestListConversations_MissingUserToken(t *testing.T) {
	srv, _ := mockBotAPI(t, func(w http.ResponseWriter, r *http.Request, body map[string]any) {
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{}})
	})
	cs := setupMCP(t, client.New(srv.URL, "qbot_test", ""))
	msg := callToolErr(t, cs, "list_conversations", map[string]any{})
	assert.Contains(t, msg, "userToken")
}

// --- 身份 ---

func TestWhoami(t *testing.T) {
	srv, log := mockBotAPI(t, func(w http.ResponseWriter, r *http.Request, body map[string]any) {
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{
			"id": float64(14), "username": "e2e", "nickname": "E2E",
		}})
	})
	cs := setupMCP(t, client.New(srv.URL, "qbot_test", "qusr_test"))

	out := callTool(t, cs, "whoami", map[string]any{})
	assert.Contains(t, out, `"username":"e2e"`)
	assert.Equal(t, "/api/v1/users/me", log.last().path)
	assert.Equal(t, "Bearer qusr_test", log.last().auth)
}

func TestGetBotIdentity(t *testing.T) {
	srv, log := mockBotAPI(t, func(w http.ResponseWriter, r *http.Request, body map[string]any) {
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{
			"id": float64(8), "name": "E2EBot", "type": "assistant", "virtual_user_id": float64(15),
		}})
	})
	cs := setupMCP(t, client.New(srv.URL, "qbot_test", ""))

	out := callTool(t, cs, "get_bot_identity", map[string]any{})
	assert.Contains(t, out, `"virtual_user_id":15`)
	assert.Equal(t, "/api/v1/bot/me", log.last().path)
}

// --- 任务 ---

func TestCreateTask_UsesDueAndDesc(t *testing.T) {
	srv, log := mockBotAPI(t, func(w http.ResponseWriter, r *http.Request, body map[string]any) {
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"id": float64(72)}})
	})
	cs := setupMCP(t, client.New(srv.URL, "qbot_test", "qusr_test"))

	out := callTool(t, cs, "create_task", map[string]any{
		"title": "写周报", "due": "2026-11-25", "priority": "high", "desc": "含进度",
	})
	assert.Contains(t, out, `"task_id":72`)
	r := log.last()
	assert.Equal(t, "2026-11-25", r.body["due_date"], "due 应映射为 due_date")
	assert.Equal(t, "含进度", r.body["description"], "desc 应映射为 description")
}

// --- 日历 ---

func TestUpdateEvent_MergePreservesOmittedFields(t *testing.T) {
	srv, log := mockBotAPI(t, func(w http.ResponseWriter, r *http.Request, body map[string]any) {
		if r.Method == http.MethodGet {
			json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{
				"id": float64(7), "title": "旧标题",
				"start": "2026-11-26T16:00:00+08:00", "end": "2026-11-26T17:00:00+08:00",
				"reminder": float64(10), "description": "保留我",
			}})
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{}})
	})
	cs := setupMCP(t, client.New(srv.URL, "qbot_test", "qusr_test"))

	callTool(t, cs, "update_event", map[string]any{"event_id": float64(7), "title": "新标题"})
	reqs := log.all()
	require.Len(t, reqs, 2)
	put := reqs[1]
	assert.Equal(t, "PUT", put.method)
	assert.Equal(t, "新标题", put.body["title"])
	assert.Equal(t, "2026-11-26T16:00:00+08:00", put.body["start"], "未传 start 应保留原值")
	assert.Equal(t, "2026-11-26T17:00:00+08:00", put.body["end"])
	assert.EqualValues(t, 10, put.body["reminder"], "未传 reminder 应保留原值")
	assert.Equal(t, "保留我", put.body["description"])
}

func TestUpdateEvent_ReminderZeroClears(t *testing.T) {
	srv, log := mockBotAPI(t, func(w http.ResponseWriter, r *http.Request, body map[string]any) {
		if r.Method == http.MethodGet {
			json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{
				"id": float64(7), "title": "t",
				"start": "2026-11-26T16:00:00+08:00", "end": "2026-11-26T17:00:00+08:00",
				"reminder": float64(10),
			}})
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{}})
	})
	cs := setupMCP(t, client.New(srv.URL, "qbot_test", "qusr_test"))

	callTool(t, cs, "update_event", map[string]any{"event_id": float64(7), "reminder": float64(0)})
	assert.EqualValues(t, 0, log.last().body["reminder"], "reminder=0 应取消提醒")
}

func TestCreateEvent_LocalTime(t *testing.T) {
	srv, log := mockBotAPI(t, func(w http.ResponseWriter, r *http.Request, body map[string]any) {
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"id": float64(7)}})
	})
	cs := setupMCP(t, client.New(srv.URL, "qbot_test", "qusr_test"))

	callTool(t, cs, "create_event", map[string]any{
		"title": "会议", "start": "2026-11-26 14:00", "end": "2026-11-26 15:00", "reminder": float64(10),
	})
	body := log.last().body
	assert.NotContains(t, body["start"], " ", "start 应转成 RFC3339")
	assert.Contains(t, body["start"], "T14:00:00")
}

// --- 笔记 ---

func TestListNotes_FiltersSticky(t *testing.T) {
	srv, _ := mockBotAPI(t, func(w http.ResponseWriter, r *http.Request, body map[string]any) {
		json.NewEncoder(w).Encode(map[string]any{"data": []any{
			map[string]any{"id": float64(1), "title": "正式", "type": "note"},
			map[string]any{"id": float64(2), "title": "便签", "type": "sticky"},
			map[string]any{"id": float64(3), "title": "无类型"},
		}})
	})
	cs := setupMCP(t, client.New(srv.URL, "qbot_test", "qusr_test"))

	out := callTool(t, cs, "list_notes", map[string]any{})
	assert.Contains(t, out, "正式")
	assert.Contains(t, out, "无类型")
	assert.NotContains(t, out, "便签")
}

func TestUpdateNote_MergesExisting(t *testing.T) {
	srv, log := mockBotAPI(t, func(w http.ResponseWriter, r *http.Request, body map[string]any) {
		if r.Method == http.MethodGet {
			json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{
				"id": float64(14), "title": "旧标题", "content": "旧正文",
			}})
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{}})
	})
	cs := setupMCP(t, client.New(srv.URL, "qbot_test", "qusr_test"))

	callTool(t, cs, "update_note", map[string]any{"note_id": float64(14), "title": "新标题"})
	reqs := log.all()
	require.Len(t, reqs, 2)
	assert.Equal(t, "PUT", reqs[1].method)
	assert.Equal(t, "新标题", reqs[1].body["title"])
	assert.Equal(t, "旧正文", reqs[1].body["content"], "未传 content 不应清空")
}

func TestUpdateNote_RequiresAField(t *testing.T) {
	srv, _ := mockBotAPI(t, func(w http.ResponseWriter, r *http.Request, body map[string]any) {
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{}})
	})
	cs := setupMCP(t, client.New(srv.URL, "qbot_test", "qusr_test"))
	callToolErr(t, cs, "update_note", map[string]any{"note_id": float64(14)})
}

// --- 工具清单 / schema ---

func TestToolListAndSchemas(t *testing.T) {
	srv, _ := mockBotAPI(t, func(w http.ResponseWriter, r *http.Request, body map[string]any) {
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{}})
	})
	cs := setupMCP(t, client.New(srv.URL, "qbot_test", ""))

	byName := map[string]*mcp.Tool{}
	for _, tl := range listTools(t, cs) {
		byName[tl.Name] = tl
	}
	for _, want := range []string{
		"list_messages", "poll_messages", "list_actions",
		"list_group_conversations", "list_bot_conversations", "list_conversations",
		"send_message", "start_streaming_message", "stream_message",
		"edit_message", "search_messages", "whoami", "get_bot_identity",
		"list_tasks", "create_task", "update_task",
		"list_events", "create_event", "update_event",
		"list_notes", "get_note", "create_note", "update_note", "search_notes",
	} {
		assert.True(t, byName[want] != nil, "缺少工具 %s", want)
	}
	for _, gone := range []string{"list_conversations_bot", "append_streaming_chunk", "finish_streaming_message"} {
		assert.Nil(t, byName[gone], "旧工具应已移除: %s", gone)
	}

	updateEvent := schemaOf(t, byName["update_event"])
	assert.NotContains(t, updateEvent["required"], "reminder", "reminder 应为可选")
	updateNote := schemaOf(t, byName["update_note"])
	assert.NotContains(t, updateNote["required"], "title", "title 应为可选")
	assert.NotContains(t, updateNote["required"], "content", "content 应为可选")
}

func schemaOf(t *testing.T, tool *mcp.Tool) map[string]any {
	t.Helper()
	require.NotNil(t, tool)
	raw, err := json.Marshal(tool.InputSchema)
	require.NoError(t, err)
	var schema map[string]any
	require.NoError(t, json.Unmarshal(raw, &schema))
	return schema
}

// --- helpers ---

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o600)
}

// assertMultipartFile 校验 multipart 请求里指定字段的文件名与内容。
func assertMultipartFile(t *testing.T, r capturedRequest, field, wantName, wantBody string) {
	t.Helper()
	_, params, err := mime.ParseMediaType(r.contentType)
	require.NoError(t, err)
	mr := multipart.NewReader(strings.NewReader(string(r.rawBody)), params["boundary"])
	form, err := mr.ReadForm(1 << 20)
	require.NoError(t, err)
	files := form.File[field]
	require.Len(t, files, 1, "缺少 multipart 字段 %s", field)
	assert.Equal(t, wantName, files[0].Filename)
	f, err := files[0].Open()
	require.NoError(t, err)
	defer f.Close()
	b, err := io.ReadAll(f)
	require.NoError(t, err)
	assert.Equal(t, wantBody, string(b))
}
