package cpsession_test

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

const ttl = time.Hour

var now = time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)

// Where the account and the expiry sit, for the tests that forge one.
const (
	expiryAt  = 1
	accountAt = 17
)

func keys(t *testing.T) (cpsession.SignerConfig, string) {
	t.Helper()

	secret, public := cpsession.TestKeyPair()

	return cpsession.SignerConfig{Enabled: true, Secret: secret, TTL: ttl}, public
}

func newPair(t *testing.T) (*cpsession.Signer, *cpsession.Verifier) {
	t.Helper()

	signing, public := keys(t)

	signer, err := cpsession.NewSigner(signing)
	require.NoError(t, err)
	verifier, err := cpsession.NewVerifier(public)
	require.NoError(t, err)

	return signer, verifier
}

func TestNewSignerRejectsASeedItCannotUse(t *testing.T) {
	signing, _ := keys(t)

	for name, secret := range map[string]string{
		"empty":      "",
		"not hex":    strings.Repeat("z", 64),
		"too short":  strings.Repeat("ab", 8),
		"too long":   strings.Repeat("ab", 64),
		"odd length": strings.Repeat("a", 63),
	} {
		t.Run(name, func(t *testing.T) {
			signing.Secret = secret
			_, err := cpsession.NewSigner(signing)
			assert.Error(t, err)
		})
	}
}

func TestNewSignerRejectsANegativeTTL(t *testing.T) {
	signing, _ := keys(t)
	signing.TTL = -1

	_, err := cpsession.NewSigner(signing)
	assert.Error(t, err)
}

func TestNewVerifierRejectsAPublicKeyItCannotUse(t *testing.T) {
	for name, public := range map[string]string{
		"empty":     "",
		"not hex":   strings.Repeat("z", 64),
		"too short": strings.Repeat("ab", 8),
	} {
		t.Run(name, func(t *testing.T) {
			_, err := cpsession.NewVerifier(public)
			assert.Error(t, err)
		})
	}
}

func TestAnUnsetTTLTakesTheDefault(t *testing.T) {
	signing, _ := keys(t)
	signing.TTL = 0

	signer, err := cpsession.NewSigner(signing)
	require.NoError(t, err)

	token, err := signer.Mint("203.0.113.7", cpsession.NoAccount, now)
	require.NoError(t, err)
	assert.Equal(t, now.Add(time.Hour), token.ExpiresAt)
}

func TestTheTwoHalvesOfOneBlockAgree(t *testing.T) {
	signer, verifier := newPair(t)

	token, err := signer.Mint("1.2.3.4", cpsession.NoAccount, now)
	require.NoError(t, err)

	_, err = verifier.Verify(token.Value, "1.2.3.4", now)
	assert.NoError(t, err, "planet builds its verifier from the public half of the block auth mints with")
}

func TestAMintedTokenVerifiesForTheAddressItWasMintedFor(t *testing.T) {
	signer, verifier := newPair(t)

	token, err := signer.Mint("203.0.113.7", cpsession.NoAccount, now)
	require.NoError(t, err)
	assert.Equal(t, now.Add(ttl), token.ExpiresAt)

	claims, err := verifier.Verify(token.Value, "203.0.113.7", now)
	require.NoError(t, err)
	assert.Equal(t, token.ID, claims.ID)
	assert.Equal(t, cpsession.NoAccount, claims.Account)
}

func TestATokenCarriesTheAccountItWasMintedFor(t *testing.T) {
	signer, verifier := newPair(t)
	account := cpsession.AccountID(uuid.MustParse("01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e5f"))

	token, err := signer.Mint("203.0.113.7", account, now)
	require.NoError(t, err)

	claims, err := verifier.Verify(token.Value, "203.0.113.7", now)
	require.NoError(t, err)
	assert.Equal(t, account, claims.Account)
}

func TestASwappedAccountDoesNotVerify(t *testing.T) {
	signer, verifier := newPair(t)

	token, err := signer.Mint("203.0.113.7", cpsession.AccountID(uuid.MustParse("01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e5f")), now)
	require.NoError(t, err)

	raw := decode(t, token.Value)
	raw[accountAt+3] ^= 0xff

	_, err = verifier.Verify(encode(raw), "203.0.113.7", now)
	assert.ErrorIs(t, err, cpsession.ErrBadSignature)
}

func TestATokenIsRefusedForAnotherAddress(t *testing.T) {
	signer, verifier := newPair(t)

	token, err := signer.Mint("203.0.113.7", cpsession.NoAccount, now)
	require.NoError(t, err)

	_, err = verifier.Verify(token.Value, "203.0.113.8", now)
	assert.ErrorIs(t, err, cpsession.ErrBadSignature)
}

func TestATokenIsRefusedOnceItHasExpired(t *testing.T) {
	signer, verifier := newPair(t)

	token, err := signer.Mint("203.0.113.7", cpsession.NoAccount, now)
	require.NoError(t, err)

	_, err = verifier.Verify(token.Value, "203.0.113.7", now.Add(ttl-time.Second))
	require.NoError(t, err)

	_, err = verifier.Verify(token.Value, "203.0.113.7", now.Add(ttl))
	assert.ErrorIs(t, err, cpsession.ErrExpired)
}

