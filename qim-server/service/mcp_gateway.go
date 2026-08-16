package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
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
	Name         string   `json:"name"`
	Transport    string   `json:"transport"` // "streamable-http"（MVP 支持）| "stdio"
	URL          string   `json:"url"`       // streamable-http 时的端点
	Token        string   `json:"token"`     // 可选，Authorization: Bearer
	Enabled      bool     `json:"enabled"`
	AllowedTools []string `json:"allowed_tools"` // nil=未配置=全部开放；[]（空数组）=全不选=不开放任何工具
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
	mu        sync.RWMutex // 保护 registered / connClients / sessions 的并发读写
	// syncMu 串行化整次 Sync：re-sync 含「摘除旧工具 -> 网络重连 -> 重新注册」多步，
	// 并发 Sync 会互相踩 registry。热同步改为异步后并发概率上升，故显式串行；
	// mu 仍只保护共享 map，工具 Execute 不持 syncMu 故不被同步阻塞。
	syncMu sync.Mutex
	// registered 记录本次 Sync 已注册的外部工具名，用于 re-sync 去重/覆盖
	registered map[string]bool
	// toolDescriptions 外部 MCP 工具的描述（由远程 tools/list 提供），
	// 供 FriendlyToolLabel 在无内置映射时直接显示工具原始描述。
	toolDescriptions map[string]string
	// toolTitles 外部 MCP 工具的人类可读标题（由远程 Tool.Title 或 Annotations.Title 提供），
	// 优先于 Name 用于 UI 展示。
	toolTitles map[string]string
	// connClients 按连接名缓存 mcp client（供工具 Execute 复用同一 HTTP 连接池）
	connClients map[string]*mcp.Client
	// sessions 按连接名追踪活跃 session，re-sync 时关闭旧 session 防泄漏
	sessions map[string]*mcp.ClientSession
	// groupEnabled 运行时是否把外部 MCP 工具放行给群 @AI。
	groupEnabled bool
	// AllowPrivate 是否允许连接内网/本机地址的外部 MCP Server（由配置 mcp.allow_private 注入）。
	// false 时运行时连接路径校验 URL 拒绝私网/本机地址（SSRF 防护）；true 时仅校验协议 http/https。
	// 测试中直接置 true 以连接本地测试服务器。
	AllowPrivate bool
}

// NewMCPClientGateway 构造网关。配置经 configSvc 读取、工具注册到 registry。
func NewMCPClientGateway(configSvc *SystemConfigService, registry *ai.ToolRegistry) *MCPClientGateway {
	if registry == nil {
		registry = ai.NewToolRegistry(nil)
	}
	return &MCPClientGateway{
		configSvc:        configSvc,
		registry:         registry,
		registered:       make(map[string]bool),
		toolDescriptions: make(map[string]string),
		toolTitles:       make(map[string]string),
		connClients:      make(map[string]*mcp.Client),
		sessions:         make(map[string]*mcp.ClientSession),
	}
}

// Registry 暴露底层注册表，供外部（如群 AI 白名单拼接）使用。
func (g *MCPClientGateway) Registry() *ai.ToolRegistry {
	return g.registry
}

// ListExternalToolNames 返回当前已注册的外部 MCP 工具名（mcp_*），
// 供群 AI 白名单门控放行。
func (g *MCPClientGateway) ListExternalToolNames() []string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	names := make([]string, 0, len(g.registered))
	for n := range g.registered {
		names = append(names, n)
	}
	return names
}

// ToolDescription 返回指定外部 MCP 工具的描述（由远程 tools/list 提供）。
// 无描述时返回空串，调用方应 fallback 到 FriendlyToolLabel 等默认标签。
func (g *MCPClientGateway) ToolDescription(toolName string) string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.toolDescriptions[toolName]
}

// ToolDescriptions 返回所有已注册外部 MCP 工具的描述快照（name → description）。
// 供 NewToolCallFeedback 在工具调用时直接查找，避免逐次加锁。
func (g *MCPClientGateway) ToolDescriptions() map[string]string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	snapshot := make(map[string]string, len(g.toolDescriptions))
	for k, v := range g.toolDescriptions {
		snapshot[k] = v
	}
	return snapshot
}

