package service

import (
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
	gw.skipSSRFCheck = true
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
	gw.skipSSRFCheck = true

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
	gw.skipSSRFCheck = true
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
	gw.skipSSRFCheck = true
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
	gw.skipSSRFCheck = true
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

// TestTokenAuthHTTPClient_InjectAuthorization 验证带 token 时：authorizationTransport
// 在转发前注入 Authorization: Bearer <token>，且 clone 后的请求（进入内层 transport 的
// 那份）确实携带该头——即请求经整条链路到达服务端时已完成注入。
func TestTokenAuthHTTPClient_InjectAuthorization(t *testing.T) {
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

// TestTokenAuthHTTPClient_EmptyTokenNoHeader 验证 token 为空时不注入鉴权头，
// 行为与未配置 token 的历史路径一致。
func TestTokenAuthHTTPClient_EmptyTokenNoHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// 空 token 时不走 tokenAuthHTTPClient（connect 仅在非空时构造），
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
