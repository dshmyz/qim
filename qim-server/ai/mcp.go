package ai

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// MCPTool 定义工具接口
type MCPTool interface {
	Name() string
	Description() string
	Parameters() map[string]interface{}
	Execute(params map[string]interface{}) (interface{}, error)
}

// CallerContext 调用者上下文（用于权限控制）
type CallerContext struct {
	UserID    uint
	Username  string
	Role      string
	GroupID   uint
	GroupRole string
}

// MCPServer MCP服务器
type MCPServer struct {
	tools      map[string]MCPTool
	enabledTools map[string]bool // 工具启用状态
	tokens     map[string]string
	mu         sync.RWMutex
	server     *http.Server
	authoriz   bool
}

// NewMCPServer 创建MCP服务器
func NewMCPServer(authoriz bool) *MCPServer {
	server := &MCPServer{
		tools:      make(map[string]MCPTool),
		enabledTools: make(map[string]bool),
		tokens:     make(map[string]string),
		authoriz:   authoriz,
	}

	server.RegisterTool(&ServerMonitorTool{})
	server.RegisterTool(&LogAnalyzerTool{})
	server.RegisterTool(&ProcessManagerTool{})
	server.RegisterTool(&NetworkTools{})

	return server
}

// RegisterTool 注册工具
func (s *MCPServer) RegisterTool(tool MCPTool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tools[tool.Name()] = tool
	s.enabledTools[tool.Name()] = true // 默认启用
	log.Printf("Registered tool: %s", tool.Name())
}

// GetTool 获取工具
func (s *MCPServer) GetTool(name string) (MCPTool, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	tool, ok := s.tools[name]
	return tool, ok
}

// ListTools 列出所有工具
func (s *MCPServer) ListTools() []map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tools := make([]map[string]interface{}, 0, len(s.tools))
	for name, tool := range s.tools {
		enabled := true
		if e, ok := s.enabledTools[name]; ok {
			enabled = e
		}
		tools = append(tools, map[string]interface{}{
			"name":        name,
			"description": tool.Description(),
			"parameters":  tool.Parameters(),
			"enabled":     enabled,
		})
	}

	return tools
}

// EnableTool 启用工具
func (s *MCPServer) EnableTool(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tools[name]; !ok {
		return fmt.Errorf("tool not found: %s", name)
	}
	s.enabledTools[name] = true
	return nil
}

// DisableTool 禁用工具
func (s *MCPServer) DisableTool(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tools[name]; !ok {
		return fmt.Errorf("tool not found: %s", name)
	}
	s.enabledTools[name] = false
	return nil
}

// IsToolEnabled 检查工具是否启用
func (s *MCPServer) IsToolEnabled(name string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if e, ok := s.enabledTools[name]; ok {
		return e
	}
	return true // 默认启用
}

// ExecuteTool 执行工具
func (s *MCPServer) ExecuteTool(name string, params map[string]interface{}, ctx *CallerContext) (interface{}, error) {
	tool, ok := s.GetTool(name)
	if !ok {
		return nil, fmt.Errorf("tool not found: %s", name)
	}

	// 如果工具实现了权限检查接口，则传递上下文
	if authorizedTool, ok := tool.(AuthorizedTool); ok {
		return authorizedTool.ExecuteWithAuth(params, ctx)
	}

	return tool.Execute(params)
}

// AuthorizedTool 需要权限控制的工具接口
type AuthorizedTool interface {
	ExecuteWithAuth(params map[string]interface{}, ctx *CallerContext) (interface{}, error)
}

// Start 启动MCP服务器
func (s *MCPServer) Start(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/tools", s.handleListTools)
	mux.HandleFunc("/execute", s.handleExecuteTool)

	s.server = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Printf("MCP server starting on %s", addr)
	return s.server.ListenAndServe()
}

// Stop 停止MCP服务器
func (s *MCPServer) Stop() error {
	if s.server != nil {
		return s.server.Close()
	}
	return nil
}

func (s *MCPServer) handleListTools(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tools := s.ListTools()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tools": tools,
	})
}

