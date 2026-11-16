package client

// -probe-c：对真实 qim-server 验证 C 组链路。
// 反馈：POST /feedbacks（纯文字）→ GET /my-feedbacks 回读第一条 →
//   POST 带截图 → 列表条数 +1；
// 版本：GET /client/versions?platform=macos&pageSize=1（公开接口，无记录也过）。

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"qim-newclient/internal/qapi"
)

func runCProbe() int {
	server := strings.TrimSpace(*flagServer)
	user := strings.TrimSpace(*flagUser)
	pass := *flagPass
	if pass == "" {
		pass = os.Getenv("QIM_PASS")
	}
	if server == "" || user == "" || pass == "" {
		fmt.Println("用法: -probe-c -server http://host:port -user <用户名> -pass <密码>")
		return 2
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	c := qapi.New(server)
	fmt.Printf("服务端: %s\n", c.BaseURL())

	if _, err := c.Login(ctx, user, pass); err != nil {
		fmt.Printf("[登录] ✗ %v\n", err)
		return 1
	}
	fmt.Println("[登录] ✓")

	// 1. 反馈：先记基线条数
	base, err := c.MyFeedbacks(ctx, 1, 20)
	if err != nil {
		fmt.Printf("[反馈] ✗ 拉基线失败: %v\n", err)
		return 1
	}
	baseLen := len(base)
	marker := fmt.Sprintf("C组探针反馈-%d", time.Now().Unix())
	if err := c.SubmitFeedback(ctx, "other", marker, ""); err != nil {
		fmt.Printf("[反馈] ✗ 提交失败: %v\n", err)
		return 1
	}
	list, err := c.MyFeedbacks(ctx, 1, 20)
	if err != nil {
		fmt.Printf("[反馈] ✗ 回读失败: %v\n", err)
		return 1
	}
	if len(list) != baseLen+1 {
		fmt.Printf("[反馈] ✗ 条数应 +1: %d -> %d\n", baseLen, len(list))
		return 1
	}
	if list[0].Content != marker || list[0].Type != "other" || list[0].Status != "pending" {
		fmt.Printf("[反馈] ✗ 首条内容不符: %+v\n", list[0])
		return 1
	}
	fmt.Printf("[反馈] ✓ 提交+回读一致（status=%s）\n", list[0].Status)

	// 2. 带截图提交
	shot := filepath.Join(os.TempDir(), fmt.Sprintf("probe-c-%d.png", time.Now().Unix()))
	if err := os.WriteFile(shot, []byte{0x89, 'P', 'N', 'G'}, 0o600); err != nil {
		fmt.Printf("[截图] ⚠ 写临时文件失败，跳过: %v\n", err)
	} else {
		if err := c.SubmitFeedback(ctx, "bug", marker+"-带截图", shot); err != nil {
			fmt.Printf("[截图] ✗ 提交失败: %v\n", err)
			return 1
		}
		list2, err := c.MyFeedbacks(ctx, 1, 20)
		if err != nil || len(list2) != baseLen+2 {
			fmt.Printf("[截图] ✗ 条数应再 +1: err=%v len=%d\n", err, len(list2))
			return 1
		}
		if list2[0].Screenshot == "" {
			fmt.Printf("[截图] ⚠ screenshot 路径为空（存储未配?），提交链路仍通过\n")
		} else {
			fmt.Printf("[截图] ✓ screenshot=%s\n", list2[0].Screenshot)
		}
	}

	// 3. 版本检查（公开接口；无记录也视为通过）
	v, err := c.LatestVersion(ctx, "macos")
	if err != nil {
		fmt.Printf("[版本] ✗ %v\n", err)
		return 1
	}
	if v == nil {
		fmt.Println("[版本] ✓ 公开接口可达，暂无版本记录")
	} else {
		fmt.Printf("[版本] ✓ 最新 v%s 更新说明=%q\n", v.Version, v.UpdateNotes)
	}

	fmt.Println("=== 探针全过 ===")
	return 0
}
