// Package random_secret_generator draws sign-in secrets from crypto/rand: 32 bytes, base64url, which is also a valid PKCE verifier.
package random_secret_generator

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin"
)

const secretBytes = 32

type Generator struct{}

var _ signin.Secrets = Generator{}

func (Generator) NewSecret() (string, error) {
	raw := make([]byte, secretBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("failed to read random bytes: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}
