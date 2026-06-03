package ai

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/dshmyz/qim/qim-server/pkg/logger"
)

// MCPTool 定义工具接口
type MCPTool interface {
	Name() string
	Description() string
	Parameters() map[string]interface{}
	Execute(params map[string]interface{}, ctx *CallerContext) (interface{}, error)
}

// CallerContext 调用者上下文（用于权限控制）
type CallerContext struct {
	UserID       uint
	Username     string
	Role         string
	GroupID      uint
	GroupRole    string
	AllowedTools []string // 允许使用的工具名列表，为空则允许全部
}

// MCPServer MCP服务器
type MCPServer struct {
	tools        map[string]MCPTool
	enabledTools map[string]bool // 工具启用状态
	tokens       map[string]string
	mu           sync.RWMutex
	server       *http.Server
	authoriz     bool
}

// NewMCPServer 创建MCP服务器
func NewMCPServer(authoriz bool, aiService *AIService) *MCPServer {
	server := &MCPServer{
		tools:        make(map[string]MCPTool),
		enabledTools: make(map[string]bool),
		tokens:       make(map[string]string),
		authoriz:     authoriz,
	}

	if aiService != nil {
		server.RegisterTool(NewIntelligentTroubleshootingTool(aiService))
		server.RegisterTool(NewCommandGenerationTool(aiService))
		server.RegisterTool(NewLogAnalysisTool(aiService))
		server.RegisterTool(NewIntelligentAlertTool(aiService))
		server.RegisterTool(NewOpsKnowledgeTool(aiService))
	}

	return server
}

// RegisterTool 注册工具
func (s *MCPServer) RegisterTool(tool MCPTool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tools[tool.Name()] = tool
	s.enabledTools[tool.Name()] = true // 默认启用
	logger.WithModule("MCPServer").Info("Registered tool", "tool", tool.Name())
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

	return tool.Execute(params, ctx)
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

	logger.WithModule("MCPServer").Info("MCP server starting", "addr", addr)
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