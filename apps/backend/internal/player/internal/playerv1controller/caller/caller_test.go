package caller_test

import (
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/caller"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

func TestTheAccountOnTheContextIsTheCaller(t *testing.T) {
	ctx := cpctx.AddAccountToContext(t.Context(), "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11")

	account, err := caller.AccountOf(ctx)

	require.NoError(t, err)
	assert.Equal(t, "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11", account.String())
}

func TestNoAccountIsUnauthenticated(t *testing.T) {
	_, err := caller.AccountOf(t.Context())

	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	assert.ErrorIs(t, err, caller.ErrNoAccount)
	assert.NotErrorIs(t, err, players.ErrInvalidAccount, "the parse error is not what the caller reads")
}
