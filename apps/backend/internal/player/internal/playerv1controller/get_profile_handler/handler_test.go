package get_profile_handler_test

import (
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/inmemory_player_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/get_profile_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_profile_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

func handler() get_profile_handler.GetProfileHandler {
	return get_profile_handler.New(get_profile_usecase.New(inmemory_player_store.New()))
}

func TestAnUnnamedCallerGetsItsAccountAndNoName(t *testing.T) {
	res, err := handler().GetProfile(cpctx.AddAccountToContext(t.Context(), "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"),
		connect.NewRequest(&playerv1.GetProfileRequest{}))

	require.NoError(t, err)
	assert.Equal(t, "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11", res.Msg.GetProfile().GetAccountId())
	assert.Empty(t, res.Msg.GetProfile().GetName())
}

func TestACallerWithNoAccountIsUnauthenticated(t *testing.T) {
	_, err := handler().GetProfile(t.Context(), connect.NewRequest(&playerv1.GetProfileRequest{}))

	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}
