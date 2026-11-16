package client

import (
	"strings"
	"testing"

	"qim-newclient/internal/qapi"
)

// TestInsertMention 验证信息栏点成员后，输入框插入「@昵称 」并登记后端 token，
// 与键盘打 @ 完全等价：空输入追加、连续追加、末尾已在打 @ 则替换。
func TestInsertMention(t *testing.T) {
	a := &app{active: 0, convs: []conversation{{ID: "1", Kind: "group"}}}
	zhang := member{UserID: 8, Name: "张三"}

	// 空输入 -> 追加 "@张三 "
	a.insertMention(zhang)
	if a.input != "@张三 " {
		t.Errorf("空输入插入, got %q", a.input)
	}
	if tok := a.mentionPicks["@张三"]; tok == "" || !strings.Contains(tok, "@{mention:8") {
		t.Errorf("应登记 @张三 的 token, got %q", tok)
	}

	// 句末已带空格，再插第二个 -> "@张三 @李四 "
	a.insertMention(member{UserID: 9, Name: "李四"})
	if a.input != "@张三 @李四 " {
		t.Errorf("连续插入, got %q", a.input)
	}

	// 末尾正在打 @张 -> 替换成 @张三
	a.input = "看看 @张"
	a.insertMention(zhang)
	if a.input != "看看 @张三 " {
		t.Errorf("替换末尾 @token, got %q", a.input)
	}
}

// TestNewMessageNotify 验证 P4 桌面通知的触发时机：
//   - 非当前会话、且不是自己发的消息 -> 弹通知（标题=会话名，正文=预览）
//   - 当前活跃会话的消息 -> 不弹（你正在看）
//   - 自己发出的消息 -> 不弹
//   - 用户关掉 notifyOnNew -> 不弹
func TestNewMessageNotify(t *testing.T) {
	fn := &fakeNotifier{}
	a := &app{
		active:      0,
		loggedIn:    true,
		notifyOnNew: true,
		notifier:    fn,
		convs: []conversation{
			{ID: "1", Name: "当前会话"},
			{ID: "2", Name: "项目群"},
		},
	}

	// 非活跃会话来消息 -> 通知一次
	a.applyIncoming(qapi.Message{ID: qapi.ID("9001"), ConversationID: qapi.ID("2"), Content: "新需求看下"}, "41")
	if len(fn.calls) != 1 {
		t.Fatalf("非活跃会话应通知一次, got=%d", len(fn.calls))
	}
	if fn.calls[0].title != "项目群" || !strings.Contains(fn.calls[0].body, "新需求看下") {
		t.Errorf("通知内容不对: %+v", fn.calls[0])
	}

	// 当前活跃会话来消息 -> 不通知
	a.applyIncoming(qapi.Message{ID: qapi.ID("9002"), ConversationID: qapi.ID("1"), Content: "活跃会话消息"}, "41")
	if len(fn.calls) != 1 {
		t.Errorf("活跃会话不应通知, calls=%d", len(fn.calls))
	}

	// 自己发的消息 -> 不通知
	a.applyIncoming(qapi.Message{ID: qapi.ID("9003"), ConversationID: qapi.ID("2"), Content: "我发的", SenderID: qapi.ID("41")}, "41")
	if len(fn.calls) != 1 {
		t.Errorf("自己发的消息不应通知, calls=%d", len(fn.calls))
	}

	// 关闭通知 -> 不再通知
	a.notifyOnNew = false
	a.applyIncoming(qapi.Message{ID: qapi.ID("9004"), ConversationID: qapi.ID("2"), Content: "又来一条"}, "41")
	if len(fn.calls) != 1 {
		t.Errorf("关闭通知后不应通知, calls=%d", len(fn.calls))
	}
}
