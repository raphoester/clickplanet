// Package cpsession mints and verifies the opaque token a caller has to hold to
// click. It is deliberately stateless: the token carries its own expiry and a
// MAC over it, so nothing has to be stored, swept, or replicated, and a restart
// does not log every player out.
package cpsession

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpipscope"
)

var (
	ErrMalformed    = errors.New("malformed session token")
	ErrBadSignature = errors.New("session token signature does not match")
	ErrExpired      = errors.New("session token has expired")
)

const (
	expiryLen = 8
	idLen     = 8
	macLen    = sha256.Size

	tokenLen = expiryLen + idLen + macLen
)

// ID identifies one minted session. It is not a secret and not an identity —
// it exists so a refusal can be correlated in the logs with the mint that
// preceded it.
type ID string

type Token struct {
	Value     string
	ID        ID
	ExpiresAt time.Time
}

type Signer struct {
	key []byte
	ttl time.Duration
}

// NewSigner builds the same signer for anyone holding the same config, which
// is what lets the minting context and the verifying context each build their
// own instead of passing one between them.
func NewSigner(config Config) (*Signer, error) {
	config = config.withDefaults()

	if config.Secret == "" {
		return nil, errors.New("session secret is empty")
	}
	if config.TTL <= 0 {
		return nil, fmt.Errorf("session ttl must be positive, got %s", config.TTL)
	}

	return &Signer{key: []byte(config.Secret), ttl: config.TTL}, nil
}

func (s *Signer) TTL() time.Duration {
	return s.ttl
}

// Mint binds the token to the scope ip sits in — the address itself over IPv4,
// the surrounding /64 over IPv6. A caller that leaves it mid-session — a phone
// moving from wifi to cellular — fails verification and mints again, which is
// the intended behaviour: the point of the binding is that a token lifted off
// the wire is worth nothing outside the scope it was minted for.
//
// The scope rather than the address, for the same reason the throttle uses it:
// the two must cover the same ground, or a v6 caller sheds a spent bucket by
// re-minting on the next address in a prefix it already owns. It also stops an
// IPv6 privacy address rotating under a player mid-session, which on an exact
// binding would have logged them out on their own connection's schedule.
func (s *Signer) Mint(ip string, now time.Time) (Token, error) {
	id := make([]byte, idLen)
	if _, err := rand.Read(id); err != nil {
		return Token{}, fmt.Errorf("failed to read random bytes: %w", err)
	}

	expiresAt := now.Add(s.ttl)

	payload := make([]byte, expiryLen+idLen)
	binary.BigEndian.PutUint64(payload[:expiryLen], uint64(expiresAt.UnixMilli()))
	copy(payload[expiryLen:], id)

	token := append(payload, s.mac(payload, ip)...)

	return Token{
		Value:     base64.RawURLEncoding.EncodeToString(token),
		ID:        ID(hex.EncodeToString(id)),
		ExpiresAt: expiresAt,
	}, nil
}

func (s *Signer) Verify(value string, ip string, now time.Time) (ID, error) {
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrMalformed, err)
	}

	if len(raw) != tokenLen {
		return "", fmt.Errorf("%w: got %d bytes, want %d", ErrMalformed, len(raw), tokenLen)
	}

	payload, mac := raw[:expiryLen+idLen], raw[expiryLen+idLen:]

	// Constant time, and before the expiry check: an attacker must not learn
	// whether a forged token would have been in date.
	if !hmac.Equal(mac, s.mac(payload, ip)) {
		return "", ErrBadSignature
	}

	expiresAt := time.UnixMilli(int64(binary.BigEndian.Uint64(payload[:expiryLen])))
	if !now.Before(expiresAt) {
		return "", fmt.Errorf("%w at %s", ErrExpired, expiresAt.UTC().Format(time.RFC3339))
	}

	return ID(hex.EncodeToString(payload[expiryLen:])), nil
}

// mac binds the token to the caller's scope rather than its exact address.
// Normalising here rather than in Mint and Verify is what makes the two
// incapable of disagreeing: a token is verified under the same key it was
// minted under, by construction.
func (s *Signer) mac(payload []byte, ip string) []byte {
	h := hmac.New(sha256.New, s.key)
	h.Write(payload)
	h.Write([]byte(cpipscope.Of(ip)))
	return h.Sum(nil)
}
