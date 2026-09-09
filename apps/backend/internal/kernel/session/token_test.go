package session_test

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/session"
)

const (
	secret = "a-test-secret"
	ttl    = time.Hour
)

var now = time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)

func newSigner(t *testing.T) *session.Signer {
	t.Helper()
	signer, err := session.NewSigner(secret, ttl)
	require.NoError(t, err)
	return signer
}

func TestNewSignerRejectsAnEmptySecretAndANonPositiveTTL(t *testing.T) {
	_, err := session.NewSigner("", ttl)
	assert.Error(t, err)

	_, err = session.NewSigner(secret, 0)
	assert.Error(t, err)
}

func TestAMintedTokenVerifiesForTheAddressItWasMintedFor(t *testing.T) {
	signer := newSigner(t)

	token, err := signer.Mint("203.0.113.7", now)
	require.NoError(t, err)
	assert.Equal(t, now.Add(ttl), token.ExpiresAt)

	id, err := signer.Verify(token.Value, "203.0.113.7", now)
	require.NoError(t, err)
	assert.Equal(t, token.ID, id)
}

func TestATokenIsRefusedForAnotherAddress(t *testing.T) {
	signer := newSigner(t)

	token, err := signer.Mint("203.0.113.7", now)
	require.NoError(t, err)

	_, err = signer.Verify(token.Value, "203.0.113.8", now)
	assert.ErrorIs(t, err, session.ErrBadSignature)
}

func TestATokenIsRefusedOnceItHasExpired(t *testing.T) {
	signer := newSigner(t)

	token, err := signer.Mint("203.0.113.7", now)
	require.NoError(t, err)

	_, err = signer.Verify(token.Value, "203.0.113.7", now.Add(ttl-time.Second))
	assert.NoError(t, err)

	_, err = signer.Verify(token.Value, "203.0.113.7", now.Add(ttl))
	assert.ErrorIs(t, err, session.ErrExpired)
}

func TestATokenIsRefusedByASignerHoldingAnotherSecret(t *testing.T) {
	signer := newSigner(t)

	token, err := signer.Mint("203.0.113.7", now)
	require.NoError(t, err)

	other, err := session.NewSigner("another-secret", ttl)
	require.NoError(t, err)

	_, err = other.Verify(token.Value, "203.0.113.7", now)
	assert.ErrorIs(t, err, session.ErrBadSignature)
}

// The expiry travels in the clear, so a caller can read it. What it must not be
// able to do is push it out and still verify.
func TestAnExtendedExpiryDoesNotVerify(t *testing.T) {
	signer := newSigner(t)

	token, err := signer.Mint("203.0.113.7", now)
	require.NoError(t, err)

	forged := extendExpiry(t, token.Value)

	_, err = signer.Verify(forged, "203.0.113.7", now.Add(10*ttl))
	assert.ErrorIs(t, err, session.ErrBadSignature)
}

func TestMalformedTokensAreRefusedRatherThanPanicking(t *testing.T) {
	signer := newSigner(t)

	valid, err := signer.Mint("203.0.113.7", now)
	require.NoError(t, err)

	for name, value := range map[string]string{
		"empty":         "",
		"not base64":    "!!!!not-base64!!!!",
		"too short":     "AAAA",
		"truncated":     valid.Value[:len(valid.Value)-4],
		"padded longer": valid.Value + "AAAA",
		"whitespace":    strings.Repeat(" ", 64),
	} {
		t.Run(name, func(t *testing.T) {
			_, err := signer.Verify(value, "203.0.113.7", now)
			assert.Error(t, err)
		})
	}
}

func TestTwoMintsProduceDifferentTokensAndIDs(t *testing.T) {
	signer := newSigner(t)

	first, err := signer.Mint("203.0.113.7", now)
	require.NoError(t, err)

	second, err := signer.Mint("203.0.113.7", now)
	require.NoError(t, err)

	assert.NotEqual(t, first.Value, second.Value)
	assert.NotEqual(t, first.ID, second.ID)
}

func extendExpiry(t *testing.T, value string) string {
	t.Helper()

	raw := decode(t, value)
	for i := 0; i < 8; i++ {
		raw[i] = 0x7f
	}
	return encode(raw)
}

func decode(t *testing.T, value string) []byte {
	t.Helper()
	raw, err := base64.RawURLEncoding.DecodeString(value)
	require.NoError(t, err)
	return raw
}

func encode(raw []byte) string {
	return base64.RawURLEncoding.EncodeToString(raw)
}
