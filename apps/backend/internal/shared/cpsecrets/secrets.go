package cpsecrets

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

const defaultLength = 16

func RandomHex() (string, error) {
	buf := make([]byte, defaultLength)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("failed to read random bytes: %w", err)
	}

	return hex.EncodeToString(buf), nil
}
