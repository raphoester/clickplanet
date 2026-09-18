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

func getAuthor(t *testing.T, accountID string) (*connect.Response[playerv1.GetAuthorResponse], error) {
	t.Helper()

	store := inmemory_player_store.New()
	ada, err := players.AccountIDOf("0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11")
	require.NoError(t, err)
	require.NoError(t, store.SaveProfile(t.Context(), players.Profile{Account: ada, Name: "Ada_L", UpdatedAt: time.Now()}))

	useCase := get_author_usecase.New(store, players.NewGuestCodes(store, &players.SequentialCodes{}))
	return get_author_handler.New(useCase).GetAuthor(t.Context(), //nolint:wrapcheck // the test reads the handler's own error.
		connect.NewRequest(&playerv1.GetAuthorRequest{AccountId: accountID}))
}

func TestTheUsernameIsAnswered(t *testing.T) {
	res, err := getAuthor(t, "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11")

	require.NoError(t, err)
	assert.Equal(t, "Ada_L", res.Msg.GetName())
	assert.False(t, res.Msg.GetAdmin())
}

func TestAGuestIsAnsweredWithItsCode(t *testing.T) {
	res, err := getAuthor(t, "5e0c1b2a-3d4e-4f60-8a71-9b2c3d4e5f60")

	require.NoError(t, err)
	assert.Equal(t, "guest_000001", res.Msg.GetName())
}

func TestAnIdThatIsNotAnAccountIsRefused(t *testing.T) {
	for _, id := range []string{"", "not-an-account", "00000000-0000-0000-0000-000000000000"} {
		_, err := getAuthor(t, id)

		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), id)
	}
}
