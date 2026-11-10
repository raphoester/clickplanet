package caller_test

import (
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/antibotv1controller/caller"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

func TestTheAccountOnTheContextIsTheCaller(t *testing.T) {
	account, err := caller.AccountOf(cpctx.AddAccountToContext(t.Context(), "0B6D4F7E-5D7C-4A36-9A51-3F1F8F0C2A11"))

	require.NoError(t, err)
	assert.Equal(t, "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11", account)
}

func TestNoAccountIsUnauthenticated(t *testing.T) {
	_, err := caller.AccountOf(t.Context())

	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	assert.ErrorIs(t, err, caller.ErrNoAccount)
}
