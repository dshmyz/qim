package service

import (
	"testing"
)

func TestParseRememberVerdictJSON(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		remember bool
		imp      float64
	}{
		{"标准 JSON", `{"remember": true, "importance": 4}`, true, 4},
		{"带噪音文本", `好的，我判断一下：{"remember":false,"importance":2} 结束`, false, 2},
		{"importance 为字符串档位", `{"remember":true,"importance":"5"}`, true, 5},
		{"importance 越界", `{"remember":true,"importance":99}`, true, 5},
		{"importance 负数", `{"remember":true,"importance":-3}`, true, 1},
		{"无 importance 默认 3", `{"remember":true}`, true, 3},
		{"should_remember 别名", `{"should_remember":true,"importance":3}`, true, 3},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			v, ok := parseRememberVerdictJSON(c.input)
			if !ok {
				t.Fatalf("parse failed for %q", c.input)
			}
			if v.ShouldRemember != c.remember {
				t.Errorf("remember = %v, want %v", v.ShouldRemember, c.remember)
			}
			if v.Importance != c.imp {
				t.Errorf("importance = %v, want %v", v.Importance, c.imp)
			}
		})
	}
}

func TestParseRememberVerdictJSON_NoJSON(t *testing.T) {
	if _, ok := parseRememberVerdictJSON("true"); ok {
		t.Fatal("plain true should not be parsed as JSON verdict")
	}
}

func TestImportance01(t *testing.T) {
	cases := []struct{ in, want float64 }{
		{1, 0.2},
		{3, 0.6},
		{5, 1.0},
		{0, 0.2},   // 低于下限，收敛到 1
		{99, 1.0},  // 高于上限，收敛到 5
	}
	for _, c := range cases {
		if got := importance01(c.in); got != c.want {
			t.Errorf("importance01(%v) = %v, want %v", c.in, got, c.want)
		}
	}
}
