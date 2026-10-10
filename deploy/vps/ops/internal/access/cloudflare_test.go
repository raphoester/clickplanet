package access_test

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-ops/internal/access"
)

const audience = "4714c1358e65fe4b408ad6d432a5f878f08194bdb4752441fd56faefa9b2b6f2"

type team struct {
	issuer string
	key    *rsa.PrivateKey
}

func startTeam(t *testing.T) team {
	t.Helper()

	key := newKey(t)
	certs, err := json.Marshal(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{
		{Key: &key.PublicKey, KeyID: "current", Algorithm: string(jose.RS256), Use: "sig"},
	}})
	require.NoError(t, err)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /cdn-cgi/access/certs", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(certs)
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return team{issuer: server.URL, key: key}
}

func newKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	return key
}

func (tm team) verifier(t *testing.T) *access.CloudflareVerifier {
	t.Helper()
	return access.NewCloudflareVerifier(t.Context(), access.Issuer(tm.issuer), audience)
}

func (tm team) claims() map[string]any {
	return map[string]any{
		"iss":         tm.issuer,
		"aud":         []string{audience},
		"iat":         time.Now().Add(-time.Minute).Unix(),
		"exp":         time.Now().Add(time.Hour).Unix(),
		"common_name": "88bf3b6d86161464f6509f7219099e57.access",
	}
}

func sign(t *testing.T, key *rsa.PrivateKey, claims map[string]any) string {
	t.Helper()

	signer, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.RS256, Key: key},
		(&jose.SignerOptions{}).WithHeader("kid", "current"),
	)
	require.NoError(t, err)
	payload, err := json.Marshal(claims)
	require.NoError(t, err)
	signed, err := signer.Sign(payload)
	require.NoError(t, err)
	assertion, err := signed.CompactSerialize()
	require.NoError(t, err)
	return assertion
}

func TestAServiceTokenIsNamedByItsClientID(t *testing.T) {
	team := startTeam(t)

	caller, err := team.verifier(t).Verify(t.Context(), sign(t, team.key, team.claims()))

	require.NoError(t, err)
	assert.Equal(t, access.Caller("88bf3b6d86161464f6509f7219099e57.access"), caller)
}

func TestAPersonIsNamedByTheirEmail(t *testing.T) {
	team := startTeam(t)
	claims := team.claims()
	delete(claims, "common_name")
	claims["email"] = "operator@example.com"

	caller, err := team.verifier(t).Verify(t.Context(), sign(t, team.key, claims))

	require.NoError(t, err)
	assert.Equal(t, access.Caller("operator@example.com"), caller)
}

func TestAnAssertionMadeForAnotherApplicationIsRefused(t *testing.T) {
	team := startTeam(t)
	claims := team.claims()
	claims["aud"] = []string{"another-application"}

	_, err := team.verifier(t).Verify(t.Context(), sign(t, team.key, claims))

	require.Error(t, err)
}

func TestAnAssertionFromAnotherTeamIsRefused(t *testing.T) {
	team := startTeam(t)
	claims := team.claims()
	claims["iss"] = "https://someone-else.cloudflareaccess.com"

	_, err := team.verifier(t).Verify(t.Context(), sign(t, team.key, claims))

	require.Error(t, err)
}

func TestAnExpiredAssertionIsRefused(t *testing.T) {
	team := startTeam(t)
	claims := team.claims()
	claims["exp"] = time.Now().Add(-time.Minute).Unix()

	_, err := team.verifier(t).Verify(t.Context(), sign(t, team.key, claims))

	require.Error(t, err)
}

func TestAnAssertionSignedWithAnotherKeyIsRefused(t *testing.T) {
	team := startTeam(t)

	_, err := team.verifier(t).Verify(t.Context(), sign(t, newKey(t), team.claims()))

	require.Error(t, err)
}

func TestAnAssertionThatNamesNobodyIsRefused(t *testing.T) {
	team := startTeam(t)
	claims := team.claims()
	delete(claims, "common_name")

	_, err := team.verifier(t).Verify(t.Context(), sign(t, team.key, claims))

	require.ErrorIs(t, err, access.ErrAnonymous)
}

func TestNoAssertionIsRefused(t *testing.T) {
	team := startTeam(t)

	_, err := team.verifier(t).Verify(t.Context(), "")

	require.Error(t, err)
}
