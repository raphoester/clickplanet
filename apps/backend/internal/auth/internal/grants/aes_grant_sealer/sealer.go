package aes_grant_sealer

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hkdf"
	"crypto/sha256"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/grants"
)

// Its own label, so this key is not the cp_oauth cookie's though both come from auth.secret.
const keyInfo = "clickplanet auth discord_grants v1"

var ErrNotSealedHere = errors.New("the token was not sealed here, or not for this grant")

type Sealer struct {
	aead cipher.AEAD
}

var _ grants.Sealer = (*Sealer)(nil)

func New(seed []byte) (*Sealer, error) {
	if len(seed) == 0 {
		return nil, errors.New("the seed is empty")
	}

	key, err := hkdf.Key(sha256.New, seed, nil, keyInfo, 32)
	if err != nil {
		return nil, fmt.Errorf("failed to derive the key: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to build the cipher: %w", err)
	}
	aead, err := cipher.NewGCMWithRandomNonce(block)
	if err != nil {
		return nil, fmt.Errorf("failed to build the AEAD: %w", err)
	}
	return &Sealer{aead: aead}, nil
}

// The grant's id is the additional data: a sealed token opens for its own grant alone.
func (s *Sealer) Sealed(grant grants.GrantID, token string) ([]byte, error) {
	return s.aead.Seal(nil, nil, []byte(token), grant[:]), nil
}

func (s *Sealer) Opened(grant grants.GrantID, sealed []byte) (string, error) {
	plain, err := s.aead.Open(nil, nil, sealed, grant[:])
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrNotSealedHere, grant)
	}
	return string(plain), nil
}
