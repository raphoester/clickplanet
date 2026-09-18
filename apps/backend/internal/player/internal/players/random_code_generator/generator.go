// Package random_code_generator draws guest codes from crypto/rand: 3 bytes, as 6 hex characters.
package random_code_generator

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type Generator struct{}

var _ players.CodeGenerator = Generator{}

func (Generator) NewGuestCode() (players.GuestCode, error) {
	raw := make([]byte, players.GuestCodeLength/2)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("failed to read random bytes: %w", err)
	}

	return players.GuestCodeOf(hex.EncodeToString(raw)) //nolint:wrapcheck // hex of 3 bytes is always a code.
}
