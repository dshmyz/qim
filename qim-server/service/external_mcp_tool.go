package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/pkg/logger"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// externalToolPrefix 外部 MCP 工具在本进程 ToolRegistry 里的命名空间前缀。
// 采用前缀是因为：QIM 内置管理工具（group_management 等）用裸名，外部工具可能重名，
// 前缀可避免冲突、且能从名字追溯来源连接。MVP 阶段群 AI 白名单据此按前缀放行。
const externalToolPrefix = "mcp_"

// ExternalMCPTool 把外部 MCP Server 暴露的一个工具适配成进程内 ai.Tool，
// 使群 @AI 的 ReAct 循环能像内置工具一样选择并调用它。
//
// Execute 是唯一跨网络的桥接点：把 ReAct 给出的参数原样转发为远程
// session.CallTool，把远程返回的 content 文本作为结果回喂 LLM（现有
// GetCompletionWithToolsMultiStep 会把 tool 结果 JSON 化后追加进多轮上下文）。
// 远程调用失败返回 error，ReAct 会把错误作为 tool 结果让 LLM 自行决定，
// 而非直接中断（复用现有行为），从而保证外部 MCP 故障不拖垮群 AI 主路径。
type ExternalMCPTool struct {
	// connName 来源连接名（用于同名工具溯源与日志）
	connName string
	// toolName 外部 MCP 工具原始名
	toolName string
	// schema 外部工具的参数 JSON Schema（ai.Tool.Parameters 的 Map 形态）
	schema map[string]interface{}
	// session 指向外部 MCP Server 的已连接客户端会话；nil 表示当前不可用（降级态）
	session *mcp.ClientSession
	// callTimeout 单次远程调用超时，防外部 server 挂起拖住回复
	callTimeout time.Duration
}

// NewExternalMCPTool 构造外部工具适配器。toolName 来自远程 tools/list 的 name，
// schema 来自远程工具定义的 inputSchema（已转成 map）。
func NewExternalMCPTool(connName, toolName string, schema map[string]interface{}, session *mcp.ClientSession) *ExternalMCPTool {
	if schema == nil {
		schema = map[string]interface{}{"type": "object"}
	}
	return &ExternalMCPTool{
		connName:    connName,
		toolName:    toolName,
		schema:      schema,
		session:     session,
		callTimeout: 30 * time.Second,
	}
}

// Name 返回带命名空间的工具名：mcp_<conn>_<tool>。
// 与内置工具取名空间隔离，白名单/日志可据此归属来源。
func (t *ExternalMCPTool) Name() string {
	return externalToolPrefix + sanitizeName(t.connName) + "_" + sanitizeName(t.toolName)
}

func sanitizeName(s string) string {
	b := strings.Builder{}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return b.String()
}

func (t *ExternalMCPTool) Description() string {
	return fmt.Sprintf("调用外部 MCP 服务「%s」提供的工具 %s。", t.connName, t.toolName)
}

func (t *ExternalMCPTool) Parameters() map[string]interface{} {
	return t.schema
}

// Execute 把参数代理到远程 MCP Server 的 CallTool。
func (t *ExternalMCPTool) Execute(params map[string]interface{}, ctx *ai.CallerContext) (interface{}, error) {
	if t.session == nil {
		return nil, fmt.Errorf("外部 MCP 工具 %s 不可用：未连接到「%s」", t.toolName, t.connName)
	}

	ctx2, cancel := context.WithTimeout(context.Background(), t.callTimeout)
	defer cancel()

	args := map[string]any{}
	for k, v := range params {
		args[k] = v
	}

	res, err := t.session.CallTool(ctx2, &mcp.CallToolParams{
		Name:      t.toolName,
		Arguments: args,
	})
	if err != nil {
		logger.WithModule("ExternalMCP").Error("外部 MCP 工具调用失败",
			"conn", t.connName, "tool", t.toolName, "error", err)
		return nil, fmt.Errorf("mcp tool %s(%s) error: %w", t.toolName, t.connName, err)
	}

	if res.IsError {
		return nil, fmt.Errorf("外部 MCP 工具 %s 返回错误", t.toolName)
	}

	texts := make([]string, 0, len(res.Content))
	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok && tc.Text != "" {
			texts = append(texts, tc.Text)
		}
	}
	if len(texts) == 0 {
		// 非文本内容（如图片/结构化）：退化返回有用提示而非丢弃
		if res.StructuredContent != nil {
			if b, err := json.Marshal(res.StructuredContent); err == nil {
				return string(b), nil
			}
		}
		return "", nil
	}
	return strings.Join(texts, "\n"), nil
}

// SetSession 更新会话（连接重建/失效时由网关调用）。nil 表示降级为不可用。
func (t *ExternalMCPTool) SetSession(session *mcp.ClientSession) {
	t.session = session
}
