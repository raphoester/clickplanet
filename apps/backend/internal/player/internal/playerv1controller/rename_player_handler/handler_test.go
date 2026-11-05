package rename_player_handler_test

import (
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/inmemory_player_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/rename_player_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/rename_player_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const (
	ada     = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"
	bob     = "5c3a1b2d-8e9f-4a0b-9c1d-2e3f4a5b6c7d"
	unnamed = "9f8e7d6c-5b4a-4392-8170-6f5e4d3c2b1a"
)

var now = time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)

func handler(t *testing.T) rename_player_handler.RenamePlayerHandler {
	t.Helper()

	store := inmemory_player_store.New()
	for id, name := range map[string]players.Name{ada: "Rubbish", bob: "Bob"} {
		account, err := players.AccountIDOf(id)
		require.NoError(t, err)
		require.NoError(t, store.SaveProfile(t.Context(), players.NewProfile(account, name, now)))
	}
	return rename_player_handler.New(rename_player_usecase.New(store, cptime.NewFixedClock(now)))
}

func rename(t *testing.T, account, name string) (*playerv1.RenamePlayerResponse, error) {
	t.Helper()

	res, err := handler(t).RenamePlayer(t.Context(), connect.NewRequest(&playerv1.RenamePlayerRequest{AccountId: account, Name: name}))
	if err != nil {
		return nil, fmt.Errorf("RenamePlayer failed: %w", err)
	}
	return res.Msg, nil
}

func TestTheNameIsAnsweredAsKept(t *testing.T) {
	res, err := rename(t, ada, " Ada_L ")

	require.NoError(t, err)
	assert.Equal(t, "Ada_L", res.GetName())
}

func TestEachRefusalHasItsCode(t *testing.T) {
	for _, tc := range []struct {
		name    string
		account string
		typed   string
		code    connect.Code
	}{
		{name: "not an account id", account: "ada", typed: "Ada", code: connect.CodeInvalidArgument},
		{name: "an invalid name", account: ada, typed: "guest_ada", code: connect.CodeInvalidArgument},
		{name: "no username", account: unnamed, typed: "Ada", code: connect.CodeNotFound},
		{name: "a name another player holds", account: ada, typed: "BOB", code: connect.CodeAlreadyExists},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := rename(t, tc.account, tc.typed)

			assert.Equal(t, tc.code, connect.CodeOf(err))
		})
	}
}
