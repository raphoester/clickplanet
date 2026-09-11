// Package secrets generates the values a config may leave empty.
//
// Only the chat tag salt qualifies: one module derives it and nothing else
// reads it, so an invented one costs a restart's worth of tags. The session
// secret is not here — two contexts derive a signer from it, and one the server
// invented would differ between them.
package secrets

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

const defaultLength = 16

// RandomHex returns a hex-encoded random string.
func RandomHex() (string, error) {
	buf := make([]byte, defaultLength)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("failed to read random bytes: %w", err)
	}

	return hex.EncodeToString(buf), nil
}
