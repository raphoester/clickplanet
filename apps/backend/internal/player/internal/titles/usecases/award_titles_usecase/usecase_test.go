package award_titles_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/inmemory_player_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles/inmemory_title_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles/usecases/award_titles_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	monday  = time.Date(2026, 9, 14, 20, 0, 0, 0, time.UTC)
	ada     = players.AccountID{15: 1}
	catalog = titles.CatalogOf([]titles.Title{titles.FakeTitle{Key: "first", Tiles: 1}, titles.FakeTitle{Key: "third", Tiles: 3}})
)

type fixture struct {
	stats    *inmemory_player_store.Store
	titles   *inmemory_title_store.Store
	accounts *titles.FakeAccounts
}

func setUp() fixture {
	f := fixture{stats: inmemory_player_store.New(), titles: inmemory_title_store.New(), accounts: titles.NewFakeAccounts()}
	f.accounts.Create(ada, players.Account{Linked: true, CreatedAt: monday})
	return f
}

func (f fixture) award(t *testing.T, catalog titles.Catalog) error {
	t.Helper()

	_, err := f.awarded(t, catalog)
	return err
}

func (f fixture) awarded(t *testing.T, catalog titles.Catalog) (titles.IDs, error) {
	t.Helper()

	useCase := award_titles_usecase.New(f.stats, f.accounts, titles.NewBook(f.titles, catalog), cptime.NewFixedClock(monday))
	return useCase.Execute(t.Context(), ada) //nolint:wrapcheck // the tests read the use case's error.
}

func TestTheAnswerIsWhatThisTakeGranted(t *testing.T) {
	f := setUp()
	for range 3 {
		require.NoError(t, f.stats.RecordTake(t.Context(), ada, monday))
	}
	require.NoError(t, f.titles.Grant(t.Context(), titles.Holdings{ada: {"first"}}, monday))

	granted, err := f.awarded(t, catalog)
	require.NoError(t, err)
	assert.Equal(t, titles.IDs{"third"}, granted)

	granted, err = f.awarded(t, catalog)
	require.NoError(t, err)
	assert.Empty(t, granted, "a take that earns nothing new grants nothing")
}

func (f fixture) held(t *testing.T) titles.IDs {
	t.Helper()

	held, err := f.titles.Held(t.Context(), ada)
	require.NoError(t, err)
	return held
}

func TestTheTakeThatReachesATitleGrantsIt(t *testing.T) {
	f := setUp()
	for range 2 {
		require.NoError(t, f.stats.RecordTake(t.Context(), ada, monday))
	}
	require.NoError(t, f.award(t, catalog))
	require.Equal(t, titles.IDs{"first"}, f.held(t))

	require.NoError(t, f.stats.RecordTake(t.Context(), ada, monday))
	require.NoError(t, f.award(t, catalog))

	assert.Equal(t, titles.IDs{"first", "third"}, f.held(t))
}

func TestANewAccountMadeBeforeNovemberIsOGAtItsFirstTake(t *testing.T) {
	f := setUp()
	f.accounts.Create(ada, players.Account{Linked: true, CreatedAt: time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC)})
	require.NoError(t, f.stats.RecordTake(t.Context(), ada, monday))

	require.NoError(t, f.award(t, titles.NewCatalog()))

	assert.Equal(t, titles.IDs{"og"}, f.held(t))
}

func TestAGuestsTakeGrantsNothing(t *testing.T) {
	f := setUp()
	f.accounts.Create(ada, players.Account{CreatedAt: time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)})
	for range 3 {
		require.NoError(t, f.stats.RecordTake(t.Context(), ada, monday))
	}

	require.NoError(t, f.award(t, titles.NewCatalog()))
	require.NoError(t, f.award(t, catalog))

	assert.Empty(t, f.held(t))
}

func TestAnAccountWithNoStatsEarnsNothing(t *testing.T) {
	f := setUp()

	require.NoError(t, f.award(t, catalog))

	assert.Empty(t, f.held(t))
}

func TestAStatsFailureIsAnError(t *testing.T) {
	f := setUp()
	require.NoError(t, f.stats.RecordTake(t.Context(), ada, monday))
	f.stats.FailWith(errors.New("postgres is down"))

	assert.Error(t, f.award(t, catalog))
}

func TestAnAuthFailureIsAnErrorAndGrantsNothing(t *testing.T) {
	f := setUp()
	require.NoError(t, f.stats.RecordTake(t.Context(), ada, monday))
	f.accounts.FailWith(errors.New("auth is down"))

	require.Error(t, f.award(t, catalog))

	assert.Empty(t, f.held(t))
}
