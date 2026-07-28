// Package tools 把 QIM Bot API 的操作封装为标准 MCP 工具。
// agent（Claude Code/Cursor 等）经 MCP 调用这些工具即可在 QIM 内收发消息，
// 无需手搓轮询脚本。工具清单与 cmd/qim CLI 操作面对齐。
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dshmyz/qim/qim-server/cmd/qim-mcp/internal/client"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Adapter 持有 Bot API 客户端，方法作为 MCP tool handler。
type Adapter struct {
	api *client.BotAPIClient
}

// New 创建 Adapter。
func New(api *client.BotAPIClient) *Adapter { return &Adapter{api: api} }

// Register 在 MCP server 上注册全部工具。
func Register(s *mcp.Server, a *Adapter) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_messages",
		Description: "列出指定 QIM 会话（thread）的最近消息。用于首次进入会话时读取历史。返回每行一条 JSON 消息。",
	}, a.listMessages)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "poll_messages",
		Description: "增量拉取指定会话中 after_id 之后的新消息。用于感知用户回复，建议轮询时传入上次拿到的最大消息 id。返回每行一条 JSON 消息。",
	}, a.pollMessages)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "send_message",
		Description: "向指定用户发送一条消息（thread_id 省略时自动建/复用会话）。msg_type 可选 text|markdown|card（默认 text）；card 时 content 为按钮卡片 JSON。返回 {message_id, conversation_id}。后续 list/poll 用 conversation_id 作 thread_id。\n\n何时用 card：当需要用户在几个明确选项中做决策（确认/取消、批准/拒绝、选择方案）时，优先用 card 而非让用户用文字回复。card 的 content 形如 {\"title\":\"标题\",\"text\":\"说明\",\"buttons\":[{\"id\":\"confirm\",\"text\":\"确认\"},{\"id\":\"cancel\",\"text\":\"取消\"}]}，每个 button 需 id 和 text，可选 style(value=\"primary\")。用户点击后你会通过 poll/list 收到 type=card_action 的消息，含 action_id 字段标识点了哪个按钮。",
	}, a.sendMessage)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "start_streaming_message",
		Description: "在指定用户会话中创建一条流式消息（typing 占位），返回 {message_id, conversation_id}。随后用 append_streaming_chunk 追加内容，最后用 finish_streaming_message 收尾。thread_id 省略时自动建/复用会话。",
	}, a.startStreaming)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "append_streaming_chunk",
		Description: "向 start_streaming_message 返回的流式消息追加一段内容增量（delta）。可多次调用。",
	}, a.appendChunk)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "finish_streaming_message",
		Description: "收尾一条流式消息，将其转为最终 markdown 渲染。调用后该消息不再接受追加。",
	}, a.finishStreaming)
}

// --- 参数类型（字段 json tag 由 SDK 反射生成 input schema）---

type listMessagesParams struct {
	ThreadID uint64 `json:"thread_id"`
	Limit    int    `json:"limit,omitempty"`
}

type pollMessagesParams struct {
	ThreadID uint64 `json:"thread_id"`
	AfterID  uint64 `json:"after_id"`
}

type sendMessageParams struct {
	ToUserID uint64 `json:"to_user_id"`
	ThreadID uint64 `json:"thread_id,omitempty"`
	Content  string `json:"content"`
	MsgType  string `json:"msg_type,omitempty"`
}

type startStreamingParams struct {
	ToUserID uint64 `json:"to_user_id"`
	ThreadID uint64 `json:"thread_id,omitempty"`
}

type appendChunkParams struct {
	MessageID uint64 `json:"message_id"`
	Delta     string `json:"delta"`
}

type finishStreamingParams struct {
	MessageID uint64 `json:"message_id"`
}

// --- handlers ---

func (a *Adapter) listMessages(ctx context.Context, req *mcp.CallToolRequest, p listMessagesParams) (*mcp.CallToolResult, any, error) {
	msgs, err := a.api.ListMessages(p.ThreadID, 0, p.Limit)
	if err != nil {
		return nil, nil, fmt.Errorf("list_messages 失败: %w", err)
	}
	return textResult(messagesToLines(msgs)), nil, nil
}

func (a *Adapter) pollMessages(ctx context.Context, req *mcp.CallToolRequest, p pollMessagesParams) (*mcp.CallToolResult, any, error) {
	if p.AfterID == 0 {
		return nil, nil, fmt.Errorf("poll_messages 需传入 after_id（上次最大消息 id）")
	}
	msgs, err := a.api.ListMessages(p.ThreadID, p.AfterID, 200)
	if err != nil {
		return nil, nil, fmt.Errorf("poll_messages 失败: %w", err)
	}
	return textResult(messagesToLines(msgs)), nil, nil
}

func (a *Adapter) sendMessage(ctx context.Context, req *mcp.CallToolRequest, p sendMessageParams) (*mcp.CallToolResult, any, error) {
	msgType := p.MsgType
	if msgType == "" {
		msgType = "text"
	}
	id, convID, err := a.api.SendMessage(p.ToUserID, p.ThreadID, p.Content, msgType)
	if err != nil {
		return nil, nil, fmt.Errorf("send_message 失败: %w", err)
	}
	return textResult(fmt.Sprintf(`{"message_id":%d,"conversation_id":%d}`, id, convID)), nil, nil
}

func (a *Adapter) startStreaming(ctx context.Context, req *mcp.CallToolRequest, p startStreamingParams) (*mcp.CallToolResult, any, error) {
	id, convID, err := a.api.SendMessage(p.ToUserID, p.ThreadID, "", "streaming")
	if err != nil {
		return nil, nil, fmt.Errorf("start_streaming_message 失败: %w", err)
	}
	return textResult(fmt.Sprintf(`{"message_id":%d,"conversation_id":%d}`, id, convID)), nil, nil
}

func (a *Adapter) appendChunk(ctx context.Context, req *mcp.CallToolRequest, p appendChunkParams) (*mcp.CallToolResult, any, error) {
	if err := a.api.StreamChunk(p.MessageID, p.Delta, false); err != nil {
		return nil, nil, fmt.Errorf("append_streaming_chunk 失败: %w", err)
	}
	return textResult(`{"ok":true}`), nil, nil
}

func (a *Adapter) finishStreaming(ctx context.Context, req *mcp.CallToolRequest, p finishStreamingParams) (*mcp.CallToolResult, any, error) {
	if err := a.api.StreamChunk(p.MessageID, "", true); err != nil {
		return nil, nil, fmt.Errorf("finish_streaming_message 失败: %w", err)
	}
	return textResult(`{"ok":true}`), nil, nil
}

// --- helpers ---

func textResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}
}

// messagesToLines 把消息切片序列化为每行一条 JSON（agent 易解析）。
func messagesToLines(msgs []client.Message) string {
	if len(msgs) == 0 {
		return ""
	}
	var b strings.Builder
	for i := range msgs {
		line, _ := json.Marshal(msgs[i])
		b.Write(line)
		b.WriteByte('\n')
	}
	return b.String()
}
