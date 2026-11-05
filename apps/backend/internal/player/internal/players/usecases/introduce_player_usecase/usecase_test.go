package introduce_player_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/inmemory_player_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/introduce_player_usecase"
)

var (
	ada   = players.AccountID{15: 1}
	guest = players.AccountID{15: 2}
)

func store(t *testing.T) *inmemory_player_store.Store {
	t.Helper()
	store := inmemory_player_store.New()
	require.NoError(t, store.SaveProfile(t.Context(), players.NewProfile(ada, "Ada_L", time.Now())))
	return store
}

func useCase(store *inmemory_player_store.Store) *introduce_player_usecase.UseCase {
	return introduce_player_usecase.New(store, players.NewGuestCodes(store, &players.SequentialCodes{}))
}

func TestAnAccountWithAUsernameIsAnsweredWithIt(t *testing.T) {
	player, err := useCase(store(t)).Execute(t.Context(), ada)

	require.NoError(t, err)
	assert.Equal(t, players.NamedPlayer(players.ProfileSnapshot{Name: "Ada_L"}.Restore()), player)
}

func TestAnAdminIsSaidToBeOne(t *testing.T) {
	admins := store(t)
	admins.MakeAdmin(ada)

	player, err := useCase(admins).Execute(t.Context(), ada)

	require.NoError(t, err)
	assert.Equal(t, players.NamedPlayer(players.ProfileSnapshot{Name: "Ada_L", Admin: true}.Restore()), player)
}

func TestAGuestIsGivenACodeOnceAndKeepsIt(t *testing.T) {
	guests := useCase(store(t))

	first, err := guests.Execute(t.Context(), guest)
	require.NoError(t, err)
	second, err := guests.Execute(t.Context(), guest)
	require.NoError(t, err)

	assert.Equal(t, players.GuestPlayer("000001"), first)
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

	player, err := useCase(colored).Execute(t.Context(), ada)

	require.NoError(t, err)
	assert.Equal(t, players.Color(7), player.Color())
}
