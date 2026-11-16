package client

// 消息渲染截图：system 胶囊 / 富文本（@chip+链接）/ 合并转发卡片 / share 卡片同屏。

import (
	"encoding/json"
	"testing"

	"github.com/egoist/mygo/ui"

	"qim-newclient/internal/qapi"
)

func TestShotMsgRender(t *testing.T) {
	a := &app{active: 0, loggedIn: true}
	a.convs = []conversation{{ID: "1", Name: "产品讨论组", Sub: "(6人)", Preview: "聊天记录", Time: "刚刚", Color: "#3385ff", Kind: "discussion"}}

	sys := messageFromAPI(mustMsg(`{"id":"31","type":"system","content":"张三 加入了群聊","timestamp":1700000000000}`), "7", 0)
	rich := messageFromAPI(mustMsg(`{"id":"32","type":"text","content":"@{mention:5|李四} 这个文档 [需求说明](https://docs.qq.com/doc) 先看下，反馈贴 https://example.com/issue/12 截止周五","sender":{"id":"8","name":"李四"},"timestamp":1700000001000}`), "7", 0)
	mfrich := messageFromAPI(mustMsg(`{"id":"33","type":"text","content":"收到，[链接](https://example.com) 也发一份","sender":{"id":"8","name":"李四"},"timestamp":1700000002000}`), "7", 0)
	self := messageFromAPI(mustMsg(`{"id":"34","type":"text","content":"@{mention:all|所有人} 下午三点评审记得来","sender":{"id":"7","name":"我"},"sender_id":7,"timestamp":1700000003000}`), "7", 0)
	mfraw := `{"version":1,"title":"项目群的聊天记录","messages":[` +
		`{"id":"1","type":"text","content":"在吗","senderName":"张三","timestamp":1700000000000},` +
		`{"id":"2","type":"image","content":"{\"url\":\"/s/a.png\"}","senderName":"李四","timestamp":1700000001000},` +
		`{"id":"3","type":"text","content":"收到","senderName":"王五","timestamp":1700000002000},` +
		`{"id":"4","type":"text","content":"周五前给结论","senderName":"赵六","timestamp":1700000003000}]}`
	mf := messageFromAPI(mustMsg(`{"id":"35","type":"merged_forward","content":`+quoteJSON(mfraw)+`,"sender":{"id":"8","name":"李四"},"timestamp":1700000004000}`), "7", 0)
	shraw := `{"type":"sticky","id":"42","name":"评审纪要","content":"分享了便签: 评审纪要","originalContent":"1. 排期确认\n2. 接口评审通过"}`
	sh := messageFromAPI(mustMsg(`{"id":"36","type":"share","content":`+quoteJSON(shraw)+`,"sender":{"id":"8","name":"李四"},"timestamp":1700000005000}`), "7", 0)

	a.msgs = []message{sys, rich, self, mf, mfrich, sh}

	tt := ui.NewTester(a.view, 1200, 800)
	tt.Frame()
	tt.Frame()
	write(t, tt, "/tmp/qim-msgrender.png")

	// 顶部两条（系统胶囊 + 富文本）单独截一张，避免被 FollowEnd 滚出视口
	a.msgs = []message{sys, rich}
	tt.Frame()
	write(t, tt, "/tmp/qim-msgrender-top.png")
	a.msgs = []message{sys, rich, self, mf, mfrich, sh}

	// 记录弹窗
	a.openMergedRecord(mf.MF)
	tt.Frame()
	write(t, tt, "/tmp/qim-mfrecord.png")
	a.closeMergedRecord()

	// share 原文弹窗
	a.openShareOriginal(sh.SH)
	tt.Frame()
	write(t, tt, "/tmp/qim-shareorig.png")
	a.closeShareOriginal()
}

// mustMsg 把 JSON 字符串解成 qapi.Message（测试辅助）。
func mustMsg(raw string) qapi.Message {
	var m qapi.Message
	_ = json.Unmarshal([]byte(raw), &m)
	return m
}

// quoteJSON 把一段 JSON 原文编码成 JSON 字符串字面量（嵌进消息 content）。
func quoteJSON(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
