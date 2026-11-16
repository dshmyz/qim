package client

// -probe-ai 真服务端 AI 助手探针：不开窗口，验证
// 登录 → 推荐提示 → 侧边栏流式对话 三段链路。
//
// 用法：QIM_PASS='密码' go run . -probe-ai -server http://127.0.0.1:8080 -user native_probe

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"qim-newclient/internal/qapi"
)

var probeAI = flag.Bool("probe-ai", false, "只跑 AI 助手真协议验证（推荐提示+流式对话，不开窗口），跑完退出")

func runAIProbe() int {
	server := strings.TrimSpace(*flagServer)
	user := strings.TrimSpace(*flagUser)
	pass := *flagPass
	if pass == "" {
		pass = os.Getenv("QIM_PASS")
	}
	if server == "" || user == "" || pass == "" {
		fmt.Println("用法: -probe-ai -server http://host:port -user <用户名> -pass <密码>")
		return 2
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	c := qapi.New(server)
	fmt.Printf("服务端: %s\n\n", c.BaseURL())

	fmt.Println("[1/3] 登录")
	if _, err := c.Login(ctx, user, pass); err != nil {
		fmt.Printf("  ✗ 登录: %v\n", err)
		return 1
	}
	fmt.Printf("  ✓ 登录成功 用户=%s\n", user)

	fmt.Println("[2/3] 推荐提示")
	prompts, err := c.AISuggestedPrompts(ctx)
	if err != nil {
		fmt.Printf("  ! 拉提示失败（非致命）: %v\n", err)
	} else {
		fmt.Printf("  ✓ %d 条提示: %v\n", len(prompts), prompts)
	}

	fmt.Println("[3/3] 侧边栏流式对话")
	var reply strings.Builder
	var frames int
	var finished bool
	err = c.AIStreamSidebar(ctx, qapi.AISidebarRequest{
		Message: "请只回复两个字：收到", Scope: "cross",
	}, func(ch qapi.AIStreamChunk) {
		if ch.Content != "" {
			frames++
			reply.WriteString(ch.Content)
		}
		if ch.Finish != nil {
			finished = true
		}
		if ch.Error != nil {
			fmt.Printf("  ✗ AI 返回错误帧: %s\n", *ch.Error)
		}
	})
	if err != nil {
		fmt.Printf("  ✗ 流式请求: %v\n", err)
		return 1
	}
	if reply.Len() == 0 {
		fmt.Println("  ✗ 没收到任何内容帧")
		return 1
	}
	if !finished {
		fmt.Println("  ! 没收到 finish 帧（可能被超时截断）")
	}
	got := reply.String()
	fmt.Printf("  ✓ 收到 %d 帧，回复=%q\n", frames, truncateRunes(got, 80))
	if strings.Contains(got, "[错误：") || strings.Contains(got, "provider error") {
		fmt.Println("  ! 服务端 AI provider 异常：收到的是错误文本而非正常回复。")
		fmt.Println("    协议链路（登录 → SSE → 帧解析）本身已通，需在服务端配好模型 provider。")
		return 0
	}

	fmt.Println("\n结论: AI 助手链路（推荐提示 → SSE 流式回复）真实协议可用。")
	return 0
}
