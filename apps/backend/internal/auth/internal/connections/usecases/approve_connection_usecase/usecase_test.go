package approve_connection_usecase_test

import (
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/connections"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/connections/aes_connection_sealer"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/connections/usecases/approve_connection_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const (
	callback = "https://claude.ai/api/mcp/auth_callback"
	issuer   = "https://api.clickplanet.lol"
)

var (
	now = time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
	ada = accounts.AccountID{15: 1}
)

type ids struct{}

func (ids) NewSecret() (string, error) {
	return "code 1", nil
}

func sealed(t *testing.T, sealer *aes_connection_sealer.Sealer) string {
	t.Helper()
	request, err := connections.RequestSnapshot{
		ClientID: "the id", Client: "Claude", RedirectURI: callback, Challenge: connections.TestChallenge(), State: "s",
		ExpiresAt: now.Add(connections.RequestTTL),
	}.Restore()
	require.NoError(t, err)
	value, err := sealer.SealedRequest(request)
	require.NoError(t, err)
	return value
}

func TestTheAppIsSentACodeForTheOperatorWhoLetItIn(t *testing.T) {
	sealer, err := aes_connection_sealer.New([]byte("the seed"))
	require.NoError(t, err)

	out, err := approve_connection_usecase.New(issuer, sealer, ids{}, cptime.NewFixedClock(now)).
		Execute(t.Context(), approve_connection_usecase.In{Request: sealed(t, sealer), Operator: ada})
	require.NoError(t, err)

	assert.Equal(t, "Claude", out.Client)
	assert.Equal(t, "claude.ai", out.Host)
	sent, err := url.Parse(out.RedirectURL)
	require.NoError(t, err)
	assert.Equal(t, callback, sent.Scheme+"://"+sent.Host+sent.Path)
	assert.Equal(t, "s", sent.Query().Get("state"))
	assert.Equal(t, issuer, sent.Query().Get("iss"))
	code, err := sealer.OpenedCode(sent.Query().Get("code"))
	require.NoError(t, err)
	assert.Equal(t, connections.CodeSnapshot{
		ID: "code 1", Account: ada, ClientID: "the id", Client: "Claude", RedirectURI: callback,
		Challenge: connections.TestChallenge(), ExpiresAt: now.Add(connections.CodeTTL),
	}, code.TakeSnapshot())
}

func TestARequestPastItsTimeLetsNobodyIn(t *testing.T) {
	sealer, err := aes_connection_sealer.New([]byte("the seed"))
	require.NoError(t, err)

	_, err = approve_connection_usecase.New(issuer, sealer, ids{}, cptime.NewFixedClock(now.Add(connections.RequestTTL))).
		Execute(t.Context(), approve_connection_usecase.In{Request: sealed(t, sealer), Operator: ada})

	require.ErrorIs(t, err, connections.ErrExpired)
}
