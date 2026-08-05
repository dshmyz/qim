package utils

import (
	crand "crypto/rand"
	"math/big"
)

// 常用字符集：与历史 ws.randomString / handler.generateShortCode 保持一致。
const (
	// Alphanumeric 小写+大写+数字，共 62 字符。
	Alphanumeric = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	// AlphanumericLower 仅小写+数字，共 36 字符。适用于短码等不区分大小写的场景。
	AlphanumericLower = "abcdefghijklmnopqrstuvwxyz0123456789"
)

// RandomString 用 crypto/rand 生成指定长度的随机字符串。
//
// 为什么不直接 rand.Read + %len(charset)？
// 256 不是任意 charset 长度的整数倍（如 62），直接取模会让前几个字符概率偏高（mod bias）。
// 标准库 crand.Int 在 [0, max) 上严格均匀采样，是消除 mod bias 的官方推荐方法，
// 也是项目内 generateShortCode 既有实现。这里抽到 utils 供多处复用。
//
// charset 为空时用 Alphanumeric。rand 失败时回退到 charset[0]（与原 generateShortCode 一致），
// 避免返回空串导致上游索引越界。
func RandomString(n int, charset string) string {
	if n <= 0 {
		return ""
	}
	if charset == "" {
		charset = Alphanumeric
	}
	max := big.NewInt(int64(len(charset)))
	out := make([]byte, n)
	for i := 0; i < n; i++ {
		idx, err := crand.Int(crand.Reader, max)
		if err != nil {
			// 极端情况（如熵不足）回退到首字符，避免返回空串
			out[i] = charset[0]
			continue
		}
		out[i] = charset[idx.Int64()]
	}
	return string(out)
}
