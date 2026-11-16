package qapi

// P1 第二批的协议测试：@ 提及 token、引用请求体、三个消息操作端点、
// 以及真实线格式里新增的几个字段。

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestP1BMentionCodec(t *testing.T) {
	content := "早上好 " + EncodeMention("41", "张 三") + " 请看下，" + EncodeMentionAll() + " 也同步"
	text, mentions := DecodeMentions(content)
	if strings.Contains(text, "@{mention") {
		t.Fatalf("token 没解码: %q", text)
	}
	if !strings.Contains(text, "@张 三") || !strings.Contains(text, "@所有人") {
		t.Fatalf("解码后名字不对: %q", text)
	}
	if len(mentions) != 2 || mentions[0].Target != "41" || mentions[0].Name != "张 三" || !strings.HasPrefix(mentions[1].Target, "all") {
		t.Fatalf("提及解析错: %+v", mentions)
	}
	if got, _ := DecodeMentions("普通文本 @example.com"); got != "普通文本 @example.com" {
		t.Fatalf("非 token 文本被改写了: %q", got)
	}
	// 老数据可能只有 id 没有名字，不能显示成空白
	if got, _ := DecodeMentions(EncodeMention("42", "")); got != "@用户42" {
		t.Fatalf("缺名字的兜底不对: %q", got)
	}
}

func TestP1BMentionsMe(t *testing.T) {
	m := &Message{Content: "hi " + EncodeMention("41", "小明") + " 看下"}
	if !m.MentionsMe("41") {
		t.Error("按 content token 应该判定为 @我")
	}
	if m.MentionsMe("7") {
		t.Error("无关用户不该被判为 @我")
	}
	all := &Message{Content: EncodeMentionAll() + " 集合"}
	if !all.MentionsMe("7") {
		t.Error("@所有人应该算 @我")
	}
	byIDs := &Message{MentionUserIDs: []ID{"41"}, Content: "纯文本"}
	if !byIDs.MentionsMe("41") {
		t.Error("mention_user_ids 应该能判定 @我")
	}
	byFlag := &Message{IsAtMention: true, Content: "纯文本"}
	if !byFlag.MentionsMe("41") {
		t.Error("is_at_mention 应该能判定 @我")
	}
	if (&Message{Content: "纯文本"}).MentionsMe("41") {
		t.Error("普通消息不该被判为 @我")
	}
}

func TestP1BQuotedTextRequest(t *testing.T) {
	plain := TextRequest("你好")
	if plain.QuotedMessageID != 0 || plain.Content != "你好" {
		t.Fatalf("普通文本请求不该带引用: %+v", plain)
	}
	quoted := TextRequestWithQuote("收到", "896")
	if quoted.QuotedMessageID != 896 {
		t.Fatalf("引用 id 没进请求: %+v", quoted)
	}
	b, _ := json.Marshal(quoted)
	if !strings.Contains(string(b), `"quoted_message_id":896`) {
		t.Fatalf("字段名不对: %s", b)
	}
	// 本地乐观消息没有数字 id，必须退化成普通文本而不是发失败
	if bad := TextRequestWithQuote("收到", "local-3"); bad.QuotedMessageID != 0 {
		t.Fatalf("非数字 id 不该进请求: %+v", bad)
	}
	b, _ = json.Marshal(TextRequest("你好"))
	if strings.Contains(string(b), "quoted_message_id") {
		t.Fatalf("omitempty 失效: %s", b)
	}
}

func TestP1BMessageWireFields(t *testing.T) {
	raw := `{
		"id": 901, "conversation_id": 106, "sender_id": 8, "type":"text",
		"content":"回复一下", "created_at":"2026-11-14T18:00:00+08:00",
		"quoted_message_id": 896,
		"quoted_message": {"id":896,"content":"原文","sender":{"id":8,"nickname":"张三"}},
		"mention_user_ids":[41,7], "is_at_mention":true
	}`
	var m Message
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		t.Fatal(err)
	}
	if m.QuotedMessageID.String() != "896" {
		t.Errorf("quoted_message_id 没解出来: %q", m.QuotedMessageID)
	}
	if m.QuotedMessage == nil || m.QuotedMessage.SenderName() != "张三" || m.QuotedMessage.Preview() != "原文" {
		t.Errorf("quoted_message 没解出来: %+v", m.QuotedMessage)
	}
	if len(m.MentionUserIDs) != 2 || m.MentionUserIDs[0] != "41" || !m.IsAtMention {
		t.Errorf("@ 字段没解出来: %+v %v", m.MentionUserIDs, m.IsAtMention)
	}
	// 后端也常把 quoted_message_id 回成 null，ID 要能吃下
	var nullMsg Message
	if err := json.Unmarshal([]byte(`{"id":1,"quoted_message_id":null}`), &nullMsg); err != nil {
		t.Fatalf("null quoted_message_id 解析失败: %v", err)
	}
	if nullMsg.QuotedMessageID.String() != "" {
		t.Error("null 应该解析成空 ID")
	}
}

func TestP1BMessageActionEndpoints(t *testing.T) {
	type call struct{ method, path string }
	var calls []call
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, call{r.Method, r.URL.Path})
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/recall") {
			// 后端撤回的 data 是 {"message":..., "data":msg} 双层包裹，
			// msg 里带 extra.original_content 供重新编辑。
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"message":"消息撤回成功","data":{"id":901,"type":"text","content":"[消息已撤回]","is_recalled":true,"extra":"{\"original_content\":\"撤回前的原话\"}"}}}`))
			return
		}
		_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":null}`))
	}))
	defer srv.Close()

	c := New(srv.URL)
	c.SetSession("tkn", "rtk")
	ctx := context.Background()
	recalled, err := c.RecallMessage(ctx, "901")
	if err != nil {
		t.Fatalf("撤回: %v", err)
	}
	if !recalled.IsRecalled || recalled.RecallOriginal() != "撤回前的原话" {
		t.Errorf("撤回返回消息不对: isRecalled=%v original=%q", recalled.IsRecalled, recalled.RecallOriginal())
	}
	if err := c.RemindMessage(ctx, "901"); err != nil {
		t.Fatalf("提醒: %v", err)
	}
	if err := c.DeleteMessage(ctx, "901"); err != nil {
		t.Fatalf("删除: %v", err)
	}
	want := []call{
		{http.MethodPost, "/api/v1/messages/901/recall"},
		{http.MethodPost, "/api/v1/messages/901/remind"},
		{http.MethodDelete, "/api/v1/messages/901"},
	}
	if len(calls) != len(want) {
		t.Fatalf("调用次数不对: %+v", calls)
	}
	for i := range want {
		if calls[i] != want[i] {
			t.Errorf("第 %d 次调用 = %+v, 想要 %+v", i+1, calls[i], want[i])
		}
	}
	// 空 id 不该发出请求
	if err := c.DeleteMessage(ctx, ""); err == nil {
		t.Error("空 id 应该直接报错")
	}
	if len(calls) != 3 {
		t.Error("空 id 不该发 HTTP 请求")
	}
}
