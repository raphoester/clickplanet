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

func store(t *testing.T) *inmemory_player_store.Store {
	t.Helper()
	store := inmemory_player_store.New()
	require.NoError(t, store.SaveProfile(t.Context(), players.Profile{Account: ada, Name: "Ada_L", UpdatedAt: time.Now()}))
	return store
}

func useCase(store *inmemory_player_store.Store) *get_author_usecase.UseCase {
	return get_author_usecase.New(store, players.NewGuestCodes(store, &players.SequentialCodes{}))
}

func TestAnAccountWithAUsernameIsAnsweredWithIt(t *testing.T) {
	author, err := useCase(store(t)).Execute(t.Context(), ada)

	require.NoError(t, err)
	assert.Equal(t, players.Author{Name: "Ada_L"}, author)
}

func TestAnAdminIsSaidToBeOne(t *testing.T) {
	admins := store(t)
	admins.MakeAdmin(ada)

	author, err := useCase(admins).Execute(t.Context(), ada)

	require.NoError(t, err)
	assert.Equal(t, players.Author{Name: "Ada_L", Admin: true}, author)
}

func TestAGuestIsGivenACodeOnceAndKeepsIt(t *testing.T) {
	guests := useCase(store(t))

	first, err := guests.Execute(t.Context(), guest)
	require.NoError(t, err)
	second, err := guests.Execute(t.Context(), guest)
	require.NoError(t, err)

	assert.Equal(t, players.Author{Name: "guest_000001", Guest: true}, first)
	assert.Equal(t, first, second)
}

func TestAStoreFailureIsAnError(t *testing.T) {
	failing := store(t)
	failing.FailWith(errors.New("postgres is down"))

	_, err := useCase(failing).Execute(t.Context(), ada)

	assert.Error(t, err)
}
