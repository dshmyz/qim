package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"os"
)

var encryptionKey []byte

func InitEncryptionKey() {
	key := os.Getenv("ENCRYPTION_KEY")
	if key == "" {
		// 生成随机密钥，每次启动都会变化（已加密的数据将无法解密）
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			panic("无法生成随机加密密钥: " + err.Error())
		}
		key = base64.StdEncoding.EncodeToString(b)[:32]
		fmt.Println("警告: 未设置 ENCRYPTION_KEY 环境变量，使用随机密钥。已加密的数据在重启后将无法解密。")
	}
	encryptionKey = []byte(key)[:32]
}

func EncryptAPIKey(apiKey string) (string, error) {
	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", fmt.Errorf("创建加密器失败: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("创建GCM失败: %w", err)
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("生成nonce失败: %w", err)
	}

	ciphertext := aesGCM.Seal(nonce, nonce, []byte(apiKey), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func DecryptAPIKey(encryptedKey string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(encryptedKey)
	if err != nil {
		return "", fmt.Errorf("解码失败: %w", err)
	}

	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", fmt.Errorf("创建解密器失败: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("创建GCM失败: %w", err)
	}

	nonceSize := aesGCM.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("密文长度无效")
	}

	plaintext, err := aesGCM.Open(nil, data[:nonceSize], data[nonceSize:], nil)
	if err != nil {
		return "", fmt.Errorf("解密失败: %w", err)
	}

	return string(plaintext), nil
}
