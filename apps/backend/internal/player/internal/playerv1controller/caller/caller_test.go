package caller_test

import (
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/caller"
)

func TestNoAccountIsUnauthenticated(t *testing.T) {
	_, err := caller.AccountOf(t.Context())

	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	assert.ErrorIs(t, err, caller.ErrNoAccount)
	assert.NotErrorIs(t, err, players.ErrInvalidAccount, "the parse error is not what the caller reads")
}
