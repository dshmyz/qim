package client

// markdown 消息渲染截图：标题/列表/引用/围栏代码块/行内样式/链接，
// 对方与自己的气泡各一屏（对齐 qim-client MarkdownMessage）。

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestShotMarkdownMsg(t *testing.T) {
	a := &app{active: 0, loggedIn: true}
	a.convs = []conversation{{ID: "1", Name: "AI 助手", Sub: "机器人", Preview: "需求要点", Time: "刚刚", Color: "#3385ff", Kind: "bot"}}

	fence := "```"
	otherSrc := "# 需求评审要点\n\n" +
		"**结论**：接口对齐 *v2 协议*，~~旧字段~~ 全部下线。\n\n" +
		"## 待办\n\n" +
		"- [x] 时序图确认\n- [ ] 联调排期\n1. 先跑通单聊\n2. 再跑群聊\n\n" +
		"> 引用：文档以 confluence 为准\n\n" +
		"行内代码用 `qapi.ID` 标注，示例：\n\n" +
		fence + "json\n{\"id\": 42, \"ok\": true}\n" + fence + "\n\n" +
		"详情见 [需求文档](https://docs.qq.com/doc)"
	other := messageFromAPI(mustMsg(`{"id":"41","type":"markdown","content":`+quoteJSON(otherSrc)+`,"sender":{"id":"8","name":"AI 助手"},"timestamp":1700000006000}`), "7", 0)
	mineSrc := "**收到**，下午给结论。\n\n" + fence + "sql\nselect * from msg limit 10;\n" + fence
	mine := messageFromAPI(mustMsg(`{"id":"42","type":"markdown","content":`+quoteJSON(mineSrc)+`,"sender":{"id":"7","name":"我"},"sender_id":7,"timestamp":1700000007000}`), "7", 0)

	// 各截一张：高消息 + FollowEnd 会停在底部，窗口拉高保证整条入镜
	a.msgs = []message{other}
	tt := ui.NewTester(a.view, 1200, 1500)
	tt.Frame()
	tt.Frame()
	write(t, tt, "/tmp/qim-md-other.png")

	a.msgs = []message{mine}
	tt.Frame()
	tt.Frame()
	write(t, tt, "/tmp/qim-md-mine.png")
}

// TestShotCodeHighlight 锁定代码块语法高亮的视觉回归：一段 Go 代码同时
// 覆盖关键字 / 注释 / 字符串 / 函数名 / 内置类型 / 数字 / 操作符各类 token，
// 便于日后主题或解析器改动后一眼看出高亮是否退化成单色。
func TestShotCodeHighlight(t *testing.T) {
	a := &app{active: 0, loggedIn: true}
	a.convs = []conversation{{ID: "1", Name: "AI 助手", Sub: "机器人", Time: "刚刚", Kind: "bot"}}

	fence := "```"
	src := "代码示例：\n\n" + fence + "go\n" +
		"package main\n\n" +
		"import \"fmt\"\n\n" +
		"// greet 打印问候语\n" +
		"func greet(name string) string {\n" +
		"    msg := \"hello, \" + name\n" +
		"    fmt.Println(msg)\n" +
		"    return msg\n" +
		"}\n" + fence
	m := messageFromAPI(mustMsg(`{"id":"43","type":"markdown","content":`+quoteJSON(src)+`,"sender":{"id":"8","name":"AI 助手"},"timestamp":1700000008000}`), "7", 0)
	a.msgs = []message{m}
	tt := ui.NewTester(a.view, 900, 620)
	tt.Frame()
	tt.Frame()
	write(t, tt, "/tmp/qim-code-highlight.png")
}
