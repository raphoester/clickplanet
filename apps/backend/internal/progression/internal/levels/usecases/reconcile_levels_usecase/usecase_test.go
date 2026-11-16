package reconcile_levels_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	bonusesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/bonuses/v1"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"

	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/levels"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/levels/inmemory_level_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/levels/outbox_bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/levels/usecases/reconcile_levels_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats/inmemory_stats_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
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
	outbox   *cpatomic.FakeOutbox
}

func setUp() fixture {
	return fixture{
		stats: inmemory_stats_store.New(), accounts: accounts.NewFakeAccounts(), levels: inmemory_level_store.New(),
		outbox: &cpatomic.FakeOutbox{},
	}
}

func (f fixture) useCase() *reconcile_levels_usecase.UseCase {
	book := levels.NewBook(cpatomic.FakeTransactioner{}, f.levels, outbox_bonuses.New(f.outbox))
	return reconcile_levels_usecase.New(f.stats, f.accounts, book, cptime.NewFixedClock(now))
}

func (f fixture) player(t *testing.T, i int, tiles uint64, known accounts.Account, kept levels.Level) {
	t.Helper()

	require.NoError(t, f.stats.Record(t.Context(), account(i), func(stats.Stats) stats.Stats {
		return stats.StatsSnapshot{Account: account(i), TilesTaken: tiles}.Restore()
	}))
	f.accounts.Create(account(i), known)
	if kept > 0 {
		require.NoError(t, f.levels.RaiseAll(t.Context(), levels.ByAccount{account(i): kept}, now))
	}
}

func (f fixture) reconcile(t *testing.T, dryRun bool) reconcile_levels_usecase.Reconciled {
	t.Helper()

	reconciled, err := f.useCase().Execute(t.Context(), dryRun)
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

func TestEachAccountIsRaisedNeverLoweredAndAGuestLosesIts(t *testing.T) {
	f := setUp()
	f.mixed(t)

	reconciled := f.reconcile(t, false)

	assert.Equal(t, reconcile_levels_usecase.Reconciled{Raised: 2, Removed: 1}, reconciled)
	assert.Equal(t, levels.ByAccount{account(1): 3, account(2): 3, account(3): 7, account(4): 3}, f.kept(t, 1, 2, 3, 4, 5),
		"a level the players saw stays, though the stats reach 3")
	assert.Equal(t, reconcile_levels_usecase.Reconciled{}, f.reconcile(t, false), "a second run changes nothing")
}

func TestADryRunCountsTheSameAndWritesNothing(t *testing.T) {
	f := setUp()
	f.mixed(t)

	reconciled := f.reconcile(t, true)

	assert.Equal(t, reconcile_levels_usecase.Reconciled{Raised: 2, Removed: 1}, reconciled)
	assert.Equal(t, levels.ByAccount{account(2): 2, account(3): 7, account(4): 3, account(5): 4}, f.kept(t, 1, 2, 3, 4, 5))
	assert.Empty(t, f.outbox.Recorded(), "a dry run tells the bonuses nothing")
}

func TestEveryAccountOfAPageIsToldToTheBonusesWithWhatItsLevelUnlocks(t *testing.T) {
	f := setUp()
	f.mixed(t)
	f.player(t, 6, 1_000, linked, 0)

	f.reconcile(t, false)

	told := map[string]*bonusesv1.KeepUnlocks{}
	for _, recorded := range f.outbox.Recorded() {
		command, ok := recorded.(*bonusesv1.KeepUnlocks)
		require.True(t, ok, "a reconciliation pays no level-up: %T", recorded)
		told[command.GetAccountId()] = command
	}
	require.Len(t, told, 6, "each linked account, changed or not, and the guest whose level went")
	assert.True(t, proto.Equal(&bonusesv1.KeepUnlocks{
		AccountId: account(6).String(),
		Kinds: []planetv1.BonusKind{
			planetv1.BonusKind_BONUS_KIND_SPREAD_CLICKS, planetv1.BonusKind_BONUS_KIND_SHIELDS,
			planetv1.BonusKind_BONUS_KIND_ENCLOSE_CLICKS,
		},
		DecidedAt: timestamppb.New(now),
	}, told[account(6).String()]), "level 5: %v", told[account(6).String()])
	assert.Equal(t, []planetv1.BonusKind{
		planetv1.BonusKind_BONUS_KIND_SPREAD_CLICKS, planetv1.BonusKind_BONUS_KIND_SHIELDS,
	}, told[account(4).String()].GetKinds(), "level 3")
	assert.Equal(t, []planetv1.BonusKind{
		planetv1.BonusKind_BONUS_KIND_SPREAD_CLICKS, planetv1.BonusKind_BONUS_KIND_SHIELDS,
		planetv1.BonusKind_BONUS_KIND_ENCLOSE_CLICKS, planetv1.BonusKind_BONUS_KIND_BOMB,
	}, told[account(3).String()].GetKinds(), "level 7, kept: the stats reach 3 only")
	assert.Empty(t, told[account(5).String()].GetKinds(), "a guest has nothing unlocked")
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

	_, err := f.useCase().Execute(t.Context(), false)

	require.ErrorContains(t, err, "player is down")
	assert.Equal(t, levels.ByAccount{account(2): 2, account(3): 7, account(4): 3, account(5): 4}, f.kept(t, 1, 2, 3, 4, 5))
}
