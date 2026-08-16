package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupGatewayTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.SystemConfig{}))
	return db
}

func setGatewayConfig(t *testing.T, db *gorm.DB, key, value string) {
	t.Helper()
	var existing model.SystemConfig
	res := db.Where("config_key = ?", key).First(&existing)
	if res.Error == nil {
		require.NoError(t, db.Model(&existing).Update("value", value).Error)
		return
	}
	cfg := model.SystemConfig{ConfigKey: key, Value: value, Type: "string"}
	require.NoError(t, db.Create(&cfg).Error)
}

// TestMCPClientGateway_SyncRegistersExternalTools 从 in-memory 配置 + 真实
// 进程内 MCP server 验证：Sync 后外部工具（mcp_<conn>_<tool>）进入 ToolRegistry，
// 且 group_enabled 门控能正确放行/隔离。
func TestMCPClientGateway_SyncRegistersExternalTools(t *testing.T) {
	db := setupGatewayTestDB(t)
	configSvc := NewSystemConfigService(db)
	registry := ai.NewToolRegistry(nil)

	ts := startTestMCPServer(t)
	conns := []MCPConnConfig{
		{Name: "demo", Transport: "streamable-http", URL: ts.URL, Enabled: true},
	}
	b, err := json.Marshal(conns)
	require.NoError(t, err)
	setGatewayConfig(t, db, externalMCPConfigKey, string(b))

	gw := NewMCPClientGateway(configSvc, registry)
	gw.AllowPrivate = true
	gw.Sync()

	// 未开 group_enabled，组助手白名单不应放行
	assert.False(t, gw.GroupEnabled(), "默认应关闭，避免外部工具无条件扩权")

	// 工具已注册进 registry
	names := gw.ListExternalToolNames()
	require.Contains(t, names, "mcp_demo_calculator")

	tool, ok := registry.GetTool("mcp_demo_calculator")
	require.True(t, ok, "外部工具应注册进进程内 ToolRegistry")
	assert.Equal(t, "mcp_demo_calculator", tool.Name())
}

// TestMCPClientGateway_GroupEnabledGate 验证开启 group_enabled 位后 GroupEnabled 返回 true。
func TestMCPClientGateway_GroupEnabledGate(t *testing.T) {
	db := setupGatewayTestDB(t)
	configSvc := NewSystemConfigService(db)
	gw := NewMCPClientGateway(configSvc, ai.NewToolRegistry(nil))
	gw.AllowPrivate = true

	assert.False(t, gw.GroupEnabled())
	setGatewayConfig(t, db, externalMCPGroupEnabledKey, "true")
	assert.True(t, gw.GroupEnabled())
}

// TestMCPClientGateway_DisabledConnNotSynced 验证 enabled=false 的连接不注册工具。
func TestMCPClientGateway_DisabledConnNotSynced(t *testing.T) {
	db := setupGatewayTestDB(t)
	configSvc := NewSystemConfigService(db)
	registry := ai.NewToolRegistry(nil)

	ts := startTestMCPServer(t)
	conns := []MCPConnConfig{
		{Name: "demo", Transport: "streamable-http", URL: ts.URL, Enabled: false},
	}
	b, err := json.Marshal(conns)
	require.NoError(t, err)
	setGatewayConfig(t, db, externalMCPConfigKey, string(b))

	gw := NewMCPClientGateway(configSvc, registry)
	gw.AllowPrivate = true
	gw.Sync()

	assert.Empty(t, gw.ListExternalToolNames(), "禁用的连接不应注册外部工具")
}

