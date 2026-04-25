package handler

import (
	"crypto/rand"
	"encoding/base32"
	"github.com/pquerna/otp/totp"
)

// GenerateTOTPSecret 生成 TOTP 密钥
func GenerateTOTPSecret() (string, error) {
	b := make([]byte, 20)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base32.StdEncoding.EncodeToString(b), nil
}

// VerifyTOTPCode 验证 TOTP 代码
func VerifyTOTPCode(secret, code string) bool {
	return totp.Validate(code, secret)
}
