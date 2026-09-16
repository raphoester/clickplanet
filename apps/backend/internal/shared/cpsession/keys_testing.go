//go:build testing

package cpsession

import (
	"crypto/ed25519"
	"encoding/hex"
)

// TestKeyPair is one fixed Ed25519 pair, so a test names both halves without
// carrying two magic strings. It is in git, so it is never a real key.
func TestKeyPair() (secret string, publicKey string) {
	seed := make([]byte, ed25519.SeedSize)
	for i := range seed {
		seed[i] = byte(i + 1)
	}
	key := ed25519.NewKeyFromSeed(seed)

	return hex.EncodeToString(seed), hex.EncodeToString(key.Public().(ed25519.PublicKey))
}
