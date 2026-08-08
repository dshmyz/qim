package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/pkg/logger"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// externalMCPConfigKey 外部 MCP Server 连接配置在 system_configs 中的键。
// Value 为 JSON 数组：
//
//	[{"name":"demo","transport":"streamable-http","url":"http://localhost:9100/mcp","token":"","enabled":true}]
const externalMCPConfigKey = "external_mcp"

// externalMCPGroupEnabledKey 是否把已注册的外部 MCP 工具放行给群 @AI 使用。
// 默认不配置 = 关。只有显式开启后，mcp_* 工具才进入群助手可用集。
const externalMCPGroupEnabledKey = "external_mcp:group_enabled"

// MCPConnConfig 一条外部 MCP Server 连接的配置。
type MCPConnConfig struct {
	Name      string `json:"name"`
	Transport string `json:"transport"` // "streamable-http"（MVP 支持）| "stdio"
	URL       string `json:"url"`       // streamable-http 时的端点
	Token     string `json:"token"`     // 可选，Authorization: Bearer
	Enabled   bool   `json:"enabled"`
}

// MCPClientGateway 是 QIM 进程内的 MCP Client 网关：读取后台配置，惰性连接
// 外部 MCP Server，把其 tools/list 拉取注册进 ToolRegistry，供群 @AI 的 ReAct
// 循环作为可达工具调用。
//
// 设计约束：外部 MCP Server 的生命周期/不可达绝不拖垮 QIM 主进程。连接按需建立、
// 失败仅日志并跳过该连接（其工具降级为不可用），不影响群 AI 主路径。
type MCPClientGateway struct {
	configSvc *SystemConfigService
	registry  *ai.ToolRegistry
	// registered 记录本次 Sync 已注册的外部工具名，用于 re-sync 去重/覆盖
	registered map[string]bool
	// connClients 按连接名缓存 mcp client（供工具 Execute 复用同一 HTTP 连接池）
	connClients map[string]*mcp.Client
}

// NewMCPClientGateway 构造网关。配置经 configSvc 读取、工具注册到 registry。
func NewMCPClientGateway(configSvc *SystemConfigService, registry *ai.ToolRegistry) *MCPClientGateway {
	if registry == nil {
		registry = ai.NewToolRegistry(nil)
	}
	return &MCPClientGateway{
		configSvc:   configSvc,
		registry:    registry,
		registered:  make(map[string]bool),
		connClients: make(map[string]*mcp.Client),
	}
}

// Registry 暴露底层注册表，供外部（如群 AI 白名单拼接）使用。
func (g *MCPClientGateway) Registry() *ai.ToolRegistry {
	return g.registry
}

// ListExternalToolNames 返回当前已注册的外部 MCP 工具名（mcp_*），
// 供群 AI 白名单门控放行。
func (g *MCPClientGateway) ListExternalToolNames() []string {
	names := make([]string, 0, len(g.registered))
	for n := range g.registered {
		names = append(names, n)
	}
	return names
}

// GroupEnabled 报告是否已开启「外部 MCP 工具供群 @AI 使用」。
func (g *MCPClientGateway) GroupEnabled() bool {
	cfg, err := g.configSvc.GetConfig(externalMCPGroupEnabledKey)
	if err != nil {
		return false
	}
	return cfg.Value == "true"
}

// loadConns 读取 system_configs 中的外部 MCP 连接配置；缺失/非法返回空列表。
func (g *MCPClientGateway) loadConns() []MCPConnConfig {
	cfg, err := g.configSvc.GetConfig(externalMCPConfigKey)
	if err != nil {
		return nil // 未配置 = 无外部 MCP
	}
	var conns []MCPConnConfig
	if err := json.Unmarshal([]byte(cfg.Value), &conns); err != nil {
		logger.WithModule("MCPClientGateway").Error("解析 external_mcp 配置失败", "error", err)
		return nil
	}
	return conns
}

// Sync 读取配置并同步外部工具注册表。失败不阻塞：单连接错误仅跳过。
// 可安全重复调用：每次先摘除上一轮注册的外部工具（应对连接被删除/禁用/改名），
// 再按当前配置重新注册，因此运行时可经 ReSyncExternalMCP() 热刷新。
func (g *MCPClientGateway) Sync() {
	// 摘除上一轮注册的外部工具，保证幂等：删除的连接对应的工具不再残留。
	for name := range g.registered {
		g.registry.RemoveTool(name)
	}
	g.registered = make(map[string]bool)

	conns := g.loadConns()
	for i := range conns {
		conn := &conns[i]
		if !conn.Enabled {
			continue
		}
		g.syncConn(conn)
	}
	logger.WithModule("MCPClientGateway").Info("外部 MCP 工具同步完成",
		"registered", len(g.registered), "conns", len(conns))
}

