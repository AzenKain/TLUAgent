package crypto

import (
	"crypto/rand"
	"encoding/hex"
	"io"
)

func GenerateRandomHex(nBytes int) (string, error) {
	b := make([]byte, nBytes)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func GenerateCSRFToken() (string, error) {
	return GenerateRandomHex(32)
}
