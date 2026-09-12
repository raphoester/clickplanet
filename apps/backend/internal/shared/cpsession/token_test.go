package cpsession_test

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

const (
	secret = "a-test-secret"
	ttl    = time.Hour
)

var now = time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)

func newSigner(t *testing.T) *cpsession.Signer {
	t.Helper()
	signer, err := cpsession.NewSigner(cpsession.Config{Secret: secret, TTL: ttl})
	require.NoError(t, err)
	return signer
}

func TestNewSignerRejectsAnEmptySecretAndANegativeTTL(t *testing.T) {
	_, err := cpsession.NewSigner(cpsession.Config{Secret: "", TTL: ttl})
	assert.Error(t, err)

	_, err = cpsession.NewSigner(cpsession.Config{Secret: secret, TTL: -1})
	assert.Error(t, err)
}

func TestAnUnsetTTLTakesTheDefault(t *testing.T) {
	signer, err := cpsession.NewSigner(cpsession.Config{Secret: secret})
	require.NoError(t, err)
	assert.Equal(t, time.Hour, signer.TTL())
}

func TestTwoSignersOverOneConfigAgree(t *testing.T) {
	config := cpsession.Config{Secret: secret, TTL: ttl}

	minter, err := cpsession.NewSigner(config)
	require.NoError(t, err)
	verifier, err := cpsession.NewSigner(config)
	require.NoError(t, err)

	token, err := minter.Mint("1.2.3.4", now)
	require.NoError(t, err)

	_, err = verifier.Verify(token.Value, "1.2.3.4", now)
	assert.NoError(t, err, "the planet context builds its own signer from the same block")
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
	assert.ErrorIs(t, err, cpsession.ErrBadSignature)
}

func TestATokenIsRefusedOnceItHasExpired(t *testing.T) {
	signer := newSigner(t)

	token, err := signer.Mint("203.0.113.7", now)
	require.NoError(t, err)

	_, err = signer.Verify(token.Value, "203.0.113.7", now.Add(ttl-time.Second))
	assert.NoError(t, err)

	_, err = signer.Verify(token.Value, "203.0.113.7", now.Add(ttl))
	assert.ErrorIs(t, err, cpsession.ErrExpired)
}

func TestATokenIsRefusedByASignerHoldingAnotherSecret(t *testing.T) {
	signer := newSigner(t)

	token, err := signer.Mint("203.0.113.7", now)
	require.NoError(t, err)

	other, err := cpsession.NewSigner(cpsession.Config{Secret: "another-secret", TTL: ttl})
	require.NoError(t, err)

	_, err = other.Verify(token.Value, "203.0.113.7", now)
	assert.ErrorIs(t, err, cpsession.ErrBadSignature)
}

// The expiry travels in the clear, so a caller can read it. What it must not be
// able to do is push it out and still verify.
func TestAnExtendedExpiryDoesNotVerify(t *testing.T) {
	signer := newSigner(t)

	token, err := signer.Mint("203.0.113.7", now)
	require.NoError(t, err)

	forged := extendExpiry(t, token.Value)

	_, err = signer.Verify(forged, "203.0.113.7", now.Add(10*ttl))
	assert.ErrorIs(t, err, cpsession.ErrBadSignature)
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

func TestAV6TokenVerifiesAcrossItsOwnPrefix(t *testing.T) {
	signer := newSigner(t)

	// A privacy address rotating under a player must not log them out: the
	// binding is to the /64, which the caller has not left.
	token, err := signer.Mint("2001:db8:1:2::1", now)
	require.NoError(t, err)

	id, err := signer.Verify(token.Value, "2001:db8:1:2:aaaa:bbbb:cccc:dddd", now)
	require.NoError(t, err)
	assert.Equal(t, token.ID, id)
}

func TestAV6TokenIsRefusedOutsideItsPrefix(t *testing.T) {
	signer := newSigner(t)

	// The other half of the same rule: leaving the /64 is leaving the scope the
	// token was minted for, so it buys nothing there.
	token, err := signer.Mint("2001:db8:1:2::1", now)
	require.NoError(t, err)

	_, err = signer.Verify(token.Value, "2001:db8:1:3::1", now)
	assert.ErrorIs(t, err, cpsession.ErrBadSignature)
}
