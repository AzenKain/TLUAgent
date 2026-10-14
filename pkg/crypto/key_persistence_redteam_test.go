package crypto

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// This passing test reproduces credential loss after a deterministic key-file write failure.
func TestRedTeamObservedWeaknessKeyPersistenceFailure(t *testing.T) {
	previousKey := llmSecretKey
	llmSecretKey = nil
	llmKeyOnce = sync.Once{}
	t.Cleanup(func() {
		llmSecretKey = previousKey
		llmKeyOnce = sync.Once{}
		if previousKey != nil {
			llmKeyOnce.Do(func() {})
		}
	})

	t.Setenv("LLM_ENCRYPTION_KEY", "")
	dataDir := t.TempDir()
	t.Setenv("DATA_DIR", dataDir)
	secretPath := filepath.Join(dataDir, ".llm_secret")
	if err := os.Mkdir(secretPath, 0700); err != nil {
		t.Fatal("cannot create deterministic key-write failure fixture")
	}

	credentialBytes := make([]byte, 32)
	if _, err := rand.Read(credentialBytes); err != nil {
		t.Fatal("cannot generate runtime credential")
	}
	credential := hex.EncodeToString(credentialBytes)
	firstKey := append([]byte(nil), GetLLMEncryptionKey()...)
	if len(firstKey) != 32 {
		t.Fatal("getter did not silently provide a usable key")
	}
	ciphertext, err := EncryptAESGCM(credential, firstKey)
	if err != nil {
		t.Fatal("getter did not permit encryption after persistence failure")
	}
	info, err := os.Stat(secretPath)
	if err != nil || !info.IsDir() {
		t.Fatal("deterministic key-write failure fixture changed")
	}

	llmSecretKey = nil
	llmKeyOnce = sync.Once{}
	secondKey := GetLLMEncryptionKey()
	if len(secondKey) != 32 {
		t.Fatal("simulated restart did not provide another usable key")
	}
	decrypted, err := DecryptAESGCM(ciphertext, secondKey)
	if err != nil || decrypted != credential {
		t.Fatalf("simulated restart failed to decrypt credential using persistent fallback: %v", err)
	}
}
