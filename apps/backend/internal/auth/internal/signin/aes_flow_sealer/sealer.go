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
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin"
)

const keyInfo = "clickplanet auth cp_oauth v1"

type Sealer struct {
	aead cipher.AEAD
}

var (
	_ signin.Sealer          = (*Sealer)(nil)
	_ signin.ChallengeSealer = (*Sealer)(nil)
)

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

// The keys and types are the cookie's format: a cookie sealed before a release opens after it.
type flowCookie struct {
	Provider  string
	State     string
	Verifier  string
	Nonce     string
	ExpiresAt time.Time
	Intent    accounts.Intent
	Account   accounts.AccountID
}

type challengeCookie struct {
	ID        string
	Address   signin.Address
	Code      string
	ExpiresAt time.Time
	Intent    accounts.Intent
	Account   accounts.AccountID
}

func (s *Sealer) Sealed(flow *signin.Flow) (string, error) {
	return s.sealed(flowCookie{
		Provider: flow.Provider(), State: flow.State(), Verifier: flow.Verifier(), Nonce: flow.Nonce(),
		ExpiresAt: flow.ExpiresAt(), Intent: flow.Intent(), Account: flow.Account(),
	}, signin.FlowCookieName)
}

func (s *Sealer) Opened(sealed string) (*signin.Flow, error) {
	var opened flowCookie
	if err := s.open(sealed, signin.FlowCookieName, &opened); err != nil {
		return nil, err
	}
	return signin.FlowOf(opened.Provider, opened.State, opened.Verifier, opened.Nonce, opened.ExpiresAt, opened.Intent, opened.Account), nil
}

func (s *Sealer) SealedChallenge(challenge *signin.Challenge) (string, error) {
	return s.sealed(challengeCookie{
		ID: challenge.ID(), Address: challenge.Address(), Code: challenge.Code(),
		ExpiresAt: challenge.ExpiresAt(), Intent: challenge.Intent(), Account: challenge.Account(),
	}, signin.ChallengeCookieName)
}

func (s *Sealer) OpenedChallenge(sealed string) (*signin.Challenge, error) {
	var opened challengeCookie
	if err := s.open(sealed, signin.ChallengeCookieName, &opened); err != nil {
		return nil, err
	}
	return signin.ChallengeOf(opened.ID, opened.Address, opened.Code, opened.ExpiresAt, opened.Intent, opened.Account), nil
}

func (s *Sealer) sealed(value any, cookie string) (string, error) {
	plain, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("failed to encode the %s cookie: %w", cookie, err)
	}
	// The cookie name is the additional data, so one cookie's value never opens as the other.
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
