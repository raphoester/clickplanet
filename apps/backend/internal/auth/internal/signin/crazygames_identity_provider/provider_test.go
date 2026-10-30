package crazygames_identity_provider_test

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin/crazygames_identity_provider"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const game = "clickplanet"

var now = time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC)

func newKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	return key
}

type portal struct {
	key     atomic.Pointer[rsa.PrivateKey]
	fetches atomic.Int32
	down    atomic.Bool
}

func (p *portal) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	p.fetches.Add(1)
	if p.down.Load() {
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	public := p.key.Load().PublicKey
	_ = json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]string{
		{"kty": "EC", "kid": "elsewhere", "crv": "P-256"},
		{
			"kty": "RSA", "kid": "ingame-auth", "use": "sig", "alg": "RS256",
			"n": base64.RawURLEncoding.EncodeToString(public.N.Bytes()),
			"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(public.E)).Bytes()),
		},
	}})
}

func crazyGames(t *testing.T, key *rsa.PrivateKey) (*crazygames_identity_provider.Provider, *portal, *cptime.FixedClock) {
	t.Helper()

	served := &portal{}
	served.key.Store(key)
	server := httptest.NewServer(served)
	t.Cleanup(server.Close)
	clock := cptime.NewFixedClock(now)
	keys := crazygames_identity_provider.NewKeySet(server.URL, server.Client(), clock)
	return crazygames_identity_provider.New(game, keys, clock), served, clock
}

func signed(t *testing.T, key *rsa.PrivateKey, head map[string]any, payload map[string]any) string {
	t.Helper()

	part := func(value map[string]any) string {
		raw, err := json.Marshal(value)
		require.NoError(t, err)
		return base64.RawURLEncoding.EncodeToString(raw)
	}
	unsigned := part(head) + "." + part(payload)
	digest := sha256.Sum256([]byte(unsigned))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	require.NoError(t, err)
	return unsigned + "." + base64.RawURLEncoding.EncodeToString(signature)
}

func goodHeader() map[string]any {
	return map[string]any{"alg": "RS256", "kid": "ingame-auth", "typ": "JWT"}
}

func goodPayload() map[string]any {
	return map[string]any{
		"userId": "cg-user-1", "gameId": game, "username": "Nova.42", "profilePictureUrl": "https://images.example/nova.png",
		"iat": now.Add(-time.Minute).Unix(), "exp": now.Add(59 * time.Minute).Unix(),
	}
}

func TestAGoodTokenNamesTheUserAndNoAddress(t *testing.T) {
	key := newKey(t)
	provider, _, _ := crazyGames(t, key)

	claim, err := provider.Claim(t.Context(), signed(t, key, goodHeader(), goodPayload()))

	require.NoError(t, err)
	assert.Equal(t, accounts.ClaimOf("cg-user-1", "", false), *claim)
}

func TestANumericUserIDIsReadAsItsDigits(t *testing.T) {
	key := newKey(t)
	provider, _, _ := crazyGames(t, key)
	payload := goodPayload()
	payload["userId"] = 12345678901

	claim, err := provider.Claim(t.Context(), signed(t, key, goodHeader(), payload))

	require.NoError(t, err)
	assert.Equal(t, "12345678901", claim.Subject())
}

func TestTheKeySetIsFetchedOnce(t *testing.T) {
	key := newKey(t)
	provider, served, clock := crazyGames(t, key)

	for range 3 {
		_, err := provider.Claim(t.Context(), signed(t, key, goodHeader(), goodPayload()))
		require.NoError(t, err)
		clock.Advance(10 * time.Minute)
	}

	assert.Equal(t, int32(1), served.fetches.Load())
}

func TestEveryBadTokenIsRefused(t *testing.T) {
	key := newKey(t)
	payload := func(change func(map[string]any)) map[string]any {
		claims := goodPayload()
		change(claims)
		return claims
	}
	header := func(change func(map[string]any)) map[string]any {
		head := goodHeader()
		change(head)
		return head
	}

	for name, token := range map[string]string{
		"not a JWT":          "not.a-jwt",
		"expired":            signed(t, key, goodHeader(), payload(func(c map[string]any) { c["exp"] = now.Unix() })),
		"for another game":   signed(t, key, goodHeader(), payload(func(c map[string]any) { c["gameId"] = "another" })),
		"names no user":      signed(t, key, goodHeader(), payload(func(c map[string]any) { delete(c, "userId") })),
		"signed by another":  signed(t, newKey(t), goodHeader(), goodPayload()),
		"the legacy key":     signed(t, key, header(func(h map[string]any) { h["kid"] = "ingame-auth-legacy" }), goodPayload()),
		"another algorithm":  signed(t, key, header(func(h map[string]any) { h["alg"] = "HS256" }), goodPayload()),
		"no algorithm":       signed(t, key, header(func(h map[string]any) { h["alg"] = "none" }), goodPayload()),
		"a payload swapped":  swapPayload(t, signed(t, key, goodHeader(), goodPayload()), payload(func(c map[string]any) { c["userId"] = "cg-user-2" })),
		"a signature erased": signed(t, key, goodHeader(), goodPayload())[:20] + "..",
	} {
		t.Run(name, func(t *testing.T) {
			provider, _, _ := crazyGames(t, key)

			_, err := provider.Claim(t.Context(), token)

			require.ErrorIs(t, err, signin.ErrProviderRefused)
		})
	}
}

func swapPayload(t *testing.T, token string, payload map[string]any) string {
	t.Helper()

	raw, err := json.Marshal(payload)
	require.NoError(t, err)
	parts := strings.Split(token, ".")
	return parts[0] + "." + base64.RawURLEncoding.EncodeToString(raw) + "." + parts[2]
}

func TestARotatedKeyIsFetchedAgainOnceAMinuteAtMost(t *testing.T) {
	old, rotated := newKey(t), newKey(t)
	provider, served, clock := crazyGames(t, old)
	_, err := provider.Claim(t.Context(), signed(t, old, goodHeader(), goodPayload()))
	require.NoError(t, err)

	served.key.Store(rotated)
	_, err = provider.Claim(t.Context(), signed(t, rotated, goodHeader(), goodPayload()))
	require.ErrorIs(t, err, signin.ErrProviderRefused, "the key set was fetched less than a minute ago")

	clock.Advance(time.Minute)
	_, err = provider.Claim(t.Context(), signed(t, rotated, goodHeader(), goodPayload()))
	require.NoError(t, err)
	assert.Equal(t, int32(2), served.fetches.Load())
}

func TestAPortalThatCannotBeReachedIsNoRefusal(t *testing.T) {
	key := newKey(t)
	provider, served, _ := crazyGames(t, key)
	served.down.Store(true)

	_, err := provider.Claim(t.Context(), signed(t, key, goodHeader(), goodPayload()))

	require.Error(t, err)
	assert.NotErrorIs(t, err, signin.ErrProviderRefused, "a refusal would tell the player their account is not theirs")
}
