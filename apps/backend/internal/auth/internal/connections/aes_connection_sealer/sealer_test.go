package aes_connection_sealer_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/connections"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/connections/aes_connection_sealer"
)

const callback = "https://claude.ai/api/mcp/auth_callback"

var now = time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)

func sealer(t *testing.T, seed string) *aes_connection_sealer.Sealer {
	t.Helper()
	sealer, err := aes_connection_sealer.New([]byte(seed))
	require.NoError(t, err)
	return sealer
}

func TestEachKindOpensAsItWasSealed(t *testing.T) {
	s := sealer(t, "the seed")
	client, err := connections.NewClient("Claude", []string{callback}, connections.Redirects{callback}, now)
	require.NoError(t, err)

	id, err := s.SealedClient(client)
	require.NoError(t, err)
	opened, err := s.OpenedClient(id)
	require.NoError(t, err)
	assert.Equal(t, client.TakeSnapshot(), opened.TakeSnapshot())

	request, err := client.Request(id, connections.Ask{
		ResponseType: "code", RedirectURI: callback, Challenge: connections.TestChallenge(), ChallengeMethod: "S256", State: "s",
	}, now)
	require.NoError(t, err)
	sealedRequest, err := s.SealedRequest(request)
	require.NoError(t, err)
	openedRequest, err := s.OpenedRequest(sealedRequest)
	require.NoError(t, err)
	assert.Equal(t, request.TakeSnapshot(), openedRequest.TakeSnapshot())

	code := request.Approved(accounts.AccountID{15: 1}, "code 1", now)
	sealedCode, err := s.SealedCode(code)
	require.NoError(t, err)
	openedCode, err := s.OpenedCode(sealedCode)
	require.NoError(t, err)
	assert.Equal(t, code.TakeSnapshot(), openedCode.TakeSnapshot())

	_, err = s.OpenedCode(sealedRequest)
	require.ErrorIs(t, err, connections.ErrInvalidGrant, "a request is no code")
	_, err = s.OpenedRequest(id)
	require.ErrorIs(t, err, connections.ErrInvalidRequest, "a client is no request")
	_, err = s.OpenedClient(sealedCode)
	require.ErrorIs(t, err, connections.ErrInvalidClient, "a code is no client")
}

func TestOnlyThisServerOpensWhatItSealed(t *testing.T) {
	client, err := connections.NewClient("Claude", []string{callback}, connections.Redirects{callback}, now)
	require.NoError(t, err)
	id, err := sealer(t, "the seed").SealedClient(client)
	require.NoError(t, err)

	for name, value := range map[string]string{
		"another seed": id,
		"not base64":   "not a client id!",
		"empty":        "",
		"tampered":     id[:len(id)-2] + "AA",
	} {
		opener := sealer(t, "the seed")
		if name == "another seed" {
			opener = sealer(t, "another seed")
		}
		_, err := opener.OpenedClient(value)
		require.ErrorIs(t, err, connections.ErrInvalidClient, name)
	}
}
