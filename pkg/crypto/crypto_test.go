package crypto

import (
	"os"
	"testing"
)

func TestHashPassword(t *testing.T) {
	pwd, err := GenerateRandomHex(16)
	if err != nil {
		t.Fatalf("GenerateRandomHex failed: %v", err)
	}

	hash, err := HashPassword(pwd)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	if !CheckPasswordHash(pwd, hash) {
		t.Fatalf("CheckPasswordHash should match for correct password")
	}

	wrongPwd, err := GenerateRandomHex(16)
	if err != nil {
		t.Fatalf("GenerateRandomHex failed: %v", err)
	}

	if CheckPasswordHash(wrongPwd, hash) {
		t.Fatalf("CheckPasswordHash should fail for wrong password")
	}
}

func TestTokens(t *testing.T) {
	hexStr, err := GenerateRandomHex(16)
	if err != nil || len(hexStr) != 32 {
		t.Fatalf("GenerateRandomHex failed or length mismatch: %v, len: %d", err, len(hexStr))
	}

	csrf, err := GenerateCSRFToken()
	if err != nil || len(csrf) != 64 {
		t.Fatalf("GenerateCSRFToken failed or length mismatch: %v, len: %d", err, len(csrf))
	}
}

func TestAESGCMEncryption(t *testing.T) {
	key := GetLLMEncryptionKey()
	if len(key) != 32 {
		t.Fatalf("Expected 32-byte key, got %d", len(key))
	}

	rawApiKey := os.Getenv("OPEN_ROUTER_API")
	if rawApiKey == "" {
		generated, err := GenerateRandomHex(24)
		if err != nil {
			t.Fatalf("Failed to generate random string: %v", err)
		}
		rawApiKey = generated
	}

	encrypted, err := EncryptAESGCM(rawApiKey, key)
	if err != nil {
		t.Fatalf("EncryptAESGCM failed: %v", err)
	}

	if encrypted == rawApiKey {
		t.Fatalf("Ciphertext should differ from plaintext")
	}

	decrypted, err := DecryptAESGCM(encrypted, key)
	if err != nil {
		t.Fatalf("DecryptAESGCM failed: %v", err)
	}

	if decrypted != rawApiKey {
		t.Fatalf("Decrypted '%s' does not match raw '%s'", decrypted, rawApiKey)
	}

	wrongKey := make([]byte, 32)
	copy(wrongKey, key)
	wrongKey[0] ^= 0xFF
	_, err = DecryptAESGCM(encrypted, wrongKey)
	if err == nil {
		t.Fatalf("Decrypt with wrong key should fail")
	}
}