func (s *MCPServer) handleExecuteTool(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ToolName string                 `json:"tool_name"`
		Params   map[string]interface{} `json:"parameters"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	result, err := s.ExecuteTool(req.ToolName, req.Params, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"result": result,
	})
}

// ServerMonitorTool 服务器监控工具
type ServerMonitorTool struct{}

func (t *ServerMonitorTool) Name() string { return "server_monitor" }
func (t *ServerMonitorTool) Description() string {
	return "服务器监控工具，用于检查服务器状态"
}
func (t *ServerMonitorTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"ip":   map[string]interface{}{"type": "string", "description": "服务器IP地址", "required": true},
		"port": map[string]interface{}{"type": "integer", "description": "端口号", "required": false, "default": 22},
	}
}
func (t *ServerMonitorTool) Execute(params map[string]interface{}) (interface{}, error) {
	ip, _ := params["ip"].(string)
	port := 22
	if p, ok := params["port"].(float64); ok {
		port = int(p)
	}

	// TODO: 这是示例工具，实际部署时需要接入真实的服务器监控 API
	// 例如：SSH 连接、Prometheus API、云厂商 API 等
	log.Printf("[MCP] ServerMonitorTool called with ip=%s, port=%d (demo mode)", ip, port)

	return map[string]interface{}{
		"ip":      ip,
		"port":    port,
		"status":  "demo",
		"message": "这是示例工具，未接入真实监控系统",
	}, nil
}

// LogAnalyzerTool 日志分析工具
type LogAnalyzerTool struct{}

func (t *LogAnalyzerTool) Name() string { return "log_analyzer" }
func (t *LogAnalyzerTool) Description() string {
	return "日志分析工具，用于分析服务器日志"
}
func (t *LogAnalyzerTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"path":    map[string]interface{}{"type": "string", "description": "日志文件路径", "required": true},
		"pattern": map[string]interface{}{"type": "string", "description": "搜索模式", "required": false},
		"lines":   map[string]interface{}{"type": "integer", "description": "返回行数", "required": false, "default": 100},
	}
}
func (t *LogAnalyzerTool) Execute(params map[string]interface{}) (interface{}, error) {
	path, _ := params["path"].(string)
	pattern, _ := params["pattern"].(string)
	lines := 100
	if l, ok := params["lines"].(float64); ok {
		lines = int(l)
	}

	// TODO: 这是示例工具，实际部署时需要接入真实的日志分析系统
	// 例如：ELK Stack、Loki、本地文件系统读取等
	log.Printf("[MCP] LogAnalyzerTool called with path=%s, pattern=%s (demo mode)", path, pattern)

	return map[string]interface{}{
		"path":    path,
		"pattern": pattern,
		"lines":   lines,
		"content": "[demo] 这是示例日志内容，未接入真实日志系统",
	}, nil
}

// ProcessManagerTool 进程管理工具
type ProcessManagerTool struct{}

func (t *ProcessManagerTool) Name() string { return "process_manager" }
func (t *ProcessManagerTool) Description() string {
	return "进程管理工具，用于管理服务器进程"
}
func (t *ProcessManagerTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"action":  map[string]interface{}{"type": "string", "description": "操作类型: start, stop, restart, status", "required": true},
		"service": map[string]interface{}{"type": "string", "description": "服务名称", "required": true},
		"host":    map[string]interface{}{"type": "string", "description": "目标主机", "required": false, "default": "localhost"},
	}
}
func (t *ProcessManagerTool) Execute(params map[string]interface{}) (interface{}, error) {
	action, _ := params["action"].(string)
	service, _ := params["service"].(string)
	host, _ := params["host"].(string)
	if host == "" {
		host = "localhost"
	}

	// TODO: 这是示例工具，实际部署时需要接入真实的进程管理系统
	// 例如：systemd、supervisord、Docker API、Kubernetes API 等
	log.Printf("[MCP] ProcessManagerTool called with action=%s, service=%s, host=%s (demo mode)", action, service, host)

	return map[string]interface{}{
		"action":  action,
		"service": service,
		"host":    host,
		"status":  "demo",
		"message": "这是示例工具，未接入真实进程管理系统",
	}, nil
}

// NetworkTools 网络工具
type NetworkTools struct{}

func (t *NetworkTools) Name() string { return "network_tools" }
func (t *NetworkTools) Description() string {
	return "网络工具，用于网络诊断"
}
func (t *NetworkTools) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"tool":    map[string]interface{}{"type": "string", "description": "工具类型: ping, traceroute, dig", "required": true},
		"target":  map[string]interface{}{"type": "string", "description": "目标地址", "required": true},
		"options": map[string]interface{}{"type": "object", "description": "额外选项", "required": false},
	}
}
func (t *NetworkTools) Execute(params map[string]interface{}) (interface{}, error) {
	tool, _ := params["tool"].(string)
	target, _ := params["target"].(string)

	// TODO: 这是示例工具，实际部署时需要接入真实的网络诊断工具
	// 例如：执行 ping、traceroute、dig 命令或调用相应 API
	log.Printf("[MCP] NetworkTools called with tool=%s, target=%s (demo mode)", tool, target)

	return map[string]interface{}{
		"tool":    tool,
		"target":  target,
		"result":  "demo",
		"output":  "这是示例工具，未接入真实网络诊断系统",
	}, nil
}
