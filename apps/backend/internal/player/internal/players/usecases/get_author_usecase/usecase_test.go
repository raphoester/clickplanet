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
)

var (
	ada   = players.AccountID{15: 1}
	guest = players.AccountID{15: 2}
)

var today = time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)

func store(t *testing.T) *inmemory_player_store.Store {
	t.Helper()
	store := inmemory_player_store.New()
	require.NoError(t, store.SaveProfile(t.Context(), players.NewProfile(ada, "Ada_L", time.Now())))
	return store
}

func useCase(store *inmemory_player_store.Store) *get_author_usecase.UseCase {
	return get_author_usecase.New(store, players.NewGuestCodes(store, &players.SequentialCodes{}))
}

func TestAnAccountWithAUsernameIsAnsweredWithIt(t *testing.T) {
	author, err := useCase(store(t)).Execute(t.Context(), ada)

	require.NoError(t, err)
	assert.Equal(t, players.NamedAuthor(players.ProfileSnapshot{Name: "Ada_L"}.Restore()), author)
}

func TestAnAdminIsSaidToBeOne(t *testing.T) {
	admins := store(t)
	admins.MakeAdmin(ada)

	author, err := useCase(admins).Execute(t.Context(), ada)

	require.NoError(t, err)
	assert.Equal(t, players.NamedAuthor(players.ProfileSnapshot{Name: "Ada_L", Admin: true}.Restore()), author)
}

func TestAGuestIsGivenACodeOnceAndKeepsIt(t *testing.T) {
	guests := useCase(store(t))

	first, err := guests.Execute(t.Context(), guest)
	require.NoError(t, err)
	second, err := guests.Execute(t.Context(), guest)
	require.NoError(t, err)

	assert.Equal(t, players.GuestAuthor("000001"), first)
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

func TestAPlayerWhoTookTilesIsNamedAsOneWhoNeverDid(t *testing.T) {
	takes := store(t)
	require.NoError(t, takes.Record(t.Context(), ada, players.TakeAt(0, 0, today.AddDate(0, 0, -1))))
	require.NoError(t, takes.Record(t.Context(), ada, players.TakeAt(1, 0, today)))

	author, err := useCase(takes).Execute(t.Context(), ada)

	require.NoError(t, err)
	assert.Equal(t, players.NamedAuthor(players.ProfileSnapshot{Name: "Ada_L"}.Restore()), author,
		"a streak is a badge, which GetBadges answers: the name carries none")
}
