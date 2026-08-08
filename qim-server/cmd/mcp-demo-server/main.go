// Command mcp-demo-server 是一个最小的「外部 MCP Server」示例，用于端到端验证
// QIM 群 @AI 作为 MCP Client 调用外部工具的能力。
//
// 它暴露两个与业务无关的示例工具（可在 QIM 群里经 @AI 触发）：
//
//	get_weather  查询指定城市的天气（返回伪造数据）
//	calculator   计算四则运算表达式（返回计算结果）
//
// 运行（streamable HTTP，默认 :9100/mcp）：
//
//	go run ./cmd/mcp-demo-server
//
// 然后在 QIM 的 system_configs 配置 external_mcp 指向它即可。
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
	"syscall"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// weatherParams get_weather 的入参。go-sdk 会据此自动推断 JSON Schema 并校验。
type weatherParams struct {
	City string `json:"city" jsonschema_description:"城市名，如 上海"`
}

// calcParams calculator 的入参。
type calcParams struct {
	Expr string `json:"expr" jsonschema_description:"四则运算表达式，如 3.5*7"`
}

func main() {
	addr := flag.String("addr", ":9100", "HTTP 监听地址")
	path := flag.String("path", "/mcp", "MCP 端点路径")
	flag.Parse()

	s := mcp.NewServer(&mcp.Implementation{
		Name:    "qim-mcp-demo",
		Version: "0.1.0",
	}, nil)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_weather",
		Description: "查询指定城市的当前天气（演示用伪数据）。",
	}, func(_ context.Context, _ *mcp.CallToolRequest, p weatherParams) (*mcp.CallToolResult, any, error) {
		if p.City == "" {
			return nil, nil, fmt.Errorf("city 不能为空")
		}
		// 演示用人为延迟：拉长工具执行窗口，便于观察前端「思考中」占位与工具卡片进行态。
		time.Sleep(4 * time.Second)
		return textResult(fmt.Sprintf("%s 今天多云，26℃，湿度 65%%，东南风 3 级。", p.City)), nil, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "calculator",
		Description: "计算一个四则运算表达式并返回整数/小数结果。支持 + - * / 与括号。",
	}, func(_ context.Context, _ *mcp.CallToolRequest, p calcParams) (*mcp.CallToolResult, any, error) {
		res, err := evalExpr(p.Expr)
		if err != nil {
			return nil, nil, err
		}
		return textResult(fmt.Sprintf("%s = %v", p.Expr, res)), nil, nil
	})

	handler := mcp.NewStreamableHTTPHandler(
		func(r *http.Request) *mcp.Server { return s },
		&mcp.StreamableHTTPOptions{Stateless: true},
	)

	http.Handle(*path, handler)

	srv := &http.Server{Addr: *addr, Handler: http.DefaultServeMux}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		srv.Shutdown(shutdownCtx)
	}()

	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		slog.Error("监听失败", "err", err)
		os.Exit(1)
	}
	slog.Info("mcp-demo-server 启动", "addr", ln.Addr().String(), "path", *path)
	if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
		slog.Error("serve 失败", "err", err)
		os.Exit(1)
	}
}

func textResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}
}

// evalExpr 极简四则运算求值器（仅演示，不支持复杂语法）。用内置 parser 避免依赖。
func evalExpr(expr string) (float64, error) {
	p := &parser{s: expr}
	v, err := p.parseExpr()
	if err != nil {
		return 0, fmt.Errorf("无法解析表达式 %q: %w", expr, err)
	}
	return v, nil
}

type parser struct {
	s string
	i int
}

func (p *parser) parseExpr() (float64, error) {
	v, err := p.parseTerm()
	if err != nil {
		return 0, err
	}
	for {
		p.skipSpace()
		if p.i >= len(p.s) {
			break
		}
		op := p.s[p.i]
		if op != '+' && op != '-' {
			break
		}
		p.i++
		rhs, err := p.parseTerm()
		if err != nil {
			return 0, err
		}
		if op == '+' {
			v += rhs
		} else {
			v -= rhs
		}
	}
	return v, nil
}

func (p *parser) parseTerm() (float64, error) {
	v, err := p.parseFactor()
	if err != nil {
		return 0, err
	}
	for {
		p.skipSpace()
		if p.i >= len(p.s) {
			break
		}
		op := p.s[p.i]
		if op != '*' && op != '/' {
			break
		}
		p.i++
		rhs, err := p.parseFactor()
		if err != nil {
			return 0, err
		}
		if op == '*' {
			v *= rhs
		} else {
			if rhs == 0 {
				return 0, fmt.Errorf("除数为零")
			}
			v /= rhs
		}
	}
	return v, nil
}

func (p *parser) parseFactor() (float64, error) {
	p.skipSpace()
	if p.i < len(p.s) && p.s[p.i] == '(' {
		p.i++
		v, err := p.parseExpr()
		if err != nil {
			return 0, err
		}
		p.skipSpace()
		if p.i >= len(p.s) || p.s[p.i] != ')' {
			return 0, fmt.Errorf("缺少右括号")
		}
		p.i++
		return v, nil
	}
	start := p.i
	for p.i < len(p.s) && (p.s[p.i] >= '0' && p.s[p.i] <= '9' || p.s[p.i] == '.' || p.s[p.i] == '-') {
		p.i++
	}
	if start == p.i {
		return 0, fmt.Errorf("在位置 %d 处发现意外字符", p.i)
	}
	var v float64
	if _, err := fmt.Sscanf(p.s[start:p.i], "%g", &v); err != nil {
		return 0, err
	}
	return v, nil
}

func (p *parser) skipSpace() {
	for p.i < len(p.s) && (p.s[p.i] == ' ' || p.s[p.i] == '\t') {
		p.i++
	}
}
