package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_ValidateCORS_RejectsWildcardWithCredentials(t *testing.T) {
	cfg := &Config{
		CORS: CORSConfig{
			AllowedOrigins:   []string{"*"},
			AllowCredentials: true,
		},
	}

	cfg.ValidateCORS()

	// 当 AllowCredentials=true 且 Origins 含 "*" 时，应将 Origins 设为空切片
	assert.NotContains(t, cfg.CORS.AllowedOrigins, "*",
		"ValidateCORS should remove wildcard origin when credentials are enabled")
}

func Test_ValidateCORS_AcceptsSpecificOrigins(t *testing.T) {
	cfg := &Config{
		CORS: CORSConfig{
			AllowedOrigins: []string{"https://example.com"},
		},
	}

	cfg.ValidateCORS()

	// 指定具体域名时，不应修改
	assert.Equal(t, []string{"https://example.com"}, cfg.CORS.AllowedOrigins,
		"ValidateCORS should not modify specific origins")
}
