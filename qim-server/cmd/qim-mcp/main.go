// Command qim-mcp 是 QIM Bot 的标准 MCP（Model Context Protocol）server。
//
// 让 Claude Code / Cursor 等支持 MCP 的 agent 即插即用 QIM 消息闭环：
//
//	qim-mcp --server http://localhost:8080 --token qbot_...
//
// 在 agent 的 MCP 配置（如 Claude Code 的 .mcp.json）中注册：
//
//	{
//	  "mcpServers": {
//	    "qim": {
//	      "command": "qim-mcp",
//	      "args": ["--server", "http://localhost:8080", "--token", "qbot_..."]
//	    }
//	  }
//	}
//
// 经 stdio 与 agent 交换 JSON-RPC。底层调用 QIM Bot API（Bearer token 鉴权），
// 与 cmd/qim CLI 同源，不耦合 server 内部。
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/dshmyz/qim/qim-server/cmd/qim-mcp/internal/client"
	"github.com/dshmyz/qim/qim-server/cmd/qim-mcp/internal/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	server := flag.String("server", "http://localhost:8080", "QIM server URL")
	token := flag.String("token", "", "Bot 访问令牌（qbot_ 开头，见客户端 BotConfigDialog 签发）")
	flag.Parse()

	if *token == "" {
		fmt.Fprintln(os.Stderr, "--token 必填：在 QIM 客户端 BotConfigDialog 中签发 Bot 令牌后填入")
		os.Exit(2)
	}

	api := client.New(*server, *token)
	adapter := tools.New(api)

	s := mcp.NewServer(&mcp.Implementation{Name: "qim-bot", Version: "1.0.0"}, nil)
	tools.Register(s, adapter)

	// Ctrl-C 优雅退出
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := s.Run(ctx, &mcp.StdioTransport{}); err != nil {
		fmt.Fprintf(os.Stderr, "qim-mcp 退出: %v\n", err)
		os.Exit(1)
	}
}
