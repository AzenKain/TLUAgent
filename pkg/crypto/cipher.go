package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"tluagent-web/pkg/config"
)

var (
	llmKeyOnce   sync.Once
	llmSecretKey []byte
)

// EncryptAESGCM encrypts a plaintext string using AES-256-GCM and returns standard Base64 string.
func EncryptAESGCM(plaintext string, key []byte) (string, error) {
	if len(key) != 32 {
		return "", errors.New("cipher key must be exactly 32 bytes for AES-256")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("create cipher block: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create gcm: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}

	sealed := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

// DecryptAESGCM decrypts a Base64 encoded AES-256-GCM ciphertext.
func DecryptAESGCM(ciphertextB64 string, key []byte) (string, error) {
	if len(key) != 32 {
		return "", errors.New("cipher key must be exactly 32 bytes for AES-256")
	}

	data, err := base64.StdEncoding.DecodeString(ciphertextB64)
	if err != nil {
		return "", fmt.Errorf("decode base64 ciphertext: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("create cipher block: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create gcm: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, actualCiphertext := data[:nonceSize], data[nonceSize:]
	decrypted, err := gcm.Open(nil, nonce, actualCiphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt ciphertext: %w", err)
	}

	return string(decrypted), nil
}

// GetLLMEncryptionKey returns a persistent 32-byte key for encrypting provider credentials.
func GetLLMEncryptionKey() []byte {
	llmKeyOnce.Do(func() {
		envKey := strings.TrimSpace(os.Getenv("LLM_ENCRYPTION_KEY"))
		if envKey != "" {
			hash := sha256.Sum256([]byte(envKey))
			llmSecretKey = hash[:]
			return
		}

		dataDir := config.GetConfigWithDefault("DATA_DIR", "./data")
		_ = os.MkdirAll(dataDir, 0700)
		secretPath := filepath.Join(dataDir, ".llm_secret")

		if content, err := os.ReadFile(secretPath); err == nil && len(content) == 32 {
			llmSecretKey = content
			return
		}

		generated := make([]byte, 32)
		if _, err := io.ReadFull(rand.Reader, generated); err != nil {
			hash := sha256.Sum256([]byte(config.GetJWTSecret() + "-llm-fallback-salt"))
			llmSecretKey = hash[:]
			return
		}

		_ = os.WriteFile(secretPath, generated, 0600)
		llmSecretKey = generated
	})

	return llmSecretKey
}
