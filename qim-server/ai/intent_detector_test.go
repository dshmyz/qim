package ai

import "testing"

func TestIntentRulesDoNotTreatDescriptiveStatementsAsRequests(t *testing.T) {
	d := NewIntentDetector(nil)
	for _, content := range []string{
		"明天开会讨论上线方案",
		"这个方案要删除旧接口",
		"下周需要完成联调",
	} {
		if got := d.detectByRules(content); got != nil {
			t.Fatalf("descriptive statement %q unexpectedly classified as %+v", content, got)
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
	}
	for _, tt := range cases {
		got := d.detectByRules(tt.content)
		if got == nil || got.Type != tt.want {
			t.Fatalf("content %q got %+v, want %s", tt.content, got, tt.want)
		}
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
