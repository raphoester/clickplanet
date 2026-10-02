//go:build testing

package cpsession

import (
	"crypto/ed25519"
	"encoding/hex"
)

func TestKeyPair() (secret string, publicKey string) {
	seed := make([]byte, ed25519.SeedSize)
	for i := range seed {
		seed[i] = byte(i + 1)
	}
	key := ed25519.NewKeyFromSeed(seed)

	return hex.EncodeToString(seed), hex.EncodeToString(key.Public().(ed25519.PublicKey))
}
