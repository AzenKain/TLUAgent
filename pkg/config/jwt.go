package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	jwtSecretOnce          sync.Once
	jwtSecretCached        string
	jwtRefreshSecretOnce   sync.Once
	jwtRefreshSecretCached string
)

const minSecretLength = 32

// validateSecretStrength rejects environment-provided secrets shorter than 32 characters.
func validateSecretStrength(envKey, secret string) error {
	if len(secret) < minSecretLength {
		return fmt.Errorf("security configuration error: %s must be at least %d characters long (got %d); leave it empty to auto-generate a secure key instead", envKey, minSecretLength, len(secret))
	}
	return nil
}

func getOrGenerateSecret(envKey, secretFileName string) string {
	val := strings.TrimSpace(os.Getenv(envKey))
	if val != "" {
		if err := validateSecretStrength(envKey, val); err != nil {
			log.Fatalf("%v", err)
		}
		return val
	}

	dataDir := GetConfigWithDefault("DATA_DIR", "./data")
	_ = os.MkdirAll(dataDir, 0700)
	secretFilePath := filepath.Join(dataDir, secretFileName)

	if content, err := os.ReadFile(secretFilePath); err == nil {
		secret := strings.TrimSpace(string(content))
		if len(secret) >= 32 {
			return secret
		}
	}

	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		log.Fatalf("Critical security error: failed to generate secure random bytes for %s: %v", envKey, err)
	}
	generatedSecret := hex.EncodeToString(randomBytes)

	if err := os.WriteFile(secretFilePath, []byte(generatedSecret), 0600); err != nil {
		log.Printf("[SECURITY NOTICE] Failed to persist generated %s to %s: %v (using memory-only secret for this session)", envKey, secretFilePath, err)
	} else {
		log.Printf("[SECURITY NOTICE] Generated new cryptographically secure %s and stored in %s", envKey, secretFilePath)
	}

	return generatedSecret
}

// GetJWTSecret returns the persistent JWT signing secret.
func GetJWTSecret() string {
	jwtSecretOnce.Do(func() {
		jwtSecretCached = getOrGenerateSecret("JWT_SECRET", ".jwt_secret")
	})
	return jwtSecretCached
}

// GetJWTRefreshSecret returns the persistent JWT refresh token signing secret.
func GetJWTRefreshSecret() string {
	jwtRefreshSecretOnce.Do(func() {
		jwtRefreshSecretCached = getOrGenerateSecret("JWT_REFRESH_SECRET", ".jwt_refresh_secret")
	})
	return jwtRefreshSecretCached
}
