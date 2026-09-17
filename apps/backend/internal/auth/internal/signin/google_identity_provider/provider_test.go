package google_identity_provider_test

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin/google_identity_provider"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const redirect = "https://clickplanet.lol/auth/callback"

var (
	now    = time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	client = signin.Client{ClientID: "the-client", ClientSecret: "the-secret"}
	flow   = &signin.Flow{Provider: "google", State: "the-state", Verifier: "the-verifier", Nonce: "the-nonce"}
)

func idToken(t *testing.T, claims map[string]any) string {
	t.Helper()

	payload, err := json.Marshal(claims)
	require.NoError(t, err)
	return "e30." + base64.RawURLEncoding.EncodeToString(payload) + ".unchecked"
}

func goodClaims() map[string]any {
	return map[string]any{
		"iss": "https://accounts.google.com", "aud": "the-client", "exp": now.Add(time.Hour).Unix(),
		"nonce": "the-nonce", "sub": "1234", "email": "a@example.com", "email_verified": true,
	}
}

// google answers the token endpoint with idToken, and records the form it was sent.
func google(t *testing.T, status int, body map[string]any) (*google_identity_provider.Provider, *url.Values) {
	t.Helper()

	sent := &url.Values{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.NoError(t, r.ParseForm())
		*sent = r.PostForm
		w.WriteHeader(status)
		assert.NoError(t, json.NewEncoder(w).Encode(body))
	}))
	t.Cleanup(server.Close)

	endpoints := google_identity_provider.Endpoints{Authorize: "https://accounts.example/auth", Token: server.URL}
	return google_identity_provider.New(client, redirect, endpoints, server.Client(), cptime.NewFixedClock(now)), sent
}

func TestTheAuthorizationURLAsksForOpenIDAndEmailWithPKCE(t *testing.T) {
	provider, _ := google(t, http.StatusOK, nil)

	authorization, err := url.Parse(provider.AuthorizationURL(flow))
	require.NoError(t, err)

	assert.Equal(t, "accounts.example", authorization.Host)
	assert.Equal(t, url.Values{
		"client_id": {"the-client"}, "redirect_uri": {redirect}, "response_type": {"code"}, "scope": {"openid email"},
		"state": {"the-state"}, "nonce": {"the-nonce"}, "code_challenge": {flow.Challenge()}, "code_challenge_method": {"S256"},
		"prompt": {"select_account"},
	}, authorization.Query())
}

func TestTheCodeIsTradedForTheUserInTheIDToken(t *testing.T) {
	provider, sent := google(t, http.StatusOK, map[string]any{"access_token": "at", "id_token": idToken(t, goodClaims())})

	claim, err := provider.Exchange(t.Context(), "the-code", flow)

	require.NoError(t, err)
	assert.Equal(t, &accounts.Claim{Subject: "1234", Email: "a@example.com", EmailVerified: true}, claim)
	assert.Equal(t, url.Values{
		"grant_type": {"authorization_code"}, "code": {"the-code"}, "client_id": {"the-client"},
		"client_secret": {"the-secret"}, "redirect_uri": {redirect}, "code_verifier": {"the-verifier"},
	}, *sent)
}

func TestAnIDTokenNotMeantForThisSignInIsRefused(t *testing.T) {
	for name, change := range map[string]func(map[string]any){
		"another issuer":   func(c map[string]any) { c["iss"] = "https://evil.example" },
		"another audience": func(c map[string]any) { c["aud"] = []string{"another-client"} },
		"expired":          func(c map[string]any) { c["exp"] = now.Unix() },
		"another nonce":    func(c map[string]any) { c["nonce"] = "replayed" },
		"no subject":       func(c map[string]any) { delete(c, "sub") },
	} {
		t.Run(name, func(t *testing.T) {
			claims := goodClaims()
			change(claims)
			provider, _ := google(t, http.StatusOK, map[string]any{"access_token": "at", "id_token": idToken(t, claims)})

			claim, err := provider.Exchange(t.Context(), "the-code", flow)

			require.ErrorIs(t, err, signin.ErrProviderRefused)
			assert.Nil(t, claim)
		})
	}
}

func TestAnAudienceListNamingThisClientIsAccepted(t *testing.T) {
	claims := goodClaims()
	claims["aud"] = []string{"another-client", "the-client"}
	provider, _ := google(t, http.StatusOK, map[string]any{"access_token": "at", "id_token": idToken(t, claims)})

	_, err := provider.Exchange(t.Context(), "the-code", flow)

	assert.NoError(t, err)
}

func TestARefusedCodeIsRefusedAndAnOutageIsNot(t *testing.T) {
	refused, _ := google(t, http.StatusBadRequest, map[string]any{"error": "invalid_grant"})
	_, err := refused.Exchange(t.Context(), "the-code", flow)
	require.ErrorIs(t, err, signin.ErrProviderRefused)
	assert.Contains(t, err.Error(), "invalid_grant")

	down, _ := google(t, http.StatusBadGateway, map[string]any{})
	_, err = down.Exchange(t.Context(), "the-code", flow)
	require.Error(t, err)
	assert.NotErrorIs(t, err, signin.ErrProviderRefused, "an outage is this server's error to report, not the player's")
}
