package authorize_usecase_test

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/connections"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/connections/aes_connection_sealer"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/connections/usecases/authorize_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const (
	callback = "https://claude.ai/api/mcp/auth_callback"
	consent  = "https://backoffice.clickplanet.lol/connect"
)

var now = time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)

func registered(t *testing.T, sealer *aes_connection_sealer.Sealer) string {
	t.Helper()
	client, err := connections.NewClient("Claude", []string{callback}, connections.Redirects{callback}, now)
	require.NoError(t, err)
	id, err := sealer.SealedClient(client)
	require.NoError(t, err)
	return id
}

func ask() connections.Ask {
	return connections.Ask{ResponseType: "code", RedirectURI: callback, Challenge: connections.TestChallenge(), ChallengeMethod: "S256", State: "s"}
}

func TestTheBrowserIsSentToTheConsentPageWithTheRequestSealed(t *testing.T) {
	sealer, err := aes_connection_sealer.New([]byte("the seed"))
	require.NoError(t, err)
	id := registered(t, sealer)

	location, err := authorize_usecase.New(consent, sealer, cptime.NewFixedClock(now)).
		Execute(t.Context(), authorize_usecase.In{ClientID: id, Ask: ask()})

	require.NoError(t, err)
	require.True(t, strings.HasPrefix(location, consent+"?request="), location)
	sent, err := url.Parse(location)
	require.NoError(t, err)
	request, err := sealer.OpenedRequest(sent.Query().Get("request"))
	require.NoError(t, err)
	assert.Equal(t, connections.RequestSnapshot{
		ClientID: connections.IDOf(id), Client: "Claude", RedirectURI: callback, Challenge: connections.TestChallenge(), State: "s",
		ExpiresAt: now.Add(connections.RequestTTL),
	}, request.TakeSnapshot())
}

func TestAClientThisServerDidNotSealIsNotSentAnywhere(t *testing.T) {
	sealer, err := aes_connection_sealer.New([]byte("the seed"))
	require.NoError(t, err)
	other, err := aes_connection_sealer.New([]byte("another seed"))
	require.NoError(t, err)

	_, err = authorize_usecase.New(consent, sealer, cptime.NewFixedClock(now)).
		Execute(t.Context(), authorize_usecase.In{ClientID: registered(t, other), Ask: ask()})

	require.ErrorIs(t, err, connections.ErrInvalidClient)
}