func TestATokenIsRefusedByAVerifierHoldingAnotherKey(t *testing.T) {
	signer, _ := newPair(t)

	token, err := signer.Mint("203.0.113.7", cpsession.NoAccount, now)
	require.NoError(t, err)

	// A second pair, from a seed that is not the test one.
	other, err := cpsession.NewVerifier("d75a980182b10ab7d54bfed3c964073a0ee172f3daa62325af021a68f707511a")
	require.NoError(t, err)

	_, err = other.Verify(token.Value, "203.0.113.7", now)
	assert.ErrorIs(t, err, cpsession.ErrBadSignature)
}

func TestAnExtendedExpiryDoesNotVerify(t *testing.T) {
	signer, verifier := newPair(t)

	token, err := signer.Mint("203.0.113.7", cpsession.NoAccount, now)
	require.NoError(t, err)

	// The expiry travels in the clear, so a caller reads it; what it must not do is push it out.
	raw := decode(t, token.Value)
	for i := expiryAt; i < expiryAt+8; i++ {
		raw[i] = 0x7f
	}

	_, err = verifier.Verify(encode(raw), "203.0.113.7", now.Add(10*ttl))
	assert.ErrorIs(t, err, cpsession.ErrBadSignature)
}

func TestATokenOfAnotherVersionIsMalformed(t *testing.T) {
	signer, verifier := newPair(t)

	token, err := signer.Mint("203.0.113.7", cpsession.NoAccount, now)
	require.NoError(t, err)

	// The byte that lets a later format be accepted beside this one instead of
	// making every token in flight unreadable.
	raw := decode(t, token.Value)
	raw[0] = cpsession.Version + 1

	_, err = verifier.Verify(encode(raw), "203.0.113.7", now)
	assert.ErrorIs(t, err, cpsession.ErrMalformed)
}

func TestMalformedTokensAreRefusedRatherThanPanicking(t *testing.T) {
	signer, verifier := newPair(t)

	valid, err := signer.Mint("203.0.113.7", cpsession.NoAccount, now)
	require.NoError(t, err)

	for name, value := range map[string]string{
		"empty":         "",
		"not base64":    "!!!!not-base64!!!!",
		"too short":     "AAAA",
		"truncated":     valid.Value[:len(valid.Value)-4],
		"padded longer": valid.Value + "AAAA",
		"whitespace":    strings.Repeat(" ", 130),
	} {
		t.Run(name, func(t *testing.T) {
			_, err := verifier.Verify(value, "203.0.113.7", now)
			assert.Error(t, err)
		})
	}
}

func TestTwoMintsProduceDifferentTokensAndIDs(t *testing.T) {
	signer, _ := newPair(t)

	first, err := signer.Mint("203.0.113.7", cpsession.NoAccount, now)
	require.NoError(t, err)

	second, err := signer.Mint("203.0.113.7", cpsession.NoAccount, now)
	require.NoError(t, err)

	assert.NotEqual(t, first.Value, second.Value)
	assert.NotEqual(t, first.ID, second.ID)
}

func TestASignerSaysWhatVerifiesIt(t *testing.T) {
	signing, public := keys(t)

	signer, err := cpsession.NewSigner(signing)
	require.NoError(t, err)

	// What auth answers over the internal listener: the half that is not a secret.
	assert.Equal(t, public, signer.PublicKey())
}

func TestADisabledBlockNeedsNoSeed(t *testing.T) {
	require.NoError(t, cpsession.SignerConfig{}.Validate())
	assert.ErrorContains(t, cpsession.SignerConfig{Enabled: true}.Validate(), "auth.secret is empty")
}

func TestAV6TokenVerifiesAcrossItsOwnPrefix(t *testing.T) {
	signer, verifier := newPair(t)

	// A privacy address rotating under a player must not log them out: the
	// binding is to the /64, which the caller has not left.
	token, err := signer.Mint("2001:db8:1:2::1", cpsession.NoAccount, now)
	require.NoError(t, err)

	claims, err := verifier.Verify(token.Value, "2001:db8:1:2:aaaa:bbbb:cccc:dddd", now)
	require.NoError(t, err)
	assert.Equal(t, token.ID, claims.ID)
}

func TestAV6TokenIsRefusedOutsideItsPrefix(t *testing.T) {
	signer, verifier := newPair(t)

	// The other half of the same rule: leaving the /64 is leaving the scope the
	// token was minted for, so it buys nothing there.
	token, err := signer.Mint("2001:db8:1:2::1", cpsession.NoAccount, now)
	require.NoError(t, err)

	_, err = verifier.Verify(token.Value, "2001:db8:1:3::1", now)
	assert.ErrorIs(t, err, cpsession.ErrBadSignature)
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