// TestMCPClientGateway_UnreachableServerDegrades 验证外部 server 不可达时不
// 阻塞、不 panic，仅降级（该连接工具不注册），模拟停掉 demo server 后的行为。
func TestMCPClientGateway_UnreachableServerDegrades(t *testing.T) {
	db := setupGatewayTestDB(t)
	configSvc := NewSystemConfigService(db)
	registry := ai.NewToolRegistry(nil)

	// 指向一个不存在的端口——连接必然失败
	conns := []MCPConnConfig{
		{Name: "demo", Transport: "streamable-http", URL: "http://127.0.0.1:1/mcp", Enabled: true},
	}
	b, err := json.Marshal(conns)
	require.NoError(t, err)
	setGatewayConfig(t, db, externalMCPConfigKey, string(b))

	gw := NewMCPClientGateway(configSvc, registry)
	gw.AllowPrivate = true
	gw.Sync() // 不应 panic / 阻塞

	assert.Empty(t, gw.ListExternalToolNames(), "不可达 server 的工具应降级为不可用")
}

// TestMCPClientGateway_EmptyAllowedToolsBlocksAll
// 全不选时 AllowedTools=[]string{}（非 nil），应不注册任何工具。
func TestMCPClientGateway_EmptyAllowedToolsBlocksAll(t *testing.T) {
	db := setupGatewayTestDB(t)
	configSvc := NewSystemConfigService(db)
	registry := ai.NewToolRegistry(nil)

	ts := startTestMCPServer(t)
	conns := []MCPConnConfig{
		{Name: "demo", Transport: "streamable-http", URL: ts.URL, Enabled: true,
			AllowedTools: []string{}}, // 显式空数组 = 全不选
	}
	b, err := json.Marshal(conns)
	require.NoError(t, err)
	setGatewayConfig(t, db, externalMCPConfigKey, string(b))

	gw := NewMCPClientGateway(configSvc, registry)
	gw.AllowPrivate = true
	gw.Sync()

	assert.Empty(t, gw.ListExternalToolNames(), "全不选时不应注册任何工具")
	assert.False(t, gw.GroupEnabled(), "全不选后 GroupEnabled 应仍为 false")
}

// recordingTransport 记录它转发的最后一个请求，把请求交给 inner 处理。
// 用于断言鉴权 header 是否被注入。
type recordingTransport struct {
	inner http.RoundTripper
	last  *http.Request
}

func (t *recordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.last = req.Clone(req.Context())
	return t.inner.RoundTrip(req)
}

// TestAuthorizationTransport_InjectAuthorization 验证带 token 时：authorizationTransport
// 在转发前注入 Authorization: Bearer <token>，且 clone 后的请求（进入内层 transport 的
// 那份）确实携带该头——即请求经整条链路到达服务端时已完成注入。
func TestAuthorizationTransport_InjectAuthorization(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// recorder 作为 authorizationTransport 的 inner：捕获「注入后、真正发出前」的请求。
	rec := &recordingTransport{inner: http.DefaultTransport}
	auth := &authorizationTransport{token: "secret-token", inner: rec}
	client := &http.Client{Transport: auth}

	req, err := http.NewRequest(http.MethodGet, srv.URL, nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	resp.Body.Close()

	require.NotNil(t, rec.last, "应捕获到注入后的出站请求")
	assert.Equal(t, "Bearer secret-token", rec.last.Header.Get("Authorization"),
		"带 token 时应注入 Authorization: Bearer")
}

// TestAuthorizationTransport_EmptyTokenNoHeader 验证 token 为空时不注入鉴权头，
// 行为与未配置 token 的历史路径一致。
func TestAuthorizationTransport_EmptyTokenNoHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// 空 token 时 SSRFProtectedHTTPClient 不挂 authorizationTransport（见其 token 非空判断），
	// 这里直接验证 authorizationTransport 对空 token 不注入头。
	authRec := &recordingTransport{inner: &authorizationTransport{token: "", inner: http.DefaultTransport}}

	req, err := http.NewRequest(http.MethodGet, srv.URL, nil)
	require.NoError(t, err)
	resp, err := authRec.RoundTrip(req)
	require.NoError(t, err)
	resp.Body.Close()

	require.NotNil(t, authRec.last, "应捕获到出站请求")
	assert.Empty(t, authRec.last.Header.Get("Authorization"),
		"token 为空时不应注入 Authorization 头")
}

