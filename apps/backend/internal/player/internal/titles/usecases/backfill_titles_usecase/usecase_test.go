package backfill_titles_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/inmemory_player_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles/inmemory_title_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles/usecases/backfill_titles_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	now     = time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	first   = titles.FakeTitle{Key: "first", Tiles: 1}
	third   = titles.FakeTitle{Key: "third", Tiles: 3}
	catalog = titles.Catalog{first, third}
)

func account(i int) players.AccountID {
	return players.AccountID{14: byte(i >> 8), 15: byte(i)}
}

type fixture struct {
	stats    *inmemory_player_store.Store
	titles   *inmemory_title_store.Store
	accounts *titles.FakeAccounts
}

func setUp() fixture {
	return fixture{stats: inmemory_player_store.New(), titles: inmemory_title_store.New(), accounts: titles.NewFakeAccounts()}
}

func (f fixture) take(t *testing.T, i, tiles int) {
	t.Helper()

	for range tiles {
		require.NoError(t, f.stats.RecordTake(t.Context(), account(i), now))
	}
}

func (f fixture) held(t *testing.T, i int) titles.IDs {
	t.Helper()

	held, err := f.titles.Held(t.Context(), account(i))
	require.NoError(t, err)
	return held
}

func (f fixture) backfill(t *testing.T, store backfill_titles_usecase.Titles, catalog titles.Catalog) (backfill_titles_usecase.Backfill, error) {
	t.Helper()

	return backfill_titles_usecase.New(f.stats, f.accounts, store, catalog, cptime.NewFixedClock(now)).Execute(t.Context()) //nolint:wrapcheck // the tests read the use case's error.
}

func TestEveryAccountGetsTheTitlesItsStatsEarnAcrossPages(t *testing.T) {
	f := setUp()
	for i := 1; i <= 1_001; i++ {
		f.take(t, i, 1)
	}
	f.take(t, 1_001, 2)

	done, err := f.backfill(t, f.titles, catalog)

	require.NoError(t, err)
	assert.Equal(t, backfill_titles_usecase.Backfill{Accounts: 1_001}, done)
	assert.Equal(t, titles.IDs{"first"}, f.held(t, 1))
	assert.Equal(t, titles.IDs{"first"}, f.held(t, 500))
	assert.Equal(t, titles.IDs{"first", "third"}, f.held(t, 1_001))
}

func TestAnAccountThatEarnsNothingIsNotCounted(t *testing.T) {
	f := setUp()
	f.take(t, 1, 1)

	done, err := f.backfill(t, f.titles, titles.Catalog{third})

	require.NoError(t, err)
	assert.Equal(t, 0, done.Accounts)
	assert.Empty(t, f.held(t, 1))
}

func TestTheAccountsMadeBeforeNovemberAreBackfilledOG(t *testing.T) {
	f := setUp()
	f.take(t, 1, 1)
	f.take(t, 2, 1)
	f.take(t, 3, 1)
	f.accounts.Create(account(1), time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC))
	f.accounts.Create(account(2), time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC))

	done, err := f.backfill(t, f.titles, titles.Catalog{titles.OG{}})

	require.NoError(t, err)
	assert.Equal(t, 1, done.Accounts)
	assert.Equal(t, titles.IDs{"og"}, f.held(t, 1))
	assert.Empty(t, f.held(t, 2), "made on the first of November")
	assert.Empty(t, f.held(t, 3), "auth does not know it")
}

func TestRunningItAgainGrantsNothingTwice(t *testing.T) {
	f := setUp()
	f.take(t, 1, 3)
	_, err := f.backfill(t, f.titles, catalog)
	require.NoError(t, err)

	_, err = f.backfill(t, f.titles, catalog)

	require.NoError(t, err)
	assert.Equal(t, titles.IDs{"first", "third"}, f.held(t, 1))
}

type refusingGrants struct {
	*inmemory_title_store.Store
}

func (refusingGrants) Grant(context.Context, titles.Grants, time.Time) error {
	return errors.New("postgres is down")
}

func TestAFailedGrantIsAnError(t *testing.T) {
	f := setUp()
	f.take(t, 1, 1)

	_, err := f.backfill(t, refusingGrants{f.titles}, catalog)

	require.Error(t, err)
}

func TestAFailureToAskAuthGrantsNothing(t *testing.T) {
	f := setUp()
	f.take(t, 1, 1)
	f.accounts.FailWith(errors.New("auth is down"))

	_, err := f.backfill(t, f.titles, catalog)

	require.Error(t, err)
	assert.Empty(t, f.held(t, 1))
}
