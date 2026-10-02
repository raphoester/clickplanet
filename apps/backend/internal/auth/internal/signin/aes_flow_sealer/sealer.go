// Package aes_flow_sealer seals a sign-in flow, or an email challenge, with AES-256-GCM, under a key derived from the click token seed.
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

var (
	_ signin.Sealer          = (*Sealer)(nil)
	_ signin.ChallengeSealer = (*Sealer)(nil)
)

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
	return s.sealed(flow, signin.FlowCookieName)
}

func (s *Sealer) Opened(sealed string) (*signin.Flow, error) {
	var flow signin.Flow
	if err := s.open(sealed, signin.FlowCookieName, &flow); err != nil {
		return nil, err
	}
	return &flow, nil
}

func (s *Sealer) SealedChallenge(challenge *signin.Challenge) (string, error) {
	return s.sealed(challenge, signin.ChallengeCookieName)
}

func (s *Sealer) OpenedChallenge(sealed string) (*signin.Challenge, error) {
	var challenge signin.Challenge
	if err := s.open(sealed, signin.ChallengeCookieName, &challenge); err != nil {
		return nil, err
	}
	return &challenge, nil
}

// The cookie name is the additional data, so one cookie's value does not open as the other.
func (s *Sealer) sealed(value any, cookie string) (string, error) {
	plain, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("failed to encode the %s cookie: %w", cookie, err)
	}
	return base64.RawURLEncoding.EncodeToString(s.aead.Seal(nil, nil, plain, []byte(cookie))), nil
}

func (s *Sealer) open(sealed string, cookie string, into any) error {
	raw, err := base64.RawURLEncoding.DecodeString(sealed)
	if err != nil {
		return fmt.Errorf("%w: the cookie is not base64url", signin.ErrFlowInvalid)
	}
	plain, err := s.aead.Open(nil, nil, raw, []byte(cookie))
	if err != nil {
		return fmt.Errorf("%w: the cookie was not sealed here", signin.ErrFlowInvalid)
	}
	if err := json.Unmarshal(plain, into); err != nil {
		return fmt.Errorf("%w: the sealed %s cookie does not decode: %w", signin.ErrFlowInvalid, cookie, err)
	}
	return nil
}
