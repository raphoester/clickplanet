package discord_identity_provider_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin/discord_identity_provider"
)

const redirect = "https://clickplanet.lol/auth/callback"

var (
	client = signin.Client{ClientID: "the-client", ClientSecret: "the-secret"}
	flow   = &signin.Flow{Provider: "discord", State: "the-state", Verifier: "the-verifier", Nonce: "unused"}
)

type discord struct {
	tokenStatus int
	token       map[string]any
	me          map[string]any
	sentForm    url.Values
	sentBearer  string
}

func (d *discord) provider(t *testing.T) *discord_identity_provider.Provider {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /token", func(w http.ResponseWriter, r *http.Request) {
		assert.NoError(t, r.ParseForm())
		d.sentForm = r.PostForm
		w.WriteHeader(d.tokenStatus)
		assert.NoError(t, json.NewEncoder(w).Encode(d.token))
	})
	mux.HandleFunc("GET /me", func(w http.ResponseWriter, r *http.Request) {
		d.sentBearer = r.Header.Get("Authorization")
		assert.NoError(t, json.NewEncoder(w).Encode(d.me))
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	endpoints := discord_identity_provider.Endpoints{Authorize: "https://discord.example/authorize", Token: server.URL + "/token", Me: server.URL + "/me"}
	return discord_identity_provider.New(client, redirect, endpoints, server.Client())
}

func TestTheAuthorizationURLAsksForIdentifyAndEmailWithPKCE(t *testing.T) {
	authorization, err := url.Parse((&discord{}).provider(t).AuthorizationURL(flow))
	require.NoError(t, err)

	assert.Equal(t, url.Values{
		"client_id": {"the-client"}, "redirect_uri": {redirect}, "response_type": {"code"}, "scope": {"identify email"},
		"state": {"the-state"}, "code_challenge": {flow.Challenge()}, "code_challenge_method": {"S256"},
	}, authorization.Query())
}

func TestTheCodeIsTradedForTheUserBehindIt(t *testing.T) {
	d := &discord{
		tokenStatus: http.StatusOK,
		token:       map[string]any{"access_token": "the-access-token"},
		me:          map[string]any{"id": "80351110224678912", "email": "a@example.com", "verified": true},
	}

	claim, err := d.provider(t).Exchange(t.Context(), "the-code", flow)

	require.NoError(t, err)
	assert.Equal(t, &accounts.Claim{Subject: "80351110224678912", Email: "a@example.com", EmailVerified: true}, claim)
	assert.Equal(t, "the-verifier", d.sentForm.Get("code_verifier"))
	assert.Equal(t, "the-secret", d.sentForm.Get("client_secret"))
	assert.Equal(t, "Bearer the-access-token", d.sentBearer)
}

func TestAnUnverifiedEmailIsSaidToBeUnverified(t *testing.T) {
	d := &discord{tokenStatus: http.StatusOK, token: map[string]any{"access_token": "at"}, me: map[string]any{"id": "1", "email": "a@example.com"}}

	claim, err := d.provider(t).Exchange(t.Context(), "the-code", flow)

	require.NoError(t, err)
	assert.False(t, claim.EmailVerified)
}

func TestARefusedCodeOrANamelessUserIsRefused(t *testing.T) {
	for name, d := range map[string]*discord{
		"refused code":    {tokenStatus: http.StatusBadRequest, token: map[string]any{"error": "invalid_grant"}},
		"no access token": {tokenStatus: http.StatusOK, token: map[string]any{}},
		"no user id":      {tokenStatus: http.StatusOK, token: map[string]any{"access_token": "at"}, me: map[string]any{}},
	} {
		t.Run(name, func(t *testing.T) {
			claim, err := d.provider(t).Exchange(t.Context(), "the-code", flow)

			require.ErrorIs(t, err, signin.ErrProviderRefused)
			assert.Nil(t, claim)
		})
	}
}
