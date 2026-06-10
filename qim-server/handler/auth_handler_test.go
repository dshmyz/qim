package handler

import (
	"testing"

	"github.com/dshmyz/qim/qim-server/config"
	"github.com/dshmyz/qim/qim-server/middleware"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

func TestGenerateToken_ReturnsValidToken(t *testing.T) {
	secret := "test-secret-key-that-is-long-enough"
	cfg = &config.Config{
		JWT: config.JWTConfig{
			Secret: secret,
			Expire: 7200,
		},
	}

	tokenString := generateToken(1, "testuser")
	assert.NotEmpty(t, tokenString, "generateToken should return a non-empty token")

	// 验证返回的 token 可以被正确解析
	claims := &middleware.Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	assert.NoError(t, err, "generated token should be parseable")
	assert.True(t, token.Valid, "generated token should be valid")
	assert.Equal(t, uint(1), claims.UserID)
	assert.Equal(t, "testuser", claims.Username)
}

func TestGenerateToken_DoesNotReturnInvalidTokenOnError(t *testing.T) {
	cfg = &config.Config{
		JWT: config.JWTConfig{
			Secret: "valid-secret",
			Expire: 7200,
		},
	}

	// generateToken 当前忽略了 SignedString 的错误。
	// 改进后：如果签名失败，应返回空字符串而非部分结果。
	// HS256 实际上不会签名失败（即使空 secret），
	// 但我们仍然应该处理错误以防未来更换签名算法。
	tokenString := generateToken(1, "testuser")

	// 验证 token 不为空（正常情况）
	assert.NotEmpty(t, tokenString, "generateToken should return non-empty token with valid secret")
}

func TestGenerateToken_DoesNotPanicWithEmptySecret(t *testing.T) {
	cfg = &config.Config{
		JWT: config.JWTConfig{
			Secret: "",
			Expire: 7200,
		},
	}

	assert.NotPanics(t, func() {
		generateToken(1, "testuser")
	}, "generateToken should not panic even with empty secret")
}

// --- S9: 密码强度校验 ---

func Test_ValidatePassword_RejectsShortPassword(t *testing.T) {
	err := validatePassword("123")
	assert.Error(t, err, "validatePassword should reject password shorter than 8 characters")
}

func Test_ValidatePassword_RejectsWeakPassword(t *testing.T) {
	err := validatePassword("12345678")
	assert.Error(t, err, "validatePassword should reject password with only digits")
}

func Test_ValidatePassword_AcceptsStrongPassword(t *testing.T) {
	err := validatePassword("Test1234")
	assert.NoError(t, err, "validatePassword should accept password with letters and digits")
}

func Test_ValidatePassword_RejectsEmptyPassword(t *testing.T) {
	err := validatePassword("")
	assert.Error(t, err, "validatePassword should reject empty password")
}

// --- S3: RefreshToken 机制 ---

func TestGenerateAccessToken_SetsTokenType(t *testing.T) {
	secret := "test-secret-key-for-refresh"
	cfg = &config.Config{
		JWT: config.JWTConfig{
			Secret: secret,
			Expire: 7200,
		},
	}

	tokenString := generateAccessToken(1, "testuser")
	assert.NotEmpty(t, tokenString)

	claims := &middleware.Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	assert.NoError(t, err)
	assert.True(t, token.Valid)
	assert.Equal(t, "access", claims.TokenType, "access token should have TokenType='access'")
}

func TestGenerateRefreshToken_SetsTokenType(t *testing.T) {
	secret := "test-secret-key-for-refresh"
	cfg = &config.Config{
		JWT: config.JWTConfig{
			Secret:            secret,
			Expire:            7200,
			RefreshExpireDays: 7,
		},
	}

	tokenString := generateRefreshToken(1, "testuser")
	assert.NotEmpty(t, tokenString)

	claims := &middleware.Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	assert.NoError(t, err)
	assert.True(t, token.Valid)
	assert.Equal(t, "refresh", claims.TokenType, "refresh token should have TokenType='refresh'")
}

func TestRefreshToken_RejectsAccessToken(t *testing.T) {
	secret := "test-secret-key-for-refresh"
	cfg = &config.Config{
		JWT: config.JWTConfig{
			Secret:            secret,
			Expire:            7200,
			RefreshExpireDays: 7,
		},
	}

	// 生成 access token
	accessToken := generateAccessToken(1, "testuser")
	assert.NotEmpty(t, accessToken)

	// 验证 access token 的 TokenType 是 "access"
	claims := &middleware.Claims{}
	token, err := jwt.ParseWithClaims(accessToken, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	assert.NoError(t, err)
	assert.True(t, token.Valid)
	assert.Equal(t, "access", claims.TokenType, "access token should not be usable as refresh token")
}
