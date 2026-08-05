package utils

import (
	"strings"
	"testing"
)

// TestRandomString_LengthAndAlphabet 验证返回长度正确，且只包含给定字符集。
func TestRandomString_LengthAndAlphabet(t *testing.T) {
	cases := []struct {
		n       int
		charset string
	}{
		{1, Alphanumeric},
		{8, Alphanumeric},
		{16, AlphanumericLower},
		{64, Alphanumeric},
		{6, "0123456789"}, // 纯数字
	}
	for _, c := range cases {
		s := RandomString(c.n, c.charset)
		if len(s) != c.n {
			t.Fatalf("期望长度 %d，实际 %d (s=%q)", c.n, len(s), s)
		}
		for i, ch := range s {
			if !strings.ContainsRune(c.charset, ch) {
				t.Fatalf("第 %d 个字符 %q 不在字符集 %q 中 (s=%q)", i, ch, c.charset, s)
			}
		}
	}
}

// TestRandomString_EmptyCharset_DefaultsToAlphanumeric 空字符集回退到 Alphanumeric。
func TestRandomString_EmptyCharset_DefaultsToAlphanumeric(t *testing.T) {
	s := RandomString(8, "")
	if len(s) != 8 {
		t.Fatalf("期望长度 8，实际 %d", len(s))
	}
	for _, ch := range s {
		if !strings.ContainsRune(Alphanumeric, ch) {
			t.Fatalf("字符 %q 不在默认字符集中 (s=%q)", ch, s)
		}
	}
}

// TestRandomString_ZeroOrNegative 空字符串安全返回。
func TestRandomString_ZeroOrNegative(t *testing.T) {
	if s := RandomString(0, Alphanumeric); s != "" {
		t.Fatalf("期望空字符串，实际 %q", s)
	}
	if s := RandomString(-5, Alphanumeric); s != "" {
		t.Fatalf("期望空字符串，实际 %q", s)
	}
}

// TestRandomString_Uniqueness 多次调用不碰撞（极小概率视为失败）。
func TestRandomString_Uniqueness(t *testing.T) {
	seen := make(map[string]struct{}, 1000)
	for i := 0; i < 1000; i++ {
		s := RandomString(8, Alphanumeric)
		if _, dup := seen[s]; dup {
			t.Fatalf("第 %d 次生成出现碰撞: %q", i, s)
		}
		seen[s] = struct{}{}
	}
}

// TestRandomString_Distribution 验证字符分布均匀（无 mod bias）。
// 旧 rand.Read+%62 实现因 256 不是 62 整数倍，前 8 个字母概率偏高。
// 改用 crand.Int 后各字符应接近 1/len(charset)。大样本 50000 + 容差 30%。
func TestRandomString_Distribution(t *testing.T) {
	const total = 50000
	const n = 8
	counts := make(map[rune]int, len(Alphanumeric))
	for i := 0; i < total/n; i++ {
		for _, c := range RandomString(n, Alphanumeric) {
			counts[c]++
		}
	}
	if len(counts) != len(Alphanumeric) {
		t.Fatalf("期望覆盖 %d 个字符，实际 %d", len(Alphanumeric), len(counts))
	}
	expected := float64(total) / float64(len(Alphanumeric))
	lo := expected * 0.7
	hi := expected * 1.3
	for _, c := range Alphanumeric {
		got := float64(counts[c])
		if got < lo || got > hi {
			t.Fatalf("字符 %q 出现 %v 次，超出 [%.1f, %.1f] 范围（mod bias 未修复）",
				c, got, lo, hi)
		}
	}
}
