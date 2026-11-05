package introduce_player_handler_test

import (
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/inmemory_player_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/introduce_player_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/introduce_player_handler"
)

const (
	adaID   = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"
	guestID = "5e0c1b2a-3d4e-4f60-8a71-9b2c3d4e5f60"
)

func introduce(t *testing.T, accountID string) (*connect.Response[playerv1.IntroducePlayerResponse], error) {
	t.Helper()

	store := inmemory_player_store.New()
	ada, err := players.AccountIDOf(adaID)
	require.NoError(t, err)
	require.NoError(t, store.SaveProfile(t.Context(), players.NewProfile(ada, "Ada_L", time.Now())))
	require.NoError(t, store.SaveColor(t.Context(), ada, players.Color(playerv1.NameColor_NAME_COLOR_TEAL)))

	useCase := introduce_player_usecase.New(store, players.NewGuestCodes(store, &players.SequentialCodes{}))
	return introduce_player_handler.New(useCase).IntroducePlayer(t.Context(), //nolint:wrapcheck // the test reads the handler's own error.
		connect.NewRequest(&playerv1.IntroducePlayerRequest{AccountId: accountID}))
}

func TestThePlayerIsItsAccountItsUsernameAndItsColor(t *testing.T) {
	res, err := introduce(t, adaID)

	require.NoError(t, err)
	want := &playerv1.Player{AccountId: adaID, Name: "Ada_L", Color: playerv1.NameColor_NAME_COLOR_TEAL}
	assert.True(t, proto.Equal(want, res.Msg.GetPlayer()), "%v", res.Msg.GetPlayer())
}

func TestAGuestIsIntroducedUnderTheCodeItIsGiven(t *testing.T) {
	res, err := introduce(t, guestID)

	require.NoError(t, err)
	want := &playerv1.Player{AccountId: guestID, Name: "guest_000001", Guest: true}
	assert.True(t, proto.Equal(want, res.Msg.GetPlayer()), "%v", res.Msg.GetPlayer())
}

func TestAnIdThatIsNotAnAccountIsRefused(t *testing.T) {
	for _, id := range []string{"", "not-an-account", "00000000-0000-0000-0000-000000000000"} {
		_, err := introduce(t, id)

		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), id)
	}
}
