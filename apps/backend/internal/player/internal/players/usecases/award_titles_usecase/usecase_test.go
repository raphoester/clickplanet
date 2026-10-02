package award_titles_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/inmemory_player_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/award_titles_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	monday  = time.Date(2026, 9, 14, 20, 0, 0, 0, time.UTC)
	ada     = players.AccountID{15: 1}
	catalog = players.Catalog{players.FakeTitle{Key: "first", Tiles: 1}, players.FakeTitle{Key: "third", Tiles: 3}}
)

func awardFor(store *inmemory_player_store.Store, accounts *players.FakeAccounts, catalog players.Catalog) *award_titles_usecase.UseCase {
	return award_titles_usecase.New(store, accounts, players.NewTitleBook(store, catalog), cptime.NewFixedClock(monday))
}

func held(t *testing.T, store *inmemory_player_store.Store) players.TitleIDs {
	t.Helper()

	titles, err := store.Titles(t.Context(), ada)
	require.NoError(t, err)
	return titles
}

func TestTheTakeThatReachesATitleGrantsIt(t *testing.T) {
	store, accounts := inmemory_player_store.New(), players.NewFakeAccounts()
	for range 2 {
		require.NoError(t, store.RecordTake(t.Context(), ada, monday))
	}
	require.NoError(t, awardFor(store, accounts, catalog).Execute(t.Context(), ada))
	require.Equal(t, players.TitleIDs{"first"}, held(t, store))

	require.NoError(t, store.RecordTake(t.Context(), ada, monday))
	require.NoError(t, awardFor(store, accounts, catalog).Execute(t.Context(), ada))

	assert.Equal(t, players.TitleIDs{"first", "third"}, held(t, store))
}

func TestANewAccountMadeBeforeNovemberIsOGAtItsFirstTake(t *testing.T) {
	store, accounts := inmemory_player_store.New(), players.NewFakeAccounts()
	accounts.Create(ada, time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC))
	require.NoError(t, store.RecordTake(t.Context(), ada, monday))

	require.NoError(t, awardFor(store, accounts, players.NewCatalog()).Execute(t.Context(), ada))

	assert.Equal(t, players.TitleIDs{"og"}, held(t, store))
}

func TestAnAccountWithNoStatsEarnsNothing(t *testing.T) {
	store := inmemory_player_store.New()

	require.NoError(t, awardFor(store, players.NewFakeAccounts(), catalog).Execute(t.Context(), ada))

	assert.Empty(t, held(t, store))
}

func TestAStoreFailureIsAnError(t *testing.T) {
	store := inmemory_player_store.New()
	require.NoError(t, store.RecordTake(t.Context(), ada, monday))
	store.FailWith(errors.New("postgres is down"))

	assert.Error(t, awardFor(store, players.NewFakeAccounts(), catalog).Execute(t.Context(), ada))
}

func TestAnAuthFailureIsAnErrorAndGrantsNothing(t *testing.T) {
	store, accounts := inmemory_player_store.New(), players.NewFakeAccounts()
	require.NoError(t, store.RecordTake(t.Context(), ada, monday))
	accounts.FailWith(errors.New("auth is down"))

	require.Error(t, awardFor(store, accounts, catalog).Execute(t.Context(), ada))

	assert.Empty(t, held(t, store))
}
