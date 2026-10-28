package set_shown_in_clips_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/inmemory_player_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/set_shown_in_clips_usecase"
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

func TestAPlayerWithAUsernameKeepsItsChoice(t *testing.T) {
	profiles := store(t)

	require.NoError(t, set_shown_in_clips_usecase.New(profiles).Execute(t.Context(), set_shown_in_clips_usecase.In{Account: ada, Shown: false}))

	assert.False(t, profiles.ShownInClips(ada))
}

func TestAnAccountWithNoUsernameIsRefused(t *testing.T) {
	err := set_shown_in_clips_usecase.New(store(t)).Execute(t.Context(), set_shown_in_clips_usecase.In{Account: guest, Shown: false})

	require.ErrorIs(t, err, players.ErrNoProfile)
}

func TestAStoreFailureIsAnError(t *testing.T) {
	failing := store(t)
	failing.FailWith(errors.New("postgres is down"))

	err := set_shown_in_clips_usecase.New(failing).Execute(t.Context(), set_shown_in_clips_usecase.In{Account: ada, Shown: false})

	require.Error(t, err)
	assert.NotErrorIs(t, err, players.ErrNoProfile)
}