// ToolTitles 返回所有已注册外部 MCP 工具的标题快照（name → title）。
// 标题优先于描述用于 UI 展示，供 NewToolCallFeedback 使用。
func (g *MCPClientGateway) ToolTitles() map[string]string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	snapshot := make(map[string]string, len(g.toolTitles))
	for k, v := range g.toolTitles {
		snapshot[k] = v
	}
	return snapshot
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
	// 串行化整次同步：re-sync 多步操作（摘除旧工具 -> 网络重连 -> 重新注册），
	// 并发 Sync 会互相踩 registry。syncMu 覆盖全程，mu 仍只保护共享 map。
	g.syncMu.Lock()
	defer g.syncMu.Unlock()

	// 持写锁：摘除上一轮注册的外部工具，保证幂等。
	g.mu.Lock()
	for name := range g.registered {
		g.registry.RemoveTool(name)
	}
	g.registered = make(map[string]bool)
	g.mu.Unlock()

	// 无锁：加载配置、连接外部 MCP Server（网络 IO，不阻塞 ListExternalToolNames 等读取者）
	conns := g.loadConns()
	for i := range conns {
		conn := &conns[i]
		if !conn.Enabled {
			continue
		}
		g.syncConn(conn)
	}
	g.mu.RLock()
	registeredCount := len(g.registered)
	g.mu.RUnlock()
	logger.WithModule("MCPClientGateway").Info("外部 MCP 工具同步完成",
		"registered", registeredCount, "conns", len(conns))
}

func (g *MCPClientGateway) syncConn(conn *MCPConnConfig) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// 关闭同名的旧 session，防止 re-sync 时 HTTP 连接池泄漏
	g.mu.Lock()
	if old, ok := g.sessions[conn.Name]; ok {
		_ = old.Close()
		delete(g.sessions, conn.Name)
	}
	g.mu.Unlock()

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
		_ = session.Close()
		return
	}

	// 持写锁：注册工具到 registry 和 registered，并记录 session
	g.mu.Lock()
	g.sessions[conn.Name] = session

	// 清除该连接的旧工具元数据（防止 re-sync 后已移除工具的描述/标题残留）
	connPrefix := "mcp_" + sanitizeName(conn.Name) + "_"
	for name := range g.registered {
		if strings.HasPrefix(name, connPrefix) {
			delete(g.toolDescriptions, name)
			delete(g.toolTitles, name)
			g.registry.RemoveTool(name)
			delete(g.registered, name)
		}
	}

	allowedSet := make(map[string]bool, len(conn.AllowedTools))
	for _, name := range conn.AllowedTools {
		allowedSet[name] = true
	}
	// 区分 nil（未配置=全部开放）与 []（用户明确全不选=不开放任何工具）。
	// 只有 conn.AllowedTools != nil 时才启用过滤；nil 时所有工具通过（向后兼容默认行为）。
	filtered := conn.AllowedTools != nil
	for _, tool := range result.Tools {
		// 过滤：配置了 allowed_tools 时只注册名单内的工具
		if filtered && !allowedSet[tool.Name] {
			logger.WithModule("MCPClientGateway").Info("跳过未开放的外部 MCP 工具",
				"conn", conn.Name, "tool", tool.Name)
			continue
		}
		schema := schemaToMap(tool.InputSchema)
		// 解析 Annotations：标题优先级 Title > Annotations.Title
		title := tool.Title
		if title == "" && tool.Annotations != nil {
			title = tool.Annotations.Title
		}
		var readOnly, destructive bool
		if tool.Annotations != nil {
			readOnly = tool.Annotations.ReadOnlyHint
			if tool.Annotations.DestructiveHint != nil {
				destructive = *tool.Annotations.DestructiveHint
			}
		}
		extTool := NewExternalMCPTool(ExternalMCPSendMeta{
			ConnName:    conn.Name,
			ToolName:    tool.Name,
			Title:       title,
			Description: tool.Description,
			Schema:      schema,
			ReadOnly:    readOnly,
			Destructive: destructive,
			Session:     session,
		})
		name := extTool.Name()
		if err := g.registry.RegisterTool(extTool); err != nil {
			// 工具名仅大小写不同与既有工具冲突：注册被拒（ToolRegistry 守卫）。
			// 跳过而非覆盖，避免大小写不敏感查找不确定。
			logger.WithModule("MCPClientGateway").Warn("外部 MCP 工具与既有工具大小写冲突，跳过注册",
				"conn", conn.Name, "tool", tool.Name, "localName", name, "error", err)
			continue
		}
		g.registered[name] = true
		// 存储远程描述和标题，供 FriendlyToolLabel 显示
		if tool.Description != "" {
			g.toolDescriptions[name] = tool.Description
		}
		if title != "" {
			g.toolTitles[name] = title
		}
		logger.WithModule("MCPClientGateway").Info("注册外部 MCP 工具",
			"conn", conn.Name, "tool", tool.Name, "localName", name)
	}
	g.mu.Unlock()
}

