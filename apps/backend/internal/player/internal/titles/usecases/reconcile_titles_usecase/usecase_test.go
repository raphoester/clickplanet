package reconcile_titles_usecase_test

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
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles/usecases/reconcile_titles_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	now     = time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	first   = titles.FakeTitle{Key: "first", Tiles: 1}
	third   = titles.FakeTitle{Key: "third", Tiles: 3}
	catalog = titles.CatalogOf([]titles.Title{first, third})
	linked  = players.Account{Linked: true, CreatedAt: now}
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

func (f fixture) player(t *testing.T, i, tiles int, known players.Account) {
	t.Helper()

	for range tiles {
		require.NoError(t, f.stats.RecordTake(t.Context(), account(i), now))
	}
	f.accounts.Create(account(i), known)
}

func (f fixture) held(t *testing.T, i int) titles.IDs {
	t.Helper()

	held, err := f.titles.Held(t.Context(), account(i))
	require.NoError(t, err)
	return held
}

func (f fixture) reconcile(t *testing.T, store reconcile_titles_usecase.Titles, catalog titles.Catalog) (reconcile_titles_usecase.Reconciled, error) {
	t.Helper()

	return reconcile_titles_usecase.New(f.stats, f.accounts, store, catalog, cptime.NewFixedClock(now)).Execute(t.Context()) //nolint:wrapcheck // the tests read the use case's error.
}

func TestEveryLinkedAccountGetsTheTitlesItsCareerEarnsAcrossPages(t *testing.T) {
	f := setUp()
	for i := 1; i <= 1_001; i++ {
		f.player(t, i, 1, linked)
	}
	f.player(t, 1_001, 2, linked)

	reconciled, err := f.reconcile(t, f.titles, catalog)

	require.NoError(t, err)
	assert.Equal(t, reconcile_titles_usecase.Reconciled{Granted: 1_002}, reconciled)
	assert.Equal(t, titles.IDs{"first"}, f.held(t, 1))
	assert.Equal(t, titles.IDs{"first"}, f.held(t, 500))
	assert.Equal(t, titles.IDs{"first", "third"}, f.held(t, 1_001))
}

func TestAGuestEarnsNothingAndLosesWhatItHeld(t *testing.T) {
	f := setUp()
	f.player(t, 1, 3, players.Account{CreatedAt: now})
	f.player(t, 2, 3, players.Account{})
	require.NoError(t, f.titles.Grant(t.Context(), titles.Holdings{account(1): {"first", "third"}}, now))

	reconciled, err := f.reconcile(t, f.titles, catalog)

	require.NoError(t, err)
	assert.Equal(t, reconcile_titles_usecase.Reconciled{Revoked: 2}, reconciled)
	assert.Empty(t, f.held(t, 1))
	assert.Empty(t, f.held(t, 2), "nor does an account auth does not know")
}

func TestATitleTheRulesNoLongerGiveIsRevoked(t *testing.T) {
	f := setUp()
	f.player(t, 1, 2, linked)
	require.NoError(t, f.titles.Grant(t.Context(), titles.Holdings{account(1): {"first", "third", "retired"}}, now))

	reconciled, err := f.reconcile(t, f.titles, catalog)

	require.NoError(t, err)
	assert.Equal(t, reconcile_titles_usecase.Reconciled{Revoked: 2}, reconciled)
	assert.Equal(t, titles.IDs{"first"}, f.held(t, 1))
}

func TestTheLinkedAccountsMadeBeforeNovemberAreOG(t *testing.T) {
	f := setUp()
	f.player(t, 1, 1, players.Account{Linked: true, CreatedAt: time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)})
	f.player(t, 2, 1, players.Account{Linked: true, CreatedAt: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)})
	f.player(t, 3, 1, players.Account{CreatedAt: time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)})

	_, err := f.reconcile(t, f.titles, titles.CatalogOf([]titles.Title{titles.OG{}}))

	require.NoError(t, err)
	assert.Equal(t, titles.IDs{"og"}, f.held(t, 1))
	assert.Empty(t, f.held(t, 2), "made on the first of November")
	assert.Empty(t, f.held(t, 3), "a guest")
}

func TestASecondRunChangesNothing(t *testing.T) {
	f := setUp()
	f.player(t, 1, 3, linked)
	f.player(t, 2, 3, players.Account{})
	_, err := f.reconcile(t, f.titles, catalog)
	require.NoError(t, err)

	reconciled, err := f.reconcile(t, f.titles, catalog)

	require.NoError(t, err)
	assert.Equal(t, reconcile_titles_usecase.Reconciled{}, reconciled)
	assert.Equal(t, titles.IDs{"first", "third"}, f.held(t, 1))
}

type refusingGrants struct {
	*inmemory_title_store.Store
}

func (refusingGrants) Grant(context.Context, titles.Holdings, time.Time) error {
	return errors.New("postgres is down")
}

func TestAFailedGrantIsAnError(t *testing.T) {
	f := setUp()
	f.player(t, 1, 1, linked)

	_, err := f.reconcile(t, refusingGrants{f.titles}, catalog)

	require.Error(t, err)
}

func TestAFailureToAskAuthChangesNothing(t *testing.T) {
	f := setUp()
	f.player(t, 1, 1, linked)
	require.NoError(t, f.titles.Grant(t.Context(), titles.Holdings{account(1): {"third"}}, now))
	f.accounts.FailWith(errors.New("auth is down"))

	_, err := f.reconcile(t, f.titles, catalog)

	require.Error(t, err)
	assert.Equal(t, titles.IDs{"third"}, f.held(t, 1))
}
