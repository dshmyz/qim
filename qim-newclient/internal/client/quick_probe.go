package client

// -probe-quick：对真实 qim-server 验证 /quick 快捷回复链路。
// GET /user-settings/quick_replies（未设置 → has_value=false，前端兜底默认短语）→
// PUT 自定义短语 → GET 回读一致 → PUT 空数组还原（前端同样兜底默认）。

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"qim-newclient/internal/qapi"
)

func runQuickProbe() int {
	server := strings.TrimSpace(*flagServer)
	user := strings.TrimSpace(*flagUser)
	pass := *flagPass
	if pass == "" {
		pass = os.Getenv("QIM_PASS")
	}
	if server == "" || user == "" || pass == "" {
		fmt.Println("用法: -probe-quick -server http://host:port -user <用户名> -pass <密码>")
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

	// 1. 初次读取（可能未设置）
	v, err := c.GetUserSetting(ctx, quickRepliesKey)
	if err != nil {
		fmt.Printf("[读取] ✗ %v\n", err)
		return 1
	}
	items := parseQuickReplies(v)
	fmt.Printf("[读取] ✓ has_value=%v -> %d 条（兜底默认=%d 条）\n",
		v.HasValue, len(items), len(defaultQuickReplies))

	// 2. PUT 自定义短语 → 回读一致
	custom := []string{"探针短语甲", "探针短语乙", "探针短语丙"}
	if err := c.PutUserSetting(ctx, quickRepliesKey, custom); err != nil {
		fmt.Printf("[保存] ✗ %v\n", err)
		return 1
	}
	v2, err := c.GetUserSetting(ctx, quickRepliesKey)
	if err != nil {
		fmt.Printf("[回读] ✗ %v\n", err)
		return 1
	}
	got := parseQuickReplies(v2)
	if len(got) != 3 || got[0] != custom[0] || got[2] != custom[2] {
		fmt.Printf("[回读] ✗ 内容不符: %v\n", got)
		return 1
	}
	fmt.Println("[回读] ✓ 自定义短语一致")

	// 3. PUT 空数组还原（has_value=true + 空数组 → parseQuickReplies 仍兜底默认）
	if err := c.PutUserSetting(ctx, quickRepliesKey, []string{}); err != nil {
		fmt.Printf("[还原] ✗ %v\n", err)
		return 1
	}
	v3, err := c.GetUserSetting(ctx, quickRepliesKey)
	if err != nil {
		fmt.Printf("[还原] ✗ 回读失败: %v\n", err)
		return 1
	}
	got3 := parseQuickReplies(v3)
	if len(got3) != len(defaultQuickReplies) {
		fmt.Printf("[还原] ✗ 空数组后应兜底默认: %d 条\n", len(got3))
		return 1
	}
	var raw []string
	_ = json.Unmarshal(v3.Value, &raw)
	if len(raw) != 0 {
		fmt.Printf("[还原] ✗ 服务端应存空数组: %v\n", raw)
		return 1
	}
	fmt.Println("[还原] ✓ 空数组已存，前端兜底默认")

	fmt.Println("=== 探针全过 ===")
	return 0
}
