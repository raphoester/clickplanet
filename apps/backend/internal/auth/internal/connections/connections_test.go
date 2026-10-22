package connections_test

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/admintokens"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/connections"
)

const callback = "https://claude.ai/api/mcp/auth_callback"

var (
	now     = time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
	allowed = connections.Redirects{callback, "https://claude.com/api/mcp/auth_callback"}
	ada     = accounts.AccountID{15: 1}
)

func claude(t *testing.T) *connections.Client {
	t.Helper()
	client, err := connections.NewClient("Claude", []string{callback}, allowed, now)
	require.NoError(t, err)
	return client
}

func ask() connections.Ask {
	return connections.Ask{
		ResponseType: "code", RedirectURI: callback, Challenge: connections.TestChallenge(), ChallengeMethod: "S256", State: "the state",
	}
}

func TestOnlyAnAllowedRedirectIsRegistered(t *testing.T) {
	for name, uris := range map[string][]string{
		"none":        nil,
		"elsewhere":   {"https://evil.example/callback"},
		"one of them": {callback, "https://evil.example/callback"},
		"a lookalike": {callback + "/"},
		"too many":    {callback, callback, callback, callback, callback, callback},
	} {
		_, err := connections.NewClient("Claude", uris, allowed, now)
		require.ErrorIs(t, err, connections.ErrInvalidRedirect, name)
	}
}

func TestANameIsOneLineShownAsItWasGiven(t *testing.T) {
	for given, kept := range map[string]string{
		"Claude":                  "Claude",
		"  Claude\n\x00":          "Claude",
		"Cla\u200bude":            "Claude",
		"":                        "an app",
		strings.Repeat("é", 80):   strings.Repeat("é", 64),
		"Claude\nFAKE LOG LINE=1": "ClaudeFAKE LOG LINE=1",
	} {
		client, err := connections.NewClient(given, []string{callback}, allowed, now)
		require.NoError(t, err)
		assert.Equal(t, kept, client.TakeSnapshot().Name, given)
	}
}

func TestARequestIsSentBackToTheAppWhenOnlyItIsWrong(t *testing.T) {
	client := claude(t)

	elsewhere := ask()
	elsewhere.RedirectURI = "https://claude.com/api/mcp/auth_callback"
	_, err := client.Request("the id", elsewhere, now)
	require.ErrorIs(t, err, connections.ErrInvalidRedirect, "allowed, but not registered by this client")

	for name, c := range map[string]struct {
		change func(*connections.Ask)
		error  string
	}{
		"a token":      {func(a *connections.Ask) { a.ResponseType = "token" }, "unsupported_response_type"},
		"no PKCE":      {func(a *connections.Ask) { a.Challenge, a.ChallengeMethod = "", "" }, "invalid_request"},
		"plain PKCE":   {func(a *connections.Ask) { a.ChallengeMethod = "plain" }, "invalid_request"},
		"a short one":  {func(a *connections.Ask) { a.Challenge = "short" }, "invalid_request"},
		"not base64ur": {func(a *connections.Ask) { a.Challenge = strings.Repeat("+", 43) }, "invalid_request"},
	} {
		asked := ask()
		c.change(&asked)

		_, err := client.Request("the id", asked, now)

		var refusal *connections.Refusal
		require.ErrorAs(t, err, &refusal, name)
		require.ErrorIs(t, err, connections.ErrInvalidRequest, name)
		sent, parseErr := url.Parse(refusal.URL)
		require.NoError(t, parseErr)
		assert.Equal(t, callback, sent.Scheme+"://"+sent.Host+sent.Path, name)
		assert.Equal(t, url.Values{"error": {c.error}, "state": {"the state"}}, sent.Query(), name)
	}
}

func TestARequestLastsTenMinutesAndSaysWhoAndWhere(t *testing.T) {
	request, err := claude(t).Request("the id", ask(), now)
	require.NoError(t, err)

	require.NoError(t, request.ExpiryError(now.Add(connections.RequestTTL-time.Nanosecond)))
	require.ErrorIs(t, request.ExpiryError(now.Add(connections.RequestTTL)), connections.ErrExpired)
	assert.Equal(t, "Claude", request.Client())
	assert.Equal(t, "claude.ai", request.RedirectHost())
	assert.Equal(t, callback+"?error=access_denied&state=the+state", request.DeniedURL())
	assert.Equal(t, callback+"?code=the+code&iss=https%3A%2F%2Fapi.clickplanet.lol&state=the+state",
		request.ApprovedURL("the code", "https://api.clickplanet.lol"))
}

func TestACodeIsRedeemedByItsClientOnItsRedirectWithItsVerifierWithinAMinute(t *testing.T) {
	request, err := claude(t).Request("the id", ask(), now)
	require.NoError(t, err)
	code := request.Approved(ada, "code 1", now)
	good := connections.Redemption{ClientID: "the id", RedirectURI: callback, Verifier: connections.TestVerifier}

	require.NoError(t, code.RedeemError(good, now.Add(connections.CodeTTL-time.Nanosecond)))
	assert.Equal(t, admintokens.Holder{Account: ada, ClientID: connections.IDOf("the id"), Client: "Claude"}, code.Holder())
	assert.Equal(t, "code 1", code.ID())

	require.ErrorIs(t, code.RedeemError(good, now.Add(connections.CodeTTL)), connections.ErrInvalidGrant, "lapsed")
	for name, redemption := range map[string]connections.Redemption{
		"another client":   {ClientID: "another id", RedirectURI: callback, Verifier: connections.TestVerifier},
		"another redirect": {ClientID: "the id", RedirectURI: callback + "?x", Verifier: connections.TestVerifier},
		"another verifier": {ClientID: "the id", RedirectURI: callback, Verifier: connections.TestVerifier + "x"},
		"the challenge":    {ClientID: "the id", RedirectURI: callback, Verifier: connections.TestChallenge()},
		"no verifier":      {ClientID: "the id", RedirectURI: callback},
	} {
		require.ErrorIs(t, code.RedeemError(redemption, now), connections.ErrInvalidGrant, name)
	}
}
