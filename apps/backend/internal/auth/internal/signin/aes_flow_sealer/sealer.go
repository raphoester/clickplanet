// Package aes_flow_sealer seals a sign-in flow with AES-256-GCM, under a key derived from the click token seed.
package aes_flow_sealer

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hkdf"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin"
)

// The label keeps this key apart from every other use of the seed.
const keyInfo = "clickplanet auth cp_oauth v1"

type Sealer struct {
	aead cipher.AEAD
}

var _ signin.Sealer = (*Sealer)(nil)

// New derives the key from seed, the auth.secret bytes, so there is no second secret to set.
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

func (s *Sealer) Sealed(flow *signin.Flow) (string, error) {
	plain, err := json.Marshal(flow)
	if err != nil {
		return "", fmt.Errorf("failed to encode the flow: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(s.aead.Seal(nil, nil, plain, []byte(signin.FlowCookieName))), nil
}

func (s *Sealer) Opened(sealed string) (*signin.Flow, error) {
	raw, err := base64.RawURLEncoding.DecodeString(sealed)
	if err != nil {
		return nil, fmt.Errorf("%w: the cookie is not base64url", signin.ErrFlowInvalid)
	}
	plain, err := s.aead.Open(nil, nil, raw, []byte(signin.FlowCookieName))
	if err != nil {
		return nil, fmt.Errorf("%w: the cookie was not sealed here", signin.ErrFlowInvalid)
	}

	var flow signin.Flow
	if err := json.Unmarshal(plain, &flow); err != nil {
		return nil, fmt.Errorf("%w: the sealed flow does not decode: %w", signin.ErrFlowInvalid, err)
	}
	return &flow, nil
}
