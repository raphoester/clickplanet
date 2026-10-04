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
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles/inmemory_title_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/wearing"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/wearing/inmemory_worn_title_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func getAuthor(t *testing.T, accountID string) (*connect.Response[playerv1.GetAuthorResponse], error) {
	t.Helper()

	store := inmemory_player_store.New()
	ada, err := players.AccountIDOf("0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11")
	require.NoError(t, err)
	require.NoError(t, store.SaveProfile(t.Context(), players.NewProfile(ada, "Ada_L", time.Now())))
	require.NoError(t, store.SaveColor(t.Context(), ada, players.Color(playerv1.NameColor_NAME_COLOR_TEAL)))
	require.NoError(t, store.RecordTake(t.Context(), ada, time.Now()))

	held, worn := inmemory_title_store.New(), inmemory_worn_title_store.New()
	require.NoError(t, held.Grant(t.Context(), titles.Holdings{ada: {"og", "settler"}}, time.Now()))
	require.NoError(t, worn.Wear(t.Context(), ada, "settler", time.Now()))
	wardrobe := wearing.NewWardrobe(worn, titles.NewBook(held, titles.NewCatalog()), titles.NewCatalog())

	useCase := get_author_usecase.New(store, players.NewGuestCodes(store, &players.SequentialCodes{}), wardrobe, cptime.NewFixedClock(time.Now()))
	return get_author_handler.New(useCase).GetAuthor(t.Context(), //nolint:wrapcheck // the test reads the handler's own error.
		connect.NewRequest(&playerv1.GetAuthorRequest{AccountId: accountID}))
}

func TestTheUsernameIsAnswered(t *testing.T) {
	res, err := getAuthor(t, "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11")

	require.NoError(t, err)
	assert.Equal(t, "Ada_L", res.Msg.GetName())
	assert.False(t, res.Msg.GetAdmin())
	assert.Equal(t, playerv1.NameColor_NAME_COLOR_TEAL, res.Msg.GetColor())
	assert.Equal(t, uint32(1), res.Msg.GetStreak())
	assert.Equal(t, "settler", res.Msg.GetWornTitle().GetId())
	assert.Equal(t, "conquest", res.Msg.GetWornTitle().GetRank().GetTrackId())
}

func TestAGuestIsAnsweredWithItsCode(t *testing.T) {
	res, err := getAuthor(t, "5e0c1b2a-3d4e-4f60-8a71-9b2c3d4e5f60")

	require.NoError(t, err)
	assert.Equal(t, "guest_000001", res.Msg.GetName())
	assert.Nil(t, res.Msg.GetWornTitle())
}

func TestAnIdThatIsNotAnAccountIsRefused(t *testing.T) {
	for _, id := range []string{"", "not-an-account", "00000000-0000-0000-0000-000000000000"} {
		_, err := getAuthor(t, id)

		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), id)
	}
}