// connect 建立到外部 MCP Server 的连接会话。streamable-http 走
// StreamableClientTransport；stdio 暂不支持（MVP 以远程 HTTP 验证链路）。
// 若连接配置了 token，通过自定义 RoundTripper 为每次请求注入
// Authorization: Bearer <token>（go-sdk 的 StreamableClientTransport 无
// 顶层 header 字段，需经 HTTPClient 挂 RoundTripper）。
// ValidateExternalURL 校验外部 MCP Server URL：仅允许 http/https，且默认禁止访问本机/内网地址
// （SSRF 防护）。当 allowPrivate 为 true 时，本机/私网地址限制被放宽（用于内网部署场景，请参阅配置 mcp.allow_private），
// 但仍只允许 http/https 协议并强制要求主机名。域名会解析并校验全部结果（fail-closed）：
// 任一解析地址属于禁止地址（loopback/私网/链路本地/组播/未指定）即拒绝。
// 预览路径与运行时连接路径共用本函数。
func ValidateExternalURL(rawURL string, allowPrivate bool) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("URL 格式错误: %v", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("仅支持 http/https 协议，不支持 %s", u.Scheme)
	}

	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("URL 缺少主机名")
	}

	if allowPrivate {
		return nil
	}

	// 禁止 localhost 与常见本地主机名形式（含 net.ParseIP 不处理的 0/0.0/0.0.0、
	// IPv6 全零 ::/0:0:0:0:0:0:0:0 等通配形式）。
	localhostNames := map[string]bool{
		"localhost": true, "127.0.0.1": true, "::1": true,
		"0.0.0.0": true, "0": true, "0.0": true, "0.0.0": true,
		"::": true, "0:0:0:0:0:0:0:0": true,
	}
	if localhostNames[host] {
		return fmt.Errorf("不允许访问本机地址")
	}

	// IP 字面量：直接按地址族校验。IPv4-mapped IPv6（::ffff:127.0.0.1）经
	// IsLoopback/IsPrivate 一并覆盖（To4 后判段）。
	if ip := net.ParseIP(host); ip != nil {
		if isBlockedAddress(ip) {
			return fmt.Errorf("不允许访问内网/私有地址")
		}
		return nil
	}

	// host 是域名：解析后校验全部结果。fail-closed——解析失败或无结果同样拒绝，
	// 防止"校验时解析暂失败→放行→拨号时恢复成功"的绕过窗口。
	// 注意：此处校验与实际拨号是两次独立解析；连接的权威防线在
	// ssrfDialContext（一次解析→校验→直接拨已校验的 IP，见 SSRFProtectedHTTPClient），
	// 此处是同一策略的提前拦截（预览路径只走到这里）。
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("域名 %s 解析失败，不允许访问: %v", host, err)
	}
	if len(ips) == 0 {
		return fmt.Errorf("域名 %s 无解析结果，不允许访问", host)
	}
	for _, r := range ips {
		if isBlockedAddress(r) {
			return fmt.Errorf("域名 %s 解析到禁止地址 %s，不允许访问", host, r.String())
		}
	}
	return nil
}

// isBlockedAddress 判断 IP 是否属于禁止访问的地址：
// loopback、私网、链路本地、组播、未指定（:: / 0.0.0.0）、IPv4 0.0.0.0/8 其余段。
// 0.0.0.0/8 是"本网络"保留段（RFC 5735），Linux 下经 local 表路由到 lo，
// 与 0.0.0.0 等价地可达本机——IsUnspecified 只认 0.0.0.0，故整段单独拦截。
func isBlockedAddress(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
		return true
	}
	if ip4 := ip.To4(); ip4 != nil && ip4[0] == 0 {
		return true
	}
	return false
}

// resolveAndValidateHost 解析主机并校验全部结果（fail-closed）：
// IP 字面量直接校验；域名解析失败/无结果/任一结果属禁止地址一律返回错误。
func resolveAndValidateHost(host string) ([]net.IP, error) {
	if ip := net.ParseIP(host); ip != nil {
		if isBlockedAddress(ip) {
			return nil, fmt.Errorf("不允许访问内网/私有地址 %s", ip.String())
		}
		return []net.IP{ip}, nil
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return nil, fmt.Errorf("解析域名 %s 失败: %v", host, err)
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("域名 %s 无解析结果", host)
	}
	for _, r := range ips {
		if isBlockedAddress(r) {
			return nil, fmt.Errorf("域名 %s 解析到禁止地址 %s", host, r.String())
		}
	}
	return ips, nil
}

