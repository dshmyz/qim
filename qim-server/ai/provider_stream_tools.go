package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/dshmyz/qim/qim-server/pkg/logger"
)

// streamChatWithToolsOpenAICompat 执行 OpenAI 兼容 /chat/completions 的「带工具 + 流式」请求，
// 供当前走 OpenAI delta 流式格式的 Provider（OpenAI/DeepSeek/Bytedance 等）共用的真·流式
// tool-call 通道。onChunk 收到两类 chunk：
//   - 内容 delta：chunk.Content 为逐 token 文本（final 答案回合用）；
//   - 工具增量：chunk.ToolCalls 为 ToolCallDelta 分片（按 Index 跨 chunk 累积，由调用方
//     回合终了整体 unmarshal）。
//
// 此实现把工具（tools）注入 /chat/completions 请求体并置 Stream:true —— 与普通流式共用
// 同一 OpenAI chat/completions SSE 协议，仅多解析 delta.tool_calls[]。流式.

func streamChatWithToolsOpenAICompat(ctx context.Context, bp *BaseProvider, baseURL, apiKey, model string, extra map[string]interface{}, messages []Message, tools []ToolDef, onChunk func(chunk StreamChunk) error) error {
	maxTokens := 4096
	if v, ok := extra["max_tokens"].(int); ok {
		maxTokens = v
	}
	temperature := 0.7
	if v, ok := extra["temperature"].(float64); ok {
		temperature = v
	}

	reqBody := struct {
		Model       string      `json:"model"`
		Messages    []Message   `json:"messages"`
		MaxTokens   int         `json:"max_tokens,omitempty"`
		Temperature float64     `json:"temperature,omitempty"`
		Stream      bool        `json:"stream"`
		Tools       []toolDefBody `json:"tools,omitempty"`
	}{
		Model:       model,
		Messages:    messages,
		MaxTokens:   maxTokens,
		Temperature: temperature,
		Stream:      true,
		Tools:       buildOpenAICompatTools(tools),
	}

	req, _, err := CreateJSONRequestWithContext(ctx,
		"POST",
		baseURL+"/chat/completions",
		apiKey,
		reqBody,
		nil,
	)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "text/event-stream")

	resp, err := bp.Client.Do(req)
	if err != nil {
		return fmt.Errorf("streaming API request failed: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return fmt.Errorf("streaming API returned non-200 status: %d", resp.StatusCode)
	}

	return bp.ReadSSEStream(resp.Body, func(data string) error {
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content   string `json:"content"`
					ToolCalls []struct {
						Index    int    `json:"index"`
						ID       string `json:"id"`
						Function struct {
							Name      string `json:"name"`
							Arguments string `json:"arguments"`
						} `json:"function"`
					} `json:"tool_calls"`
				} `json:"delta"`
				FinishReason *string `json:"finish_reason"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			logger.WithModule("AI").Debug("skip unmarshal stream data", "error", err)
			return nil
		}
		if len(chunk.Choices) == 0 {
			return nil
		}
		sc := StreamChunk{
			Content: chunk.Choices[0].Delta.Content,
			Finish:  chunk.Choices[0].FinishReason,
		}
		for _, tc := range chunk.Choices[0].Delta.ToolCalls {
			sc.ToolCalls = append(sc.ToolCalls, ToolCallDelta{
				Index:     tc.Index,
				ID:        tc.ID,
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			})
		}
		if sc.Content != "" || sc.Finish != nil || len(sc.ToolCalls) > 0 {
			return onChunk(sc)
		}
		return nil
	})
}

// toolDefBody OpenAI /chat/completions 的 tools 单项：{"type":"function","function":{...}}。
// ToolDef 本身无 type 字段，不能直接序列化为要求的结构，故在此显式包装（与
// OpenAIProvider.ChatWithTools 的 sequence 一致）。
type toolDefBody struct {
	Type     string `json:"type"`
	Function struct {
		Name        string                 `json:"name"`
		Description string                 `json:"description"`
		Parameters  map[string]interface{} `json:"parameters"`
	} `json:"function"`
}

func buildOpenAICompatTools(tools []ToolDef) []toolDefBody {
	if len(tools) == 0 {
		return nil
	}
	out := make([]toolDefBody, 0, len(tools))
	for _, t := range tools {
		var b toolDefBody
		b.Type = "function"
		b.Function.Name = t.Name
		b.Function.Description = t.Description
		b.Function.Parameters = t.Parameters
		out = append(out, b)
	}
	return out
}
