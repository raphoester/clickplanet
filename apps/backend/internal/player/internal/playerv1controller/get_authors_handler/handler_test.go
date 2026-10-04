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
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles/inmemory_title_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/wearing"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/wearing/inmemory_worn_title_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var today = time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)

const (
	adaID     = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"
	guestID   = "5e0c1b2a-3d4e-4f60-8a71-9b2c3d4e5f60"
	unknownID = "9f8e7d6c-5b4a-4938-8271-6a5b4c3d2e10"
)

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
	require.NoError(t, store.SaveColor(t.Context(), ada, players.Color(playerv1.NameColor_NAME_COLOR_PINK)))
	require.NoError(t, store.RecordTake(t.Context(), ada, today.AddDate(0, 0, -1)))
	require.NoError(t, store.RecordTake(t.Context(), ada, today))
	require.NoError(t, store.RecordTake(t.Context(), guest, today.AddDate(0, 0, -1)))
	require.NoError(t, store.RecordTake(t.Context(), guest, today))

	held, worn := inmemory_title_store.New(), inmemory_worn_title_store.New()
	require.NoError(t, held.Grant(t.Context(), titles.Holdings{ada: {"og"}, guest: {"og"}}, today))
	wardrobe := wearing.NewWardrobe(worn, titles.NewBook(held, titles.NewCatalog()), titles.NewCatalog())

	return get_authors_handler.New(get_authors_usecase.New(store, wardrobe, cptime.NewFixedClock(today))).GetAuthors(t.Context(), //nolint:wrapcheck // the test reads the handler's own error.
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

func TestEachAuthorCarriesItsColorAndItsStreakAsOfToday(t *testing.T) {
	res, err := getAuthors(t, adaID, guestID)
	require.NoError(t, err)

	by := make(map[string]*playerv1.Author, len(res.Msg.GetAuthors()))
	for _, author := range res.Msg.GetAuthors() {
		by[author.GetAccountId()] = author
	}

	assert.Equal(t, playerv1.NameColor_NAME_COLOR_PINK, by[adaID].GetColor())
	assert.Equal(t, uint32(2), by[adaID].GetStreak())
	assert.Equal(t, playerv1.NameColor_NAME_COLOR_UNSPECIFIED, by[guestID].GetColor())
	assert.Equal(t, uint32(0), by[guestID].GetStreak(), "a guest shows no streak, however long it runs")
}

func TestEachAuthorCarriesTheTitleItWearsAndAGuestNone(t *testing.T) {
	res, err := getAuthors(t, adaID, guestID)
	require.NoError(t, err)

	by := make(map[string]*playerv1.Author, len(res.Msg.GetAuthors()))
	for _, author := range res.Msg.GetAuthors() {
		by[author.GetAccountId()] = author
	}

	assert.Equal(t, "og", by[adaID].GetWornTitle().GetId())
	assert.Equal(t, "OG", by[adaID].GetWornTitle().GetName())
	assert.Nil(t, by[guestID].GetWornTitle())
}

func TestAGuestIsToldApartFromAUsername(t *testing.T) {
	res, err := getAuthors(t, adaID, guestID)
	require.NoError(t, err)

	guests := make(map[string]bool, len(res.Msg.GetAuthors()))
	for _, author := range res.Msg.GetAuthors() {
		guests[author.GetAccountId()] = author.GetGuest()
	}

	assert.Equal(t, map[string]bool{adaID: false, guestID: true}, guests)
}
