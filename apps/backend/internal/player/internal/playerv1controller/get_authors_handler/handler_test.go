package get_authors_handler_test

import (
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/inmemory_player_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/get_authors_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_authors_handler"
)

const (
	adaID     = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"
	guestID   = "5e0c1b2a-3d4e-4f60-8a71-9b2c3d4e5f60"
	unknownID = "9f8e7d6c-5b4a-4938-8271-6a5b4c3d2e10"
)

// ada has a username, the guest only a code, and nobody has ever heard of unknownID.
func getAuthors(t *testing.T, ids ...string) (*connect.Response[playerv1.GetAuthorsResponse], error) {
	t.Helper()

	store := inmemory_player_store.New()
	ada, err := players.AccountIDOf(adaID)
	require.NoError(t, err)
	require.NoError(t, store.SaveProfile(t.Context(),
		players.Profile{Account: ada, Name: "Ada_L", UpdatedAt: time.Now()}))

	guest, err := players.AccountIDOf(guestID)
	require.NoError(t, err)
	require.NoError(t, store.SaveGuestCode(t.Context(), guest, "91aa3d"))

	return get_authors_handler.New(get_authors_usecase.New(store)).GetAuthors(t.Context(), //nolint:wrapcheck // the test reads the handler's own error.
		connect.NewRequest(&playerv1.GetAuthorsRequest{AccountIds: ids}))
}

func named(res *connect.Response[playerv1.GetAuthorsResponse]) map[string]string {
	by := make(map[string]string, len(res.Msg.GetAuthors()))
	for _, author := range res.Msg.GetAuthors() {
		by[author.GetAccountId()] = author.GetName()
	}
	return by
}

func TestEachAccountIsNamedByItsIDAndAGuestByItsCode(t *testing.T) {
	res, err := getAuthors(t, adaID, guestID)

	require.NoError(t, err)
	assert.Equal(t, map[string]string{adaID: "Ada_L", guestID: "guest_91aa3d"}, named(res))
}

func TestAnAccountThatCannotBeNamedIsLeftOutRatherThanFailingTheCall(t *testing.T) {
	res, err := getAuthors(t, adaID, unknownID)

	require.NoError(t, err)
	assert.Equal(t, map[string]string{adaID: "Ada_L"}, named(res))
}

func TestAskingAboutNobodyAnswersNobody(t *testing.T) {
	res, err := getAuthors(t)

	require.NoError(t, err)
	assert.Empty(t, res.Msg.GetAuthors())
}

func TestARepeatedIDIsAnsweredOnce(t *testing.T) {
	res, err := getAuthors(t, adaID, adaID)

	require.NoError(t, err)
	assert.Len(t, res.Msg.GetAuthors(), 1)
}

func TestAnIdThatIsNotAnAccountIsRefused(t *testing.T) {
	for _, id := range []string{"", "not-an-account", "00000000-0000-0000-0000-000000000000"} {
		_, err := getAuthors(t, adaID, id)

		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), id)
	}
}

func TestTheAnswerIsNeverStored(t *testing.T) {
	res, err := getAuthors(t, adaID)

	require.NoError(t, err)
	assert.Equal(t, "no-store", res.Header().Get("Cache-Control"), "who a player is changes when it renames")
}