// ssrfDialContext 返回自定义 DialContext。allowPrivate=false 时在拨号前解析主机、
// 校验全部结果，并直接拨号到已校验的 IP——校验与实际拨号共用同一份解析结果，
// 消除 DNS rebinding 的 TOCTOU（旧实现校验解析一次、连接由 SDK 再独立解析一次，
// 两次结果不一致即可绕过）。allowPrivate=true 时委托默认拨号（内网部署场景，
// 保持系统正常解析/happy-eyeballs 行为）。
func ssrfDialContext(allowPrivate bool) func(ctx context.Context, network, addr string) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		if allowPrivate {
			return dialer.DialContext(ctx, network, addr)
		}
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}
		ips, err := resolveAndValidateHost(host)
		if err != nil {
			return nil, err
		}
		var lastErr error
		for _, ip := range ips {
			target := net.JoinHostPort(ip.String(), port)
			conn, err := dialer.DialContext(ctx, network, target)
			if err == nil {
				return conn, nil
			}
			lastErr = err
		}
		if lastErr == nil {
			lastErr = fmt.Errorf("无可用连接目标")
		}
		return nil, lastErr
	}
}

// ssrfRedirectPolicy 构造 http.Client.CheckRedirect：对每个重定向目标重新执行 URL 校验。
// http.Client 已把 Location 解析为绝对 URL 后再回调；被拒则停止跟随并返回错误，
// 该出站请求失败（连接降级，工具不可用），不会把内网响应带回。
// 提取为独立函数便于测试（重定向源 server 通常在本机，受 DialContext 拦截无法端到端验证）。
func ssrfRedirectPolicy(allowPrivate bool) func(req *http.Request, via []*http.Request) error {
	return func(req *http.Request, via []*http.Request) error {
		if err := ValidateExternalURL(req.URL.String(), allowPrivate); err != nil {
			return fmt.Errorf("重定向目标被 SSRF 防护拒绝: %w", err)
		}
		return nil
	}
}

// SSRFProtectedHTTPClient 构造外部 MCP 出站 HTTP client。allowPrivate=false 时：
//  1. DialContext 把连接 pin 在校验过的 IP 上（防 DNS rebinding，见 ssrfDialContext）；
//  2. CheckRedirect 对每个重定向目标重新执行 URL 校验（见 ssrfRedirectPolicy）——旧实现
//     跟随重定向，恶意/被攻陷的公网 server 可 302 到 http://169.254.169.254/ 等内网地址，
//     内网响应体经 MCP 协议回流。
//
// 未配置 Proxy（http.Transport 默认 ProxyFromEnvironment 会走系统代理，此时 DialContext
// 拨的是代理地址而非目标地址，SSRF pin 失效且可能误拒本地代理）：外部 MCP 出站一律直连，
// 需要代理/内网访问的场景请使用 allow_private=true。
// token 非空时经 authorizationTransport 注入 Authorization: Bearer 头。
func SSRFProtectedHTTPClient(allowPrivate bool, token string) *http.Client {
	transport := &http.Transport{
		DialContext:           ssrfDialContext(allowPrivate),
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	var rt http.RoundTripper = transport
	if token != "" {
		rt = &authorizationTransport{token: token, inner: transport}
	}
	return &http.Client{
		Transport:     rt,
		CheckRedirect: ssrfRedirectPolicy(allowPrivate),
	}
}

func (g *MCPClientGateway) connect(ctx context.Context, conn *MCPConnConfig) (*mcp.ClientSession, error) {
	var transport mcp.Transport
	switch conn.Transport {
	case "streamable-http", "", "http":
		if conn.URL == "" {
			return nil, fmt.Errorf("streamable-http 连接缺少 url")
		}
		if err := ValidateExternalURL(conn.URL, g.AllowPrivate); err != nil {
			return nil, fmt.Errorf("SSRF 防护拒绝: %w", err)
		}
		streaming := &mcp.StreamableClientTransport{
			Endpoint: conn.URL,
			// 一律走 SSRF 防护 client：allow_private 关闭时 DialContext 把连接 pin 在
			// 已校验 IP 上并复查重定向；token 非空时内部自动注入 Authorization: Bearer。
			HTTPClient: SSRFProtectedHTTPClient(g.AllowPrivate, conn.Token),
		}
		transport = streaming
	case "stdio":
		return nil, fmt.Errorf("stdio transport 暂不支持，请改用 streamable-http")
	default:
		return nil, fmt.Errorf("未知 transport: %s", conn.Transport)
	}

	// 持写锁：访问 connClients 缓存（可能被 Sync 并发写）
	g.mu.Lock()
	client := g.connClients[conn.Name]
	if client == nil {
		client = mcp.NewClient(&mcp.Implementation{Name: "qim-mcp-client", Version: "1.0.0"}, nil)
		g.connClients[conn.Name] = client
	}
	g.mu.Unlock()

	// 网络连接不持锁，避免阻塞其他读取者
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return nil, fmt.Errorf("mcp connect %s: %w", conn.URL, err)
	}
	return session, nil
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
