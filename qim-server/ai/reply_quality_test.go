package ai

import "testing"

func TestParseReplyQualityVerdictExtractsJSON(t *testing.T) {
	got, err := ParseReplyQualityVerdict("```json\n{\"relevant\":true,\"supported\":false,\"hallucinate\":true,\"confidence\":0.9,\"reason\":\"无依据\"}\n```")
	if err != nil {
		t.Fatal(err)
	}
	if !got.Relevant || got.Supported || !got.Hallucinate || got.Confidence != 0.9 || got.ShouldSend() {
		t.Fatalf("unexpected verdict: %+v", got)
	}
}

func TestReplyQualityVerdictShouldSend(t *testing.T) {
	tests := []struct {
		name string
		v    ReplyQualityVerdict
		want bool
	}{
		{name: "通过", v: ReplyQualityVerdict{Relevant: true, Supported: true}, want: true},
		{name: "低相关", v: ReplyQualityVerdict{Relevant: false, Supported: true}, want: false},
		{name: "无依据", v: ReplyQualityVerdict{Relevant: true, Supported: false}, want: false},
		{name: "幻觉", v: ReplyQualityVerdict{Relevant: true, Supported: true, Hallucinate: true}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.v.ShouldSend(); got != tt.want {
				t.Fatalf("ShouldSend()=%v, want %v; verdict=%+v", got, tt.want, tt.v)
			}
		})
	}
}