// TestSSRFProtectedHTTPClient_InjectAuthorization 端到端验证 SSRFProtectedHTTPClient
// 工厂接线：token 非空时，出站请求确实携带 Authorization: Bearer
// （覆盖 connect() 与预览路径 buildPreviewTransport 共用该工厂的场景）。
func TestSSRFProtectedHTTPClient_InjectAuthorization(t *testing.T) {
	gotAuthCh := make(chan string, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuthCh <- r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := SSRFProtectedHTTPClient(true, "secret-token")
	resp, err := client.Get(srv.URL)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, "Bearer secret-token", <-gotAuthCh, "带 token 时应注入 Authorization: Bearer")
}

// TestSSRFProtectedHTTPClient_NoAuthWithoutToken 验证 token 为空时 SSRFProtectedHTTPClient
// 不注入鉴权头（工厂的 token 非空判断接线正确）。
func TestSSRFProtectedHTTPClient_NoAuthWithoutToken(t *testing.T) {
	gotAuthCh := make(chan string, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuthCh <- r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := SSRFProtectedHTTPClient(true, "")
	resp, err := client.Get(srv.URL)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Empty(t, <-gotAuthCh, "token 为空时不应注入 Authorization 头")
}

// TestSSRFRedirectPolicy_RejectsPrivateTarget 验证 CheckRedirect 对重定向目标重新执行
// SSRF 校验：公网 server 302 到内网/元数据地址时拒绝跟随（内网响应不会回流）；
// allowPrivate=true 时放行。
func TestSSRFRedirectPolicy_RejectsPrivateTarget(t *testing.T) {
	policy := ssrfRedirectPolicy(false)
	req, err := http.NewRequest(http.MethodGet, "http://169.254.169.254/latest/meta-data/", nil)
	require.NoError(t, err)
	err = policy(req, nil)
	require.Error(t, err, "重定向到内网地址应被拒绝")
	assert.Contains(t, err.Error(), "SSRF")

	req2, err := http.NewRequest(http.MethodGet, "http://169.254.169.254/latest/meta-data/", nil)
	require.NoError(t, err)
	require.NoError(t, ssrfRedirectPolicy(true)(req2, nil), "allowPrivate=true 时应放行重定向目标")
}

// TestSSRFDialContext_BlocksBlockedLiteralIP 验证 DialContext 在拨号前直接拒绝
// 内网/本机/未指定字面量 IP（不经网络）：私有、0.0.0.0/8、链路本地、IPv6 未指定。
func TestSSRFDialContext_BlocksBlockedLiteralIP(t *testing.T) {
	ctx := context.Background()
	for _, addr := range []string{"10.0.0.1:80", "0.0.0.1:80", "169.254.169.254:80", "[::]:80"} {
		_, err := ssrfDialContext(false)(ctx, "tcp", addr)
		assert.Error(t, err, "应拒绝拨号 %s", addr)
		assert.Contains(t, err.Error(), "不允许访问", "拨号 %s 的拒绝原因应说明禁止访问", addr)
	}
}

// TestMCPClientGateway_AllowPrivateFalseRejectsLocalURL 验证运行时连接路径在
// allow_private 关闭时拒绝本机地址连接：Sync 不注册工具、不 panic。
func TestMCPClientGateway_AllowPrivateFalseRejectsLocalURL(t *testing.T) {
	db := setupGatewayTestDB(t)
	configSvc := NewSystemConfigService(db)
	registry := ai.NewToolRegistry(nil)

	conns := []MCPConnConfig{
		{Name: "demo", Transport: "streamable-http", URL: "http://127.0.0.1:9100/mcp", Enabled: true},
	}
	b, err := json.Marshal(conns)
	require.NoError(t, err)
	setGatewayConfig(t, db, externalMCPConfigKey, string(b))

	gw := NewMCPClientGateway(configSvc, registry)
	gw.AllowPrivate = false // 默认值：SSRF 防护开启
	gw.Sync()

	assert.Empty(t, gw.ListExternalToolNames(), "本机地址连接应被 SSRF 防护拒绝")
}
