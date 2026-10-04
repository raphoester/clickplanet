package get_author_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/inmemory_player_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/get_author_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles/inmemory_title_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/wearing"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/wearing/inmemory_worn_title_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	ada   = players.AccountID{15: 1}
	guest = players.AccountID{15: 2}
	bob   = players.AccountID{15: 3}
)

var today = time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)

func store(t *testing.T) *inmemory_player_store.Store {
	t.Helper()
	store := inmemory_player_store.New()
	require.NoError(t, store.SaveProfile(t.Context(), players.NewProfile(ada, "Ada_L", time.Now())))
	return store
}

var catalog = titles.NewCatalog()

func useCase(store *inmemory_player_store.Store) *get_author_usecase.UseCase {
	return dressed(store, inmemory_title_store.New(), inmemory_worn_title_store.New())
}

func dressed(
	store *inmemory_player_store.Store,
	held *inmemory_title_store.Store,
	worn *inmemory_worn_title_store.Store,
) *get_author_usecase.UseCase {
	wardrobe := wearing.NewWardrobe(worn, titles.NewBook(held, catalog), catalog)
	return get_author_usecase.New(store, players.NewGuestCodes(store, &players.SequentialCodes{}), wardrobe, cptime.NewFixedClock(today))
}

func standing(id titles.ID) titles.Standing {
	standing, _ := catalog.StandingOf(id)
	return standing
}

func TestAnAccountWithAUsernameIsAnsweredWithIt(t *testing.T) {
	author, err := useCase(store(t)).Execute(t.Context(), ada)

	require.NoError(t, err)
	assert.Equal(t, wearing.AuthorOf(players.NamedAuthor(players.ProfileOf(players.AccountID{}, "Ada_L", time.Time{}, false, 0), players.Streak{}), titles.Standing{}), author)
}

func TestAnAdminIsSaidToBeOne(t *testing.T) {
	admins := store(t)
	admins.MakeAdmin(ada)

	author, err := useCase(admins).Execute(t.Context(), ada)

	require.NoError(t, err)
	assert.Equal(t, wearing.AuthorOf(players.NamedAuthor(players.ProfileOf(players.AccountID{}, "Ada_L", time.Time{}, true, 0), players.Streak{}), titles.Standing{}), author)
}

func TestAGuestIsGivenACodeOnceAndKeepsIt(t *testing.T) {
	guests := useCase(store(t))

	first, err := guests.Execute(t.Context(), guest)
	require.NoError(t, err)
	second, err := guests.Execute(t.Context(), guest)
	require.NoError(t, err)

	assert.Equal(t, wearing.AuthorOf(players.GuestAuthor("000001", players.Streak{}), titles.Standing{}), first)
	assert.Equal(t, first, second)
}

func TestAStoreFailureIsAnError(t *testing.T) {
	failing := store(t)
	failing.FailWith(errors.New("postgres is down"))

	_, err := useCase(failing).Execute(t.Context(), ada)

	assert.Error(t, err)
}

func TestTheChosenColorIsAnswered(t *testing.T) {
	colored := store(t)
	require.NoError(t, colored.SaveColor(t.Context(), ada, 7))

	author, err := useCase(colored).Execute(t.Context(), ada)

	require.NoError(t, err)
	assert.Equal(t, players.Color(7), author.Color())
}

func TestTheStreakIsReadAsOfToday(t *testing.T) {
	takes := store(t)
	require.NoError(t, takes.SaveProfile(t.Context(), players.NewProfile(bob, "Bob", time.Now())))
	require.NoError(t, takes.RecordTake(t.Context(), ada, today.AddDate(0, 0, -2)))
	require.NoError(t, takes.RecordTake(t.Context(), ada, today.AddDate(0, 0, -1)))
	require.NoError(t, takes.RecordTake(t.Context(), bob, today.AddDate(0, 0, -3)))
	require.NoError(t, takes.RecordTake(t.Context(), bob, today.AddDate(0, 0, -2)))

	alive, err := useCase(takes).Execute(t.Context(), ada)
	require.NoError(t, err)
	lapsed, err := useCase(takes).Execute(t.Context(), bob)
	require.NoError(t, err)

	assert.Equal(t, uint32(2), alive.Streak().Days(), "a take yesterday can still be extended today")
	assert.Equal(t, uint32(0), lapsed.Streak().Days(), "a whole day went by with no take")
}

func TestAGuestShowsNoStreak(t *testing.T) {
	takes := store(t)
	require.NoError(t, takes.RecordTake(t.Context(), guest, today.AddDate(0, 0, -1)))
	require.NoError(t, takes.RecordTake(t.Context(), guest, today))

	author, err := useCase(takes).Execute(t.Context(), guest)

	require.NoError(t, err)
	assert.True(t, author.Guest())
	assert.Equal(t, players.Streak{}, author.Streak(), "a flame is for a player that signed in and chose a name")
}

func TestTheTitleWornIsAnswered(t *testing.T) {
	held := inmemory_title_store.New()
	require.NoError(t, held.Grant(t.Context(), titles.Holdings{ada: {"og", "settler"}}, today))
	worn := inmemory_worn_title_store.New()
	require.NoError(t, worn.Wear(t.Context(), ada, "og", today))

	author, err := dressed(store(t), held, worn).Execute(t.Context(), ada)

	require.NoError(t, err)
	assert.Equal(t, standing("og"), author.Worn())
}

func TestAGuestWearsNoTitleAndNoTitleIsRead(t *testing.T) {
	held := inmemory_title_store.New()
	held.FailWith(errors.New("postgres is down"))

	author, err := dressed(store(t), held, inmemory_worn_title_store.New()).Execute(t.Context(), guest)

	require.NoError(t, err)
	assert.True(t, author.Worn().Empty())
}

func TestATitleStoreFailureIsAnError(t *testing.T) {
	worn := inmemory_worn_title_store.New()
	worn.FailWith(errors.New("postgres is down"))

	_, err := dressed(store(t), inmemory_title_store.New(), worn).Execute(t.Context(), ada)

	assert.Error(t, err)
}
