package handler

import (
	"context"
	"fmt"
	"time"

	"github.com/dshmyz/qim/qim-server/pkg/response"
	"github.com/dshmyz/qim/qim-server/service"

	"github.com/gin-gonic/gin"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type previewMCPToolsReq struct {
	Name      string `json:"name"`
	Transport string `json:"transport"`
	URL       string `json:"url"`
	Token     string `json:"token"`
}

type toolInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// PreviewMCPTools 临时连接外部 MCP Server，拉取 tools/list 返回工具列表供管理后台勾选。
func PreviewMCPTools(c *gin.Context) {
	var req previewMCPToolsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}
	if req.URL == "" {
		response.BadRequest(c, "端点 URL 不能为空")
		return
	}

	transport, err := buildPreviewTransport(&req)
	if err != nil {
		response.BadRequest(c, fmt.Sprintf("传输方式错误: %v", err))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	client := mcp.NewClient(&mcp.Implementation{Name: "qim-admin-preview", Version: "1.0.0"}, nil)
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		response.InternalServerError(c, fmt.Sprintf("连接 MCP Server 失败: %v", err))
		return
	}
	defer session.Close()

	result, err := session.ListTools(ctx, nil)
	if err != nil {
		response.InternalServerError(c, fmt.Sprintf("获取工具列表失败: %v", err))
		return
	}

	tools := make([]toolInfo, 0, len(result.Tools))
	for _, t := range result.Tools {
		desc := t.Description
		if desc == "" {
			desc = t.Name
		}
		tools = append(tools, toolInfo{Name: t.Name, Description: desc})
	}

	response.Success(c, tools)
}

func buildPreviewTransport(req *previewMCPToolsReq) (mcp.Transport, error) {
	transport := req.Transport
	if transport == "" {
		transport = "streamable-http"
	}
	switch transport {
	case "streamable-http", "http":
		if req.URL == "" {
			return nil, fmt.Errorf("streamable-http 传输需要 url")
		}
		streaming := &mcp.StreamableClientTransport{Endpoint: req.URL}
		if req.Token != "" {
			streaming.HTTPClient = service.TokenAuthHTTPClient(req.Token)
		}
		return streaming, nil
	default:
		return nil, fmt.Errorf("不支持的传输方式: %s", transport)
	}
}
