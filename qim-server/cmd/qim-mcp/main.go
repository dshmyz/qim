// Command qim-mcp 是 QIM Bot 的标准 MCP（Model Context Protocol）server。
//
// 支持两种 transport：
//
//	stdio（默认，本地 Claude Code / Cursor）：
//	  qim-mcp --token qbot_...
//
//	streamable HTTP（远程部署，任意 MCP 客户端）：
//	  qim-mcp --transport http --addr :8082 --server http://localhost:8080
//
// HTTP 模式为 stateless（无会话持久化），token 经 Authorization: Bearer 头逐请求传入。
// 不耦合 server 内部，与 cmd/qim CLI 同源。
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/dshmyz/qim/qim-server/cmd/qim-mcp/internal/client"
	"github.com/dshmyz/qim/qim-server/cmd/qim-mcp/internal/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	serverURL := flag.String("server", "http://localhost:8080", "QIM server URL")
	token := flag.String("token", "", "Bot 访问令牌（qbot_ 开头）。stdio 模式必填，http 模式由请求 Authorization 头传入")
	userToken := flag.String("user-token", "", "用户 JWT（可选）。用于任务管理、日历事件、消息搜索等需要用户身份的接口")
	transport := flag.String("transport", "stdio", "传输模式：stdio（本地）或 http（远程 StreamableHTTP）")
	addr := flag.String("addr", ":8082", "HTTP 监听地址（仅 --transport http 有效）")
	flag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	switch *transport {
	case "stdio":
		runStdio(ctx, *serverURL, *token, *userToken)
	case "http":
		runHTTP(ctx, *serverURL, *addr)
	default:
		fmt.Fprintf(os.Stderr, "未知 transport: %s（支持 stdio / http）\n", *transport)
		os.Exit(2)
	}
}

func runStdio(ctx context.Context, serverURL, token, userToken string) {
	if token == "" {
		fmt.Fprintln(os.Stderr, "--token 必填：在 QIM 客户端 BotConfigDialog 中签发 Bot 令牌后填入")
		os.Exit(2)
	}

	api := client.New(serverURL, token, userToken)
	adapter := tools.New(api)

	s := mcp.NewServer(&mcp.Implementation{Name: "qim-bot", Version: "1.0.0"}, nil)
	tools.Register(s, adapter)

	if err := s.Run(ctx, &mcp.StdioTransport{}); err != nil {
		fmt.Fprintf(os.Stderr, "qim-mcp stdio 退出: %v\n", err)
		os.Exit(1)
	}
}

func runHTTP(ctx context.Context, serverURL, addr string) {
	handler := mcp.NewStreamableHTTPHandler(
		func(r *http.Request) *mcp.Server {
			token := extractBearerToken(r)
			if token == "" {
				return nil // 400 Bad Request
			}
			userToken := extractUserToken(r)
			api := client.New(serverURL, token, userToken)
			adapter := tools.New(api)
			s := mcp.NewServer(&mcp.Implementation{Name: "qim-bot", Version: "1.0.0"}, nil)
			tools.Register(s, adapter)
			return s
		},
		&mcp.StreamableHTTPOptions{Stateless: true},
	)

	srv := &http.Server{Addr: addr, Handler: handler}

	go func() {
		<-ctx.Done()
		srv.Close()
	}()

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "监听失败: %v\n", err)
		os.Exit(1)
	}
	slog.Info("qim-mcp HTTP 启动", "addr", ln.Addr().String())
	if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
		fmt.Fprintf(os.Stderr, "qim-mcp HTTP 退出: %v\n", err)
		os.Exit(1)
	}
}

func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return ""
}

// extractUserToken 从 X-QIM-User-Token 头提取用户 JWT（可选）。
func extractUserToken(r *http.Request) string {
	return r.Header.Get("X-QIM-User-Token")
}
