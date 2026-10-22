package random_token_secret_generator

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/admintokens"
)

const secretBytes = 32

type Generator struct{}

var _ admintokens.SecretGenerator = Generator{}

func (Generator) NewSecret(prefix string) (admintokens.Secret, error) {
	raw := make([]byte, secretBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("failed to read random bytes: %w", err)
	}
	return admintokens.Secret(prefix + base64.RawURLEncoding.EncodeToString(raw)), nil
}
