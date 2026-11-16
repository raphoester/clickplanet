package raise_level_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	progressionv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/progression/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/levels"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/levels/inmemory_level_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/levels/usecases/raise_level_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats/inmemory_stats_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	ada = accounts.AccountID{15: 1}
	now = time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
)

const (
	level3Tiles = 112
	level9Tiles = 7_168
)

type fixture struct {
	stats    *inmemory_stats_store.Store
	accounts *accounts.FakeAccounts
	levels   *inmemory_level_store.Store
	outbox   *cpatomic.FakeOutbox
}

func setUp(t *testing.T, linked bool) fixture {
	t.Helper()

	f := fixture{
		stats: inmemory_stats_store.New(), accounts: accounts.NewFakeAccounts(), levels: inmemory_level_store.New(),
		outbox: &cpatomic.FakeOutbox{},
	}
	f.accounts.Create(ada, accounts.AccountOf(linked, now))
	return f
}

func (f fixture) took(t *testing.T, tiles uint64) {
	t.Helper()

	require.NoError(t, f.stats.Record(t.Context(), ada, func(stats.Stats) stats.Stats {
		return stats.StatsSnapshot{Account: ada, TilesTaken: tiles}.Restore()
	}))
}

func (f fixture) raise(t *testing.T, store raise_level_usecase.Levels, visibility levels.Visibility) (levels.Level, error) {
	t.Helper()

	useCase := raise_level_usecase.New(
		f.stats, f.accounts, store, cpatomic.FakeTransactioner{}, f.outbox, cptime.NewFixedClock(now), visibility,
	)
	return useCase.Execute(t.Context(), ada) //nolint:wrapcheck // the tests read the use case's error.
}

func (f fixture) kept(t *testing.T) levels.ByAccount {
	t.Helper()

	kept, err := f.levels.Levels(t.Context(), []accounts.AccountID{ada})
	require.NoError(t, err)
	return kept
}

func reached(level, previous uint32, silent bool) *progressionv1.LevelReached {
	return &progressionv1.LevelReached{
		AccountId: ada.String(), Level: level, Previous: previous, ReachedAt: timestamppb.New(now), Silent: silent,
	}
}

func assertRecorded(t *testing.T, outbox *cpatomic.FakeOutbox, expected ...proto.Message) {
	t.Helper()

	recorded := outbox.Recorded()
	require.Len(t, recorded, len(expected))
	for i := range expected {
		assert.True(t, proto.Equal(expected[i], recorded[i]), "got %v", recorded[i])
	}
}

func TestALinkedAccountIsRaisedAndToldSilentlyWhileHidden(t *testing.T) {
	f := setUp(t, true)
	f.took(t, level3Tiles)

	raised, err := f.raise(t, f.levels, levels.Hidden)

	require.NoError(t, err)
	assert.Equal(t, levels.Level(3), raised)
	assert.Equal(t, levels.ByAccount{ada: 3}, f.kept(t))
	assertRecorded(t, f.outbox, reached(3, 0, true))
}

func TestOnceShownALevelUpIsNotSilent(t *testing.T) {
	f := setUp(t, true)
	f.took(t, level3Tiles)

	_, err := f.raise(t, f.levels, levels.Shown)

	require.NoError(t, err)
	assertRecorded(t, f.outbox, reached(3, 0, false))
}

func TestAJumpOfManyLevelsIsOneLevelUpFromTheLevelBefore(t *testing.T) {
	f := setUp(t, true)
	require.NoError(t, f.levels.Set(t.Context(), levels.ByAccount{ada: 1}, now))
	f.took(t, level9Tiles)

	_, err := f.raise(t, f.levels, levels.Hidden)

	require.NoError(t, err)
	assertRecorded(t, f.outbox, reached(9, 1, true))
}

func TestAGuestGetsNoLevelAndNoLevelUp(t *testing.T) {
	f := setUp(t, false)
	f.took(t, level9Tiles)

	raised, err := f.raise(t, f.levels, levels.Hidden)

	require.NoError(t, err)
	assert.Zero(t, raised)
	assert.Empty(t, f.kept(t))
	assert.Empty(t, f.outbox.Recorded())
	assert.Equal(t, []accounts.AccountID{ada}, f.accounts.Asked())
}

func TestNoRiseAsksPlayerNothingAndWritesNothing(t *testing.T) {
	f := setUp(t, true)
	require.NoError(t, f.levels.Set(t.Context(), levels.ByAccount{ada: 3}, now))
	f.took(t, level3Tiles)

	raised, err := f.raise(t, f.levels, levels.Hidden)

	require.NoError(t, err)
	assert.Zero(t, raised)
	assert.Empty(t, f.accounts.Asked())
	assert.Empty(t, f.outbox.Recorded())
}

func TestAnAccountWithNoStatsIsNotRaised(t *testing.T) {
	f := setUp(t, true)

	raised, err := f.raise(t, f.levels, levels.Hidden)

	require.NoError(t, err)
	assert.Zero(t, raised)
	assert.Empty(t, f.accounts.Asked())
}

func TestAFailureToAskPlayerWritesNothingAndTheNextTryRaises(t *testing.T) {
	f := setUp(t, true)
	f.took(t, level3Tiles)
	f.accounts.FailWith(errors.New("player is down"))

	_, err := f.raise(t, f.levels, levels.Hidden)

	require.ErrorContains(t, err, "player is down")
	assert.Empty(t, f.kept(t))
	assert.Empty(t, f.outbox.Recorded())

	f.accounts.FailWith(nil)
	_, err = f.raise(t, f.levels, levels.Hidden)
	require.NoError(t, err)
	assertRecorded(t, f.outbox, reached(3, 0, true))
}

// A live raise lands between the read of the level kept and this one's raise.
type racedLevels struct {
	*inmemory_level_store.Store
}

func (r racedLevels) Levels(ctx context.Context, asked []accounts.AccountID) (levels.ByAccount, error) {
	kept, err := r.Store.Levels(ctx, asked)
	if err != nil {
		return nil, err //nolint:wrapcheck // a test double.
	}
	_, err = r.Raise(ctx, ada, 9, now)
	return kept, err //nolint:wrapcheck // a test double.
}

func TestARaiseThatFindsAHigherLevelKeptTellsNothing(t *testing.T) {
	f := setUp(t, true)
	f.took(t, level3Tiles)

	raised, err := f.raise(t, racedLevels{f.levels}, levels.Hidden)

	require.NoError(t, err)
	assert.Zero(t, raised)
	assert.Equal(t, levels.ByAccount{ada: 9}, f.kept(t))
	assert.Empty(t, f.outbox.Recorded())
}

func TestAFailureToRaiseRecordsNothing(t *testing.T) {
	f := setUp(t, true)
	f.took(t, level3Tiles)
	f.levels.FailWith(errors.New("postgres is down"))

	_, err := f.raise(t, f.levels, levels.Hidden)

	require.ErrorContains(t, err, "postgres is down")
	assert.Empty(t, f.outbox.Recorded())
}
