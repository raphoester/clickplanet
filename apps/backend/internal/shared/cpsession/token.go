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

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpipscope"
)

var (
	ErrMalformed    = errors.New("malformed session token")
	ErrBadSignature = errors.New("session token signature does not match")
	ErrExpired      = errors.New("session token has expired")
)

const Version = 2

const versionWithoutLinked = 1

const (
	versionLen = 1
	expiryLen  = 8
	idLen      = 8
	accountLen = len(AccountID{})
	linkedLen  = 1

	expiryAt  = versionLen
	idAt      = expiryAt + expiryLen
	accountAt = idAt + idLen
	linkedAt  = accountAt + accountLen

	payloadLen = versionLen + expiryLen + idLen + accountLen + linkedLen
	tokenLen   = payloadLen + ed25519.SignatureSize

	payloadWithoutLinkedLen = payloadLen - linkedLen
	tokenWithoutLinkedLen   = payloadWithoutLinkedLen + ed25519.SignatureSize
)

type ID string

type Holder struct {
	Account AccountID
	Linked  bool
}

var Nobody = Holder{Account: NoAccount}

type Claims struct {
	ID      ID
	Account AccountID
	Linked  bool
}

type Token struct {
	Value     string
	ID        ID
	ExpiresAt time.Time
}

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

func (s *Signer) Mint(ip string, holder Holder, now time.Time) (*Token, error) {
	id := make([]byte, idLen)
	if _, err := rand.Read(id); err != nil {
		return nil, fmt.Errorf("failed to read random bytes: %w", err)
	}

	expiresAt := now.Add(s.ttl)

	payload := make([]byte, payloadLen)
	payload[0] = Version
	binary.BigEndian.PutUint64(payload[expiryAt:idAt], uint64(expiresAt.UnixMilli()))
	copy(payload[idAt:accountAt], id)
	copy(payload[accountAt:linkedAt], holder.Account[:])
	if holder.Linked {
		payload[linkedAt] = 1
	}

	token := make([]byte, 0, tokenLen)
	token = append(token, payload...)
	token = append(token, ed25519.Sign(s.key, signed(payload, ip))...)

	return &Token{
		Value:     base64.RawURLEncoding.EncodeToString(token),
		ID:        ID(hex.EncodeToString(id)),
		ExpiresAt: expiresAt,
	}, nil
}

func (s *Signer) PublicKey() string {
	return hex.EncodeToString(s.key.Public().(ed25519.PublicKey))
}

type Verifier struct {
	key ed25519.PublicKey
}

func NewVerifier(publicKey string) (*Verifier, error) {
	key, err := parsePublicKey(publicKey)
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

	if len(raw) == 0 {
		return nil, fmt.Errorf("%w: the token is empty", ErrMalformed)
	}

	length := tokenLen
	switch raw[0] {
	case Version:
	case versionWithoutLinked:
		length = tokenWithoutLinkedLen
	default:
		return nil, fmt.Errorf("%w: token version %d, this server mints %d", ErrMalformed, raw[0], Version)
	}
	if len(raw) != length {
		return nil, fmt.Errorf("%w: got %d bytes, want %d", ErrMalformed, len(raw), length)
	}

	payload, signature := raw[:length-ed25519.SignatureSize], raw[length-ed25519.SignatureSize:]

	// Signature before expiry: a forger must not learn whether the token is in date.
	if !ed25519.Verify(v.key, signed(payload, ip), signature) {
		return nil, ErrBadSignature
	}

	expiresAt := time.UnixMilli(int64(binary.BigEndian.Uint64(payload[expiryAt:idAt])))
	if !now.Before(expiresAt) {
		return nil, fmt.Errorf("%w at %s", ErrExpired, expiresAt.UTC().Format(time.RFC3339))
	}

	return &Claims{
		ID:      ID(hex.EncodeToString(payload[idAt:accountAt])),
		Account: AccountID(payload[accountAt:linkedAt]),
		Linked:  len(payload) > linkedAt && payload[linkedAt] == 1,
	}, nil
}

func signed(payload []byte, ip string) []byte {
	scope := cpipscope.Of(ip)

	message := make([]byte, 0, len(payload)+len(scope))
	message = append(message, payload...)
	message = append(message, scope...)

	return message
}
