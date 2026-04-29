package utils

import (
	"os"
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	os.Setenv("ENCRYPTION_KEY", "test-key-32-chars-for-encryption!!")
	InitEncryptionKey()

	originalKey := "sk-test-api-key-12345"

	encrypted, err := EncryptAPIKey(originalKey)
	if err != nil {
		t.Fatalf("加密失败: %v", err)
	}

	if encrypted == originalKey {
		t.Fatal("加密后的密文不应等于原文")
	}

	decrypted, err := DecryptAPIKey(encrypted)
	if err != nil {
		t.Fatalf("解密失败: %v", err)
	}

	if decrypted != originalKey {
		t.Fatalf("解密结果不匹配: 期望 %s, 得到 %s", originalKey, decrypted)
	}
}

func TestDecryptInvalidKey(t *testing.T) {
	os.Setenv("ENCRYPTION_KEY", "test-key-32-chars-for-encryption!!")
	InitEncryptionKey()

	_, err := DecryptAPIKey("invalid-base64")
	if err == nil {
		t.Fatal("解密无效base64应该失败")
	}
}
