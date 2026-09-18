package get_player_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/inmemory_player_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/get_player_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	monday    = time.Date(2026, 9, 14, 20, 0, 0, 0, time.UTC)
	createdAt = time.Date(2026, 9, 1, 8, 30, 0, 0, time.UTC)
	ada       = players.AccountID{15: 1}
)

type fixture struct {
	store    *inmemory_player_store.Store
	accounts *get_player_usecase.FakeAccounts
	clock    *cptime.FixedClock
	useCase  *get_player_usecase.UseCase
}

func setUp(t *testing.T) fixture {
	t.Helper()

	store := inmemory_player_store.New()
	require.NoError(t, store.SaveProfile(t.Context(), players.Profile{Account: ada, Name: "Ada_L", UpdatedAt: createdAt}))
	accounts := get_player_usecase.NewFakeAccounts()
	accounts.Create(ada, createdAt)
	clock := cptime.NewFixedClock(monday)

	return fixture{store: store, accounts: accounts, clock: clock, useCase: get_player_usecase.New(store, store, accounts, clock)}
}

func TestAPlayerIsFoundByItsNameInAnyCase(t *testing.T) {
	f := setUp(t)
	require.NoError(t, f.store.RecordTake(t.Context(), ada, monday.Add(-24*time.Hour)))
	require.NoError(t, f.store.RecordTake(t.Context(), ada, monday))

	player, err := f.useCase.Execute(t.Context(), "aDA_l")

	require.NoError(t, err)
	assert.Equal(t, players.Player{
		Name: "Ada_L",
		Stats: players.Stats{
			Account: ada, TilesTaken: 2, StreakCurrent: 2, StreakBest: 2, StreakLastDay: players.DayOf(monday),
		},
		CreatedAt: createdAt,
	}, player)
}

func TestAnAdminIsSaidToBeOne(t *testing.T) {
	f := setUp(t)
	f.store.MakeAdmin(ada)

	player, err := f.useCase.Execute(t.Context(), "Ada_L")

	require.NoError(t, err)
	assert.True(t, player.Admin)
}

func TestAPlayerThatNeverTookATileHasEmptyStats(t *testing.T) {
	player, err := setUp(t).useCase.Execute(t.Context(), "Ada_L")

	require.NoError(t, err)
	assert.Equal(t, players.Stats{Account: ada}, player.Stats)
}

func TestTheStreakIsReadAsOfToday(t *testing.T) {
	f := setUp(t)
	require.NoError(t, f.store.RecordTake(t.Context(), ada, monday))
	f.clock.Advance(48 * time.Hour)

	player, err := f.useCase.Execute(t.Context(), "Ada_L")

	require.NoError(t, err)
	assert.Equal(t, uint32(0), player.Stats.StreakCurrent)
	assert.Equal(t, uint32(1), player.Stats.StreakBest)
}

func TestANameNobodyHoldsIsNoProfile(t *testing.T) {
	_, err := setUp(t).useCase.Execute(t.Context(), "Bob")

	assert.ErrorIs(t, err, players.ErrNoProfile)
}

func TestANameNoAccountMayHoldIsNoProfileAndReadsNothing(t *testing.T) {
	f := setUp(t)
	f.store.FailWith(errors.New("postgres is down"))

	for _, name := range []string{"", "guest_Ada", "a b", "Émile"} {
		_, err := f.useCase.Execute(t.Context(), name)

		assert.ErrorIs(t, err, players.ErrNoProfile, "%q", name)
	}
}

func TestAnAccountAuthNoLongerKnowsHasNoCreationDate(t *testing.T) {
	f := setUp(t)
	require.NoError(t, f.store.SaveProfile(t.Context(), players.Profile{Account: players.AccountID{15: 2}, Name: "Bob", UpdatedAt: createdAt}))

	player, err := f.useCase.Execute(t.Context(), "Bob")

	require.NoError(t, err)
	assert.True(t, player.CreatedAt.IsZero())
}

func TestAStoreFailureIsNotNoProfile(t *testing.T) {
	f := setUp(t)
	f.store.FailWith(errors.New("postgres is down"))

	_, err := f.useCase.Execute(t.Context(), "Ada_L")

	require.Error(t, err)
	assert.NotErrorIs(t, err, players.ErrNoProfile)
}

func TestAnAuthFailureIsAnError(t *testing.T) {
	f := setUp(t)
	f.accounts.FailWith(errors.New("auth is down"))

	_, err := f.useCase.Execute(t.Context(), "Ada_L")

	require.Error(t, err)
	assert.NotErrorIs(t, err, players.ErrNoProfile)
}
