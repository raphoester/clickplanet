// Package random_code_generator draws the six digits an email sign-in sends, uniformly from crypto/rand.
package random_code_generator

import (
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin"
)

var codes = big.NewInt(1_000_000)

type Generator struct{}

var _ signin.Codes = Generator{}

func (Generator) NewCode() (string, error) {
	n, err := rand.Int(rand.Reader, codes)
	if err != nil {
		return "", fmt.Errorf("failed to read random bytes: %w", err)
	}
	return fmt.Sprintf("%06d", n), nil
}
