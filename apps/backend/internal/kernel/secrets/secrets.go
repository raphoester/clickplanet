// Package secrets generates the values a config may leave empty.
//
// Both the chat tag salt and the session signing key are secrets the server
// will invent rather than refuse to start without, and both pay the same price
// for it: what the old one covered stops being recognised on every restart.
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
