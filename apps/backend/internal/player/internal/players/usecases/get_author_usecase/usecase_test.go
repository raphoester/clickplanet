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
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
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

func TestAnAccountWithAUsernameIsAnsweredWithItAndItsTag(t *testing.T) {
	author, err := get_author_usecase.New(store(t), "pepper").Execute(t.Context(),
		get_author_usecase.In{Account: ada, IP: "1.2.3.4"})

	require.NoError(t, err)
	assert.Equal(t, players.Author{Name: "Ada_L", Tag: players.TagOf("pepper", "1.2.3.4")}, author)
}

func TestAnAccountWithNoUsernameHasATagAndNoName(t *testing.T) {
	author, err := get_author_usecase.New(store(t), "pepper").Execute(t.Context(),
		get_author_usecase.In{Account: guest, IP: "1.2.3.4"})

	require.NoError(t, err)
	assert.Equal(t, players.Author{Tag: players.TagOf("pepper", "1.2.3.4")}, author)
}

func TestACallerWithNoAccountHasATagAndTheStoreIsNotRead(t *testing.T) {
	failing := store(t)
	failing.FailWith(errors.New("postgres is down"))

	author, err := get_author_usecase.New(failing, "pepper").Execute(t.Context(),
		get_author_usecase.In{Account: cpsession.NoAccount, IP: "1.2.3.4"})

	require.NoError(t, err)
	assert.Equal(t, players.Author{Tag: players.TagOf("pepper", "1.2.3.4")}, author)
}

func TestAStoreFailureIsAnError(t *testing.T) {
	failing := store(t)
	failing.FailWith(errors.New("postgres is down"))

	_, err := get_author_usecase.New(failing, "pepper").Execute(t.Context(),
		get_author_usecase.In{Account: ada, IP: "1.2.3.4"})

	assert.Error(t, err)
}
