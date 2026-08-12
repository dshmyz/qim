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

// ExternalMCPSendMeta 批量构造 ExternalMCPTool 所需的元数据，
// 收敛参数避免逐字段传递导致签名膨胀。
type ExternalMCPSendMeta struct {
	ConnName    string
	ToolName    string
	Title       string // 人类可读显示名（来自 Tool.Title 或 Annotations.Title）
	Description string // LLM 用途描述（来自 Tool.Description）
	Schema      map[string]interface{}
	ReadOnly    bool // Annotations.ReadOnlyHint：只读工具不修改环境
	Destructive bool // Annotations.DestructiveHint：破坏性写操作标记
	Session     *mcp.ClientSession
}

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
	// title 人类可读的显示名（来自 MCP Tool.Title 或 Annotations.Title），用于 UI 展示
	title string
	// schema 外部工具的参数 JSON Schema（ai.Tool.Parameters 的 Map 形态）
	schema map[string]interface{}
	// description 远程 MCP Server 提供的工具描述，供 LLM 理解工具用途
	description string
	// readOnly 工具是否只读（来自 MCP Annotations.ReadOnlyHint），true = 不修改环境
	readOnly bool
	// destructive 工具是否有破坏性写操作（来自 MCP Annotations.DestructiveHint）
	destructive bool
	// session 指向外部 MCP Server 的已连接客户端会话；nil 表示当前不可用（降级态）
	session *mcp.ClientSession
	// callTimeout 单次远程调用超时，防外部 server 拖住回复
	callTimeout time.Duration
}

// NewExternalMCPTool 从 ExternalMCPSendMeta 构造外部工具适配器。
func NewExternalMCPTool(meta ExternalMCPSendMeta) *ExternalMCPTool {
	schema := meta.Schema
	if schema == nil {
		schema = map[string]interface{}{"type": "object"}
	}
	return &ExternalMCPTool{
		connName:    meta.ConnName,
		toolName:    meta.ToolName,
		title:       meta.Title,
		schema:      schema,
		description: meta.Description,
		readOnly:    meta.ReadOnly,
		destructive: meta.Destructive,
		session:     meta.Session,
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

// Title 返回工具的人类可读显示名。优先级：Title > Name。
func (t *ExternalMCPTool) Title() string {
	if t.title != "" {
		return t.title
	}
	return t.toolName
}

// Description 返回工具的 LLM 用途描述。优先级：远程 Description > 模板兜底。
func (t *ExternalMCPTool) Description() string {
	if t.description != "" {
		return t.description
	}
	return fmt.Sprintf("调用外部 MCP 服务「%s」提供的工具 %s。", t.connName, t.toolName)
}

func (t *ExternalMCPTool) Parameters() map[string]interface{} {
	return t.schema
}

// ReadOnly 返回工具是否只读（Annotations.ReadOnlyHint）。
func (t *ExternalMCPTool) ReadOnly() bool {
	return t.readOnly
}

// Destructive 返回工具是否有破坏性写操作（Annotations.DestructiveHint）。
func (t *ExternalMCPTool) Destructive() bool {
	return t.destructive
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
		// 透传远程错误文本，让 LLM 理解具体失败原因而非仅看到泛化错误
		errText := extractTexts(res.Content)
		if errText != "" {
			return nil, fmt.Errorf("外部 MCP 工具 %s(%s) 返回错误: %s", t.toolName, t.connName, errText)
		}
		return nil, fmt.Errorf("外部 MCP 工具 %s(%s) 返回错误", t.toolName, t.connName)
	}

	texts := extractTexts(res.Content)
	if texts == "" {
		// 非文本内容（如图片/结构化）：退化返回有用提示而非丢弃
		if res.StructuredContent != nil {
			if b, err := json.Marshal(res.StructuredContent); err == nil {
				return string(b), nil
			}
		}
		return "", nil
	}
	return texts, nil
}

// extractTexts 从 MCP CallToolResult 的 Content 中提取所有文本内容。
func extractTexts(content []mcp.Content) string {
	texts := make([]string, 0, len(content))
	for _, c := range content {
		if tc, ok := c.(*mcp.TextContent); ok && tc.Text != "" {
			texts = append(texts, tc.Text)
		}
	}
	return strings.Join(texts, "\n")
}
