package get_profile_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/inmemory_player_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/get_profile_usecase"
)

func TestAnAccountThatNeverChoseANameHasAnEmptyOne(t *testing.T) {
	profile, err := get_profile_usecase.New(inmemory_player_store.New()).Execute(t.Context(), players.AccountID{15: 1})

	require.NoError(t, err)
	assert.Equal(t, players.Profile{Account: players.AccountID{15: 1}}, profile)
}

func TestTheChosenNameIsAnswered(t *testing.T) {
	store := inmemory_player_store.New()
	saved := players.Profile{Account: players.AccountID{15: 1}, Name: "Ada", UpdatedAt: time.Date(2026, 9, 17, 0, 0, 0, 0, time.UTC)}
	require.NoError(t, store.SaveProfile(t.Context(), saved))

	profile, err := get_profile_usecase.New(store).Execute(t.Context(), players.AccountID{15: 1})

	require.NoError(t, err)
	assert.Equal(t, saved, profile)
}

func TestAStoreFailureIsNotAnEmptyProfile(t *testing.T) {
	store := inmemory_player_store.New()
	store.FailWith(errors.New("postgres is down"))

	_, err := get_profile_usecase.New(store).Execute(t.Context(), players.AccountID{15: 1})

	require.Error(t, err)
	assert.NotErrorIs(t, err, players.ErrNoProfile)
}
