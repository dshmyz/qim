package ai

import "testing"

func TestIntentRulesDoNotTreatDescriptiveStatementsAsRequests(t *testing.T) {
	d := NewIntentDetector(nil)
	for _, content := range []string{
		"明天开会讨论上线方案",
		"这个方案要删除旧接口",
		"下周需要完成联调",
		"I added a comment to the doc",
		"我已经把那个错误改完了",
		"我没什么意见",
	} {
		if got, strong := d.detectByRules(content); strong {
			t.Fatalf("descriptive statement %q unexpectedly treated as strong request: %+v", content, got)
		}
	}
}

func TestIntentRulesKeepExplicitRequests(t *testing.T) {
	d := NewIntentDetector(nil)
	cases := []struct {
		content string
		want    string
	}{
		{"请帮我记住明天上午提醒开会", "todo"},
		{"项目 Alpha 什么时候上线？", "query"},
		{"请把张三移出群聊", "command"},
		{"please delete the old files", "command"},
		{"add @张三 to the group", "command"},
		{"生产服务器挂了", "alert"},
		{"the service is down", "alert"},
	}
	for _, tt := range cases {
		got, strong := d.detectByRules(tt.content)
		if !strong || got == nil || got.Type != tt.want {
			t.Fatalf("content %q got %+v strong=%v, want strong %s", tt.content, got, strong, tt.want)
		}
	}
}

func TestIntentRulesWeakEnglishVerbsDoNotMatchWithinWords(t *testing.T) {
	d := NewIntentDetector(nil)
	// "address" 不得被 weak 命令正则里的 add 命中
	if got, strong := d.detectByRules("what address do you mean?"); strong && got.Type == "command" {
		t.Fatalf("address must not match command verb: %+v", got)
	}
}

func TestShouldTriggerAIReplyRequiresHighConfidenceForAutomaticIntents(t *testing.T) {
	d := NewIntentDetector(nil)
	if d.ShouldTriggerAIReply(&MessageIntent{Type: "query", Confidence: 0.6}, "group") {
		t.Fatal("low-confidence query must not auto-reply")
	}
	if d.ShouldTriggerAIReply(&MessageIntent{Type: "todo", Confidence: 0.7}, "group") {
		t.Fatal("low-confidence todo must not auto-reply")
	}
	if !d.ShouldTriggerAIReply(&MessageIntent{Type: "query", Confidence: 0.75}, "group") {
		t.Fatal("high-confidence query should auto-reply")
	}
}

func TestParseAIResultHonorsExplicitShouldReply(t *testing.T) {
	d := NewIntentDetector(nil)
	noReply, err := d.parseAIResult(`{"type":"query","confidence":0.95,"should_reply":false,"reason":"只是群成员之间的讨论"}`)
	if err != nil || noReply.ShouldReply == nil || *noReply.ShouldReply {
		t.Fatalf("explicit should_reply=false was lost: result=%+v err=%v", noReply, err)
	}

	reply, err := d.parseAIResult(`{"type":"query","confidence":0.95,"should_reply":true,"target":"ai"}`)
	if err != nil || reply.ShouldReply == nil || !*reply.ShouldReply || reply.Target != "ai" {
		t.Fatalf("explicit should_reply=true was lost: result=%+v err=%v", reply, err)
	}
}

func TestShouldTriggerAIReplyRespectsExplicitNoReply(t *testing.T) {
	d := NewIntentDetector(nil)
	noReply := false
	if d.ShouldTriggerAIReply(&MessageIntent{Type: "query", Confidence: 0.99, ShouldReply: &noReply}, "group") {
		t.Fatal("explicit should_reply=false must block auto-reply")
	}
}

func TestDetectFallsBackToSemanticPathOnWeakMatch(t *testing.T) {
	// AI 未配置时：弱命中返回低置信意图，不触发；无规则返回 chat
	d := NewIntentDetector(nil)
	intent, err := d.Detect("I added a comment", 1, 1)
	if err != nil || intent.Confidence >= 0.75 {
		t.Fatalf("weak match must stay low-confidence without AI: %+v err=%v", intent, err)
	}

	intent, err = d.Detect("今天天气不错", 1, 1)
	if err != nil || intent.Type != "chat" {
		t.Fatalf("no-rule message should default to chat: %+v err=%v", intent, err)
	}
}
