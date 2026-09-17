package get_names_handler_test

import (
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/inmemory_player_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/get_names_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_names_handler"
)

func TestNamesAreAnsweredByAccountIdAndTheRestLeftOut(t *testing.T) {
	storage := inmemory_player_storage.New(inmemory_player_storage.NewMemoryPersistence())
	ada, err := players.AccountIDOf("0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11")
	require.NoError(t, err)
	storage.SaveProfile(players.Profile{Account: ada, Name: "Ada", UpdatedAt: time.Now()})

	res, err := get_names_handler.New(get_names_usecase.New(storage)).GetNames(t.Context(),
		connect.NewRequest(&playerv1.GetNamesRequest{AccountIds: []string{
			ada.String(), "5e0e7a0c-7a8e-4b53-8f55-1d3c2f9d2b10", "not-an-account",
		}}))

	require.NoError(t, err)
	assert.Equal(t, map[string]string{ada.String(): "Ada"}, res.Msg.GetNames())
}
