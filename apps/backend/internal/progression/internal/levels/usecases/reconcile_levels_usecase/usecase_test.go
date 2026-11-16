package reconcile_levels_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/levels"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/levels/inmemory_level_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/levels/usecases/reconcile_levels_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats/inmemory_stats_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	now    = time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	linked = accounts.AccountOf(true, now)
	guest  = accounts.AccountOf(false, time.Time{})
)

const level3Tiles = 112

func account(i int) accounts.AccountID {
	return accounts.AccountID{14: byte(i >> 8), 15: byte(i)}
}

type fixture struct {
	stats    *inmemory_stats_store.Store
	accounts *accounts.FakeAccounts
	levels   *inmemory_level_store.Store
}

func setUp() fixture {
	return fixture{stats: inmemory_stats_store.New(), accounts: accounts.NewFakeAccounts(), levels: inmemory_level_store.New()}
}

func (f fixture) player(t *testing.T, i int, tiles uint64, known accounts.Account, kept levels.Level) {
	t.Helper()

	require.NoError(t, f.stats.Record(t.Context(), account(i), func(stats.Stats) stats.Stats {
		return stats.StatsSnapshot{Account: account(i), TilesTaken: tiles}.Restore()
	}))
	f.accounts.Create(account(i), known)
	if kept > 0 {
		require.NoError(t, f.levels.Set(t.Context(), levels.ByAccount{account(i): kept}, now))
	}
}

func (f fixture) reconcile(t *testing.T, dryRun bool) reconcile_levels_usecase.Reconciled {
	t.Helper()

	reconciled, err := reconcile_levels_usecase.New(f.stats, f.accounts, f.levels, cptime.NewFixedClock(now)).Execute(t.Context(), dryRun)
	require.NoError(t, err)
	return reconciled
}

func (f fixture) kept(t *testing.T, asked ...int) levels.ByAccount {
	t.Helper()

	ids := make([]accounts.AccountID, 0, len(asked))
	for _, i := range asked {
		ids = append(ids, account(i))
	}
	kept, err := f.levels.Levels(t.Context(), ids)
	require.NoError(t, err)
	return kept
}

func (f fixture) mixed(t *testing.T) {
	t.Helper()

	f.player(t, 1, level3Tiles, linked, 0)
	f.player(t, 2, level3Tiles, linked, 2)
	f.player(t, 3, level3Tiles, linked, 7)
	f.player(t, 4, level3Tiles, linked, 3)
	f.player(t, 5, level3Tiles, guest, 4)
}

func TestEachAccountGetsTheExactLevelAndAGuestLosesIts(t *testing.T) {
	f := setUp()
	f.mixed(t)

	reconciled := f.reconcile(t, false)

	assert.Equal(t, reconcile_levels_usecase.Reconciled{Raised: 2, Lowered: 1, Removed: 1}, reconciled)
	assert.Equal(t, levels.ByAccount{account(1): 3, account(2): 3, account(3): 3, account(4): 3}, f.kept(t, 1, 2, 3, 4, 5))
	assert.Equal(t, reconcile_levels_usecase.Reconciled{}, f.reconcile(t, false), "a second run changes nothing")
}

func TestADryRunCountsTheSameAndWritesNothing(t *testing.T) {
	f := setUp()
	f.mixed(t)

	reconciled := f.reconcile(t, true)

	assert.Equal(t, reconcile_levels_usecase.Reconciled{Raised: 2, Lowered: 1, Removed: 1}, reconciled)
	assert.Equal(t, levels.ByAccount{account(2): 2, account(3): 7, account(4): 3, account(5): 4}, f.kept(t, 1, 2, 3, 4, 5))
}

func TestEveryPageIsReconciled(t *testing.T) {
	f := setUp()
	// From 1: account(0) is the zero id, the cursor the first page starts after, and no account.
	for i := 1; i <= 1_001; i++ {
		f.player(t, i, level3Tiles, linked, 0)
	}

	assert.Equal(t, reconcile_levels_usecase.Reconciled{Raised: 1_001}, f.reconcile(t, false))
	assert.Equal(t, levels.ByAccount{account(1_001): 3}, f.kept(t, 1_001))
}

func TestNoStatsAsksNothing(t *testing.T) {
	f := setUp()

	assert.Equal(t, reconcile_levels_usecase.Reconciled{}, f.reconcile(t, false))
	assert.Empty(t, f.accounts.Asked())
}

func TestAFailureToAskPlayerWritesNothing(t *testing.T) {
	f := setUp()
	f.mixed(t)
	f.accounts.FailWith(errors.New("player is down"))

	_, err := reconcile_levels_usecase.New(f.stats, f.accounts, f.levels, cptime.NewFixedClock(now)).Execute(t.Context(), false)

	require.ErrorContains(t, err, "player is down")
	assert.Equal(t, levels.ByAccount{account(2): 2, account(3): 7, account(4): 3, account(5): 4}, f.kept(t, 1, 2, 3, 4, 5))
}
