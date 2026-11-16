package client

// P3 验证：会话列表里的公告/置顶/免打扰字段归一化、@ 候选本地成员化、
// 以及群公告条与置顶/免打扰标识在 headless 渲染里确实画得出来。

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"qim-newclient/internal/qapi"
)

func TestP3ConvFromAPIAnnouncement(t *testing.T) {
	apiconvs := []qapi.Conversation{
		{ID: qapi.ID("106"), Type: "group", Announcement: "本周五团建", GroupID: 50},
		{ID: qapi.ID("2"), Type: "single", Announcement: "", GroupID: 0},
	}
	got := convsFromAPI(apiconvs)
	if got[0].Announcement != "本周五团建" {
		t.Errorf("announcement 没填入: %+v", got[0])
	}
	if got[0].GroupID != 50 {
		t.Errorf("groupID 没填入: %+v", got[0])
	}
	if got[1].Announcement != "" {
		t.Errorf("单聊不该有公告: %+v", got[1])
	}
}

func TestP3MembersFromAPI(t *testing.T) {
	var in []qapi.ConvMember
	if err := json.Unmarshal([]byte(`[
		{"user_id":7,"role":"owner","user":{"id":7,"nickname":"我","username":"me"}},
		{"user_id":8,"role":"member","user":{"id":8,"nickname":"张三"}}
	]`), &in); err != nil {
		t.Fatal(err)
	}
	out := membersFromAPI(in)
	if len(out) != 2 || out[1].Name != "张三" || out[0].Role != "owner" {
		t.Fatalf("membersFromAPI 错: %+v", out)
	}
}

func TestP3LocalMentionHits(t *testing.T) {
	a := &app{
		convs: []conversation{
			{ID: "106", Kind: "group", Members: []member{
				{UserID: 7, Name: "我", Role: "owner"},
				{UserID: 8, Name: "张三", Role: "member"},
				{UserID: 9, Name: "李四", Role: "member"},
			}},
			{ID: "2", Kind: "single", Members: nil},
		},
	}
	a.active = 0
	all := a.localMentionHits("")
	if len(all) != 3 {
		t.Fatalf("群会话应返回 3 个候选, 实际 %d: %+v", len(all), all)
	}
	zhang := a.localMentionHits("张")
	if len(zhang) != 1 || zhang[0].Name != "张三" {
		t.Fatalf("按「张」过滤应只命中张三, 实际 %+v", zhang)
	}
	// 单聊没有成员：应回退到网络搜索（返回 nil 让调用方走 SearchUsers）。
	a.active = 1
	if hits := a.localMentionHits(""); hits != nil {
		t.Errorf("单聊无成员时应返回 nil, 实际 %+v", hits)
	}
}

func TestP3RenderShots(t *testing.T) {
	a := &app{
		active:     0,
		loggedIn:   true,
		userName:   "原生客户端测试",
		connStatus: qapi.StatusConnected,
		selfID:     "7",
		convs: []conversation{
			{ID: "106", Name: "项目群", Kind: "group", Preview: "张三: 收到", Time: "21:10",
				Pinned: true, Muted: false,
				Announcement: "本周五下午 3 点项目评审会，请提前准备材料",
				Members:      []member{{UserID: 8, Name: "张三"}, {UserID: 9, Name: "李四"}}},
			{ID: "2", Name: "李四", Kind: "single", Preview: "好的", Time: "20:30", Muted: true},
		},
		msgs: []message{
			{Key: "day", Pill: "2026-11-14"},
			{Key: "m-1", ServerID: "1", Sender: "张三",
				Text: "本周五下午 3 点项目评审会，请提前准备材料", At: "21:08", Stamp: time.Now().UnixMilli()},
		},
	}
	tt := ui.NewTester(a.view, 1050, 703)
	tt.Frame()
	tt.Frame()
	for _, want := range []string{"项目群", "(2人)", "张三: 收到", "本周五下午 3 点项目评审会"} {
		if !tt.HasText(want) {
			t.Errorf("界面缺 %q, texts=%q", want, tt.Texts())
		}
	}
	// 原版公告是 ChatHeader 里的一行内联胶囊，没有独立的黄色「群公告」横幅。
	if tt.HasText("群公告") {
		t.Errorf("顶部不应再有独立群公告横幅, texts=%q", tt.Texts())
	}
	// 置顶靠排序体现、免打扰是铃铛斜杠图标，原版列表里没有这两个文字标签。
	for _, unwanted := range []string{"置顶", "免打扰"} {
		if tt.HasText(unwanted) {
			t.Errorf("会话行不应再有 %q 文字标签, texts=%q", unwanted, tt.Texts())
		}
	}
	write(t, tt, "/tmp/qim-p3-groups.png")
}

// TestP3MemberPanel 验证群会话进入后，右侧信息栏真的渲染出成员列表：
// 标题 "群成员 (N)"、每个成员昵称、以及群主/管理员角色徽标都画得出来。
// 注意 headless 下 ui.List 只渲染可视区行，所以这里放 3 个成员（恰好占满
// 信息栏剩余高度），全部可见即可断言；真实窗口里列表可滚动、多出成员照常显示。
func TestP3MemberPanel(t *testing.T) {
	a := &app{
		active:     0,
		loggedIn:   true,
		userName:   "我",
		connStatus: qapi.StatusConnected,
		selfID:     "7",
		convs: []conversation{
			{ID: "106", Name: "项目群", Kind: "group",
				Members: []member{
					{UserID: 7, Name: "我", Role: "owner", Avatar: "/static/uploads/avatars/me.png"},
					{UserID: 8, Name: "张三", Role: "member", Avatar: "/static/uploads/avatars/zhang.png"},
					{UserID: 9, Name: "李四", Role: "admin"},
				}},
		},
	}
	tt := ui.NewTester(a.view, 1050, 850)
	tt.Frame()
	tt.Frame()
	for _, want := range []string{"群成员 (3)", "我", "张三", "李四", "群主", "管理"} {
		if !tt.HasText(want) {
			t.Errorf("成员面板缺 %q, texts=%q", want, tt.Texts())
		}
	}
	write(t, tt, "/tmp/qim-p3-members.png")
}
