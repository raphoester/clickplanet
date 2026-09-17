package get_author_handler_test

import (
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/inmemory_player_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/get_author_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_author_handler"
)

func getAuthor(t *testing.T, accountID string) *playerv1.GetAuthorResponse {
	t.Helper()

	store := inmemory_player_store.New()
	ada, err := players.AccountIDOf("0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11")
	require.NoError(t, err)
	require.NoError(t, store.SaveProfile(t.Context(), players.Profile{Account: ada, Name: "Ada_L", UpdatedAt: time.Now()}))

	res, err := get_author_handler.New(get_author_usecase.New(store, "pepper")).GetAuthor(t.Context(),
		connect.NewRequest(&playerv1.GetAuthorRequest{AccountId: accountID, Ip: "1.2.3.4"}))
	require.NoError(t, err)
	return res.Msg
}

func TestTheUsernameAndTheTagAreAnswered(t *testing.T) {
	res := getAuthor(t, "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11")

	assert.Equal(t, "Ada_L", res.GetUsername())
	assert.Equal(t, string(players.TagOf("pepper", "1.2.3.4")), res.GetTag())
	assert.False(t, res.GetAdmin())
}

func TestAnIdThatIsNotAnAccountHasOnlyATag(t *testing.T) {
	for _, id := range []string{"", "not-an-account"} {
		res := getAuthor(t, id)

		assert.Empty(t, res.GetUsername(), id)
		assert.Equal(t, string(players.TagOf("pepper", "1.2.3.4")), res.GetTag(), id)
	}
}
