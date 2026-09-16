// Package random_token_generator draws session tokens from crypto/rand: 32 bytes, base64url.
package random_token_generator

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
)

const tokenBytes = 32

type Generator struct{}

var _ accounts.TokenGenerator = Generator{}

func (Generator) NewToken() (*accounts.Token, error) {
	raw := make([]byte, tokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return nil, fmt.Errorf("failed to read random bytes: %w", err)
	}

	return accounts.TokenOf(base64.RawURLEncoding.EncodeToString(raw)), nil
}