func (g *MCPClientGateway) syncConn(conn *MCPConnConfig) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	session, err := g.connect(ctx, conn)
	if err != nil {
		logger.WithModule("MCPClientGateway").Warn("连接外部 MCP 失败，跳过该连接",
			"conn", conn.Name, "error", err)
		return
	}

	result, err := session.ListTools(ctx, nil)
	if err != nil {
		logger.WithModule("MCPClientGateway").Warn("拉取外部 MCP 工具列表失败",
			"conn", conn.Name, "error", err)
		return
	}

	for _, tool := range result.Tools {
		schema := schemaToMap(tool.InputSchema)
		extTool := NewExternalMCPTool(conn.Name, tool.Name, schema, session)
		name := extTool.Name()
		g.registry.RegisterTool(extTool)
		g.registered[name] = true
		logger.WithModule("MCPClientGateway").Info("注册外部 MCP 工具",
			"conn", conn.Name, "tool", tool.Name, "localName", name)
	}
}

// connect 建立到外部 MCP Server 的连接会话。streamable-http 走
// StreamableClientTransport；stdio 暂不支持（MVP 以远程 HTTP 验证链路）。
// 若连接配置了 token，通过自定义 RoundTripper 为每次请求注入
// Authorization: Bearer <token>（go-sdk 的 StreamableClientTransport 无
// 顶层 header 字段，需经 HTTPClient 挂 RoundTripper）。
func (g *MCPClientGateway) connect(ctx context.Context, conn *MCPConnConfig) (*mcp.ClientSession, error) {
	var transport mcp.Transport
	switch conn.Transport {
	case "streamable-http", "", "http":
		if conn.URL == "" {
			return nil, fmt.Errorf("streamable-http 连接缺少 url")
		}
		streaming := &mcp.StreamableClientTransport{Endpoint: conn.URL}
		if conn.Token != "" {
			streaming.HTTPClient = tokenAuthHTTPClient(conn.Token)
		}
		transport = streaming
	case "stdio":
		return nil, fmt.Errorf("stdio transport 暂不支持，请改用 streamable-http")
	default:
		return nil, fmt.Errorf("未知 transport: %s", conn.Transport)
	}

	client := g.connClients[conn.Name]
	if client == nil {
		client = mcp.NewClient(&mcp.Implementation{Name: "qim-mcp-client", Version: "1.0.0"}, nil)
		g.connClients[conn.Name] = client
	}

	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return nil, fmt.Errorf("mcp connect %s: %w", conn.URL, err)
	}
	return session, nil
}

// tokenAuthHTTPClient 返回一个在每次出站请求上注入
// Authorization: Bearer <token> 的 http.Client。用于为需要鉴权的
// 外部 MCP Server 附加凭据（streamable-http 传输）。
// 复用默认传输以保证既有连接池/超时行为；仅在 token 非空时构造。
func tokenAuthHTTPClient(token string) *http.Client {
	return &http.Client{
		Transport: &authorizationTransport{
			token: token,
			inner: http.DefaultTransport,
		},
	}
}

// authorizationTransport 注入 Authorization: Bearer 头的 RoundTripper。
type authorizationTransport struct {
	token string
	inner http.RoundTripper
}

func (t *authorizationTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// 复制请求，避免污染复用连接池中的客户端请求对象
	clone := req.Clone(req.Context())
	clone.Header.Set("Authorization", "Bearer "+t.token)
	return t.inner.RoundTrip(clone)
}

// schemaToMap 把 MCP 工具定义里的 inputSchema（any，通常已是 map）规范化为
// ai.Tool.Parameters 期望的 map[string]interface{}。
func schemaToMap(schema any) map[string]interface{} {
	switch v := schema.(type) {
	case map[string]interface{}:
		return v
	default:
		// 反序列化兜底
		b, err := json.Marshal(schema)
		if err != nil {
			return map[string]interface{}{"type": "object"}
		}
		var m map[string]interface{}
		if err := json.Unmarshal(b, &m); err != nil {
			return map[string]interface{}{"type": "object"}
		}
		return m
	}
}
