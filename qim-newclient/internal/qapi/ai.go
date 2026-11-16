package qapi

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// AI 助手端点。原版 qim-client 的悬浮球侧边栏走 /ai/sidebar/stream：
// 独立输入框 + SSE 流式回复，不用普通消息接口，因此不会在会话里留消息。
//
// 注意两个契约细节：
//  1. 非流式端点成功码有 code=0 也有 code=200（历史遗留），do 只认 0，
//     所以这里只用 do 拉 suggested-prompts（它返回 code=0）；其余走流式。
//  2. 流式必须用自己的 http.Client：Client.http 带 30s 总超时，AI 回复
//     超过 30s 会被拦腰截断，而流的生命周期应由 context 控制（停止生成）。
var aiStreamHTTP = &http.Client{}

// AISidebarRequest 是 POST /ai/sidebar/stream 的请求体。
// scope="current" 时带 conversation_id 关联当前会话上下文；
// scope="cross" 为跨会话自由对话，不传 conversation_id。
type AISidebarRequest struct {
	Message        string `json:"message"`
	Scope          string `json:"scope"`
	ConversationID uint   `json:"conversation_id,omitempty"`
}

// AIStreamChunk 是 SSE 帧 data: 后的 JSON（ai.StreamChunk 的客户端子集）。
// 最小闭环只用 Content / Finish / Error；思考过程与工具轨迹字段先预留，
// 后续增强渲染时不用再改传输层。
type AIStreamChunk struct {
	Content          string  `json:"content"`
	ReasoningContent string  `json:"reasoning_content,omitempty"`
	MoveToReasoning  string  `json:"move_to_reasoning,omitempty"`
	Finish           *string `json:"finish,omitempty"`
	Error            *string `json:"error,omitempty"`
}

// AISuggestedPrompts 拉输入框上方的推荐提示（GET /ai/suggested-prompts）。
// 管理员没配置时返回空数组，此时界面应隐藏提示条。
func (c *Client) AISuggestedPrompts(ctx context.Context) ([]string, error) {
	var out struct {
		Prompts []string `json:"prompts"`
	}
	if err := c.do(ctx, http.MethodGet, "/api/v1/ai/suggested-prompts", nil, &out); err != nil {
		return nil, err
	}
	return out.Prompts, nil
}

// AIStreamSidebar 发起流式对话，每收到一帧调一次 onChunk。
// 正常结束（finish 帧）和错误帧都返回 nil——上层从 chunk.Finish/Error 判断；
// 只有网络/解析失败才返回 error。ctx 取消会立刻中断读取（停止生成）。
func (c *Client) AIStreamSidebar(ctx context.Context, req AISidebarRequest, onChunk func(AIStreamChunk)) error {
	if req.Scope == "" {
		req.Scope = "current"
	}
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("序列化 AI 请求: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.base+"/api/v1/ai/sidebar/stream", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("构造 AI 请求: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	httpReq.Header.Set("X-App-Version", c.Version)
	httpReq.Header.Set("X-App-Platform", c.Platform)
	if t, _ := c.Session(); t != "" {
		httpReq.Header.Set("Authorization", "Bearer "+t)
	}

	resp, err := aiStreamHTTP.Do(httpReq)
	if err != nil {
		return fmt.Errorf("AI 请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return aiHTTPError(resp.StatusCode, raw)
	}

	scanner := bufio.NewScanner(resp.Body)
	// 工具轨迹/知识来源可能塞进一帧，默认 64KB 不够。
	scanner.Buffer(make([]byte, 64*1024), 2*1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" || data == "[DONE]" {
			continue
		}
		var chunk AIStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue // 忽略坏帧，不让一帧脏数据打断整段回复
		}
		if onChunk != nil {
			onChunk(chunk)
		}
		if chunk.Finish != nil || chunk.Error != nil {
			return nil
		}
	}
	if err := scanner.Err(); err != nil && ctx.Err() == nil {
		return fmt.Errorf("读取 AI 流: %w", err)
	}
	return nil
}

// aiHTTPError 把 AI 端点的非 2xx 响应拆成业务错误。
// AI 总开关关闭时后端回 403 + code=1003「AI 功能已关闭」，
// 这条文案要原样透到界面，用户才知道去设置页开。
func aiHTTPError(status int, raw []byte) error {
	var env struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if json.Unmarshal(raw, &env) == nil && env.Code != 0 {
		return &APIError{Code: env.Code, Message: env.Message}
	}
	return &APIError{Code: status, Message: strings.TrimSpace(truncate(string(raw), 200))}
}
