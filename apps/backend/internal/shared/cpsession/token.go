// Package cpsession mints and verifies the opaque token a caller has to hold to
// click. It is deliberately stateless: the token carries its own expiry and a
// signature over it, so nothing has to be stored, swept, or replicated, and a
// restart does not log every player out.
//
// The signature is Ed25519 rather than a MAC, so the key that mints and the key
// that verifies are different halves. Only the auth context holds the seed; the
// planet context is handed a public key and a Verifier with no Mint on it.
package cpsession

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpipscope"
)

var (
	ErrMalformed    = errors.New("malformed session token")
	ErrBadSignature = errors.New("session token signature does not match")
	ErrExpired      = errors.New("session token has expired")
)

// Version is the first byte of every token, so a later format change can be
// accepted alongside this one instead of making every token in flight malformed.
const Version = 1

const (
	versionLen = 1
	expiryLen  = 8
	idLen      = 8
	accountLen = len(uuid.UUID{})

	expiryAt  = versionLen
	idAt      = expiryAt + expiryLen
	accountAt = idAt + idLen

	payloadLen = versionLen + expiryLen + idLen + accountLen
	tokenLen   = payloadLen + ed25519.SignatureSize
)

// ID identifies one minted session. It is not a secret and not an identity —
// it exists so a refusal can be correlated in the logs with the mint that
// preceded it.
type ID string

// Claims is what a verified token says: which mint, and which account it was minted for (uuid.Nil for none).
type Claims struct {
	ID      ID
	Account uuid.UUID
}

type Token struct {
	Value     string
	ID        ID
	ExpiresAt time.Time
}

// Signer mints. It is the auth context's, and nothing else builds one.
type Signer struct {
	key ed25519.PrivateKey
	ttl time.Duration
}

func NewSigner(config SignerConfig) (*Signer, error) {
	config = config.withDefaults()

	key, err := parseSeed(config.Secret)
	if err != nil {
		return nil, err
	}
	if config.TTL <= 0 {
		return nil, fmt.Errorf("session ttl must be positive, got %s", config.TTL)
	}

	return &Signer{key: key, ttl: config.TTL}, nil
}

// Mint binds the token to the scope ip sits in — the address itself over IPv4,
// the surrounding /64 over IPv6. A caller that leaves it mid-session — a phone
// moving from wifi to cellular — fails verification and mints again, which is
// the intended behaviour: the point of the binding is that a token lifted off
// the wire is worth nothing outside the scope it was minted for.
//
// The scope rather than the address, for the same reason the throttle uses it:
// the two must cover the same ground, or a v6 caller sheds a spent bucket by
// re-minting on the next address in a prefix it already owns.
func (s *Signer) Mint(ip string, account uuid.UUID, now time.Time) (*Token, error) {
	id := make([]byte, idLen)
	if _, err := rand.Read(id); err != nil {
		return nil, fmt.Errorf("failed to read random bytes: %w", err)
	}

	expiresAt := now.Add(s.ttl)

	payload := make([]byte, payloadLen)
	payload[0] = Version
	binary.BigEndian.PutUint64(payload[expiryAt:idAt], uint64(expiresAt.UnixMilli()))
	copy(payload[idAt:accountAt], id)
	copy(payload[accountAt:], account[:])

	token := make([]byte, 0, tokenLen)
	token = append(token, payload...)
	token = append(token, ed25519.Sign(s.key, signed(payload, ip))...)

	return &Token{
		Value:     base64.RawURLEncoding.EncodeToString(token),
		ID:        ID(hex.EncodeToString(id)),
		ExpiresAt: expiresAt,
	}, nil
}

// Verifier checks, and that is the whole of it: it holds a public key, so the
// context that has one cannot mint whatever it does with it.
type Verifier struct {
	key ed25519.PublicKey
}

func NewVerifier(config VerifierConfig) (*Verifier, error) {
	key, err := parsePublicKey(config.PublicKey)
	if err != nil {
		return nil, err
	}

	return &Verifier{key: key}, nil
}

func (v *Verifier) Verify(value string, ip string, now time.Time) (*Claims, error) {
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrMalformed, err)
	}

	if len(raw) != tokenLen {
		return nil, fmt.Errorf("%w: got %d bytes, want %d", ErrMalformed, len(raw), tokenLen)
	}
	if raw[0] != Version {
		return nil, fmt.Errorf("%w: token version %d, this server mints %d", ErrMalformed, raw[0], Version)
	}

	payload, signature := raw[:payloadLen], raw[payloadLen:]

	// Before the expiry check: a forger must not learn whether their token would otherwise have been in date.
	if !ed25519.Verify(v.key, signed(payload, ip), signature) {
		return nil, ErrBadSignature
	}

	expiresAt := time.UnixMilli(int64(binary.BigEndian.Uint64(payload[expiryAt:idAt])))
	if !now.Before(expiresAt) {
		return nil, fmt.Errorf("%w at %s", ErrExpired, expiresAt.UTC().Format(time.RFC3339))
	}

	return &Claims{
		ID:      ID(hex.EncodeToString(payload[idAt:accountAt])),
		Account: uuid.UUID(payload[accountAt:]),
	}, nil
}

// signed is what the signature covers: the payload as it travels, and the scope,
// which never does — so the token is bound to an address without carrying one.
// Normalising here is what makes Mint and Verify incapable of disagreeing.
func signed(payload []byte, ip string) []byte {
	scope := cpipscope.Of(ip)

	message := make([]byte, 0, len(payload)+len(scope))
	message = append(message, payload...)
	message = append(message, scope...)

	return message
}
