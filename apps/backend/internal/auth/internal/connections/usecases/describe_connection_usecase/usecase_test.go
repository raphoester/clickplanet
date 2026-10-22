package describe_connection_usecase_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/connections"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/connections/aes_connection_sealer"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/connections/usecases/describe_connection_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const callback = "https://claude.ai/api/mcp/auth_callback"

func TestTheOperatorIsToldWhoAsksAndWhereItIs(t *testing.T) {
	now := time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
	sealer, err := aes_connection_sealer.New([]byte("the seed"))
	require.NoError(t, err)
	request, err := connections.RequestSnapshot{
		ClientID: "the id", Client: "Claude", RedirectURI: callback, Challenge: connections.TestChallenge(), State: "s",
		ExpiresAt: now.Add(connections.RequestTTL),
	}.Restore()
	require.NoError(t, err)
	sealed, err := sealer.SealedRequest(request)
	require.NoError(t, err)

	out, err := describe_connection_usecase.New(sealer, cptime.NewFixedClock(now)).Execute(t.Context(), sealed)
	require.NoError(t, err)
	assert.Equal(t, describe_connection_usecase.Out{
		Client: "Claude", RedirectHost: "claude.ai", DeniedURL: callback + "?error=access_denied&state=s",
	}, out)

	_, err = describe_connection_usecase.New(sealer, cptime.NewFixedClock(now.Add(connections.RequestTTL))).Execute(t.Context(), sealed)
	require.ErrorIs(t, err, connections.ErrExpired)
	_, err = describe_connection_usecase.New(sealer, cptime.NewFixedClock(now)).Execute(t.Context(), "made up")
	require.ErrorIs(t, err, connections.ErrInvalidRequest)
}
