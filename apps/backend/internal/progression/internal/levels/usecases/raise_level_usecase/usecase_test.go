package raise_level_usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/levels"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/levels/inmemory_level_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/levels/inprocess_level_feed"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/levels/usecases/raise_level_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats/inmemory_stats_store"
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
	feed     *inprocess_level_feed.Feed
	stream   <-chan levels.Told
}

func setUp(t *testing.T, linked bool) fixture {
	t.Helper()

	f := fixture{
		stats: inmemory_stats_store.New(), accounts: accounts.NewFakeAccounts(), levels: inmemory_level_store.New(),
		feed: inprocess_level_feed.New(),
	}
	f.stream = f.feed.Subscribe(t.Context(), ada, nil)
	f.accounts.Create(ada, accounts.AccountOf(linked, now))
	return f
}

func (f fixture) took(t *testing.T, tiles uint64) {
	t.Helper()

	f.counted(t, stats.StatsSnapshot{Account: ada, TilesTaken: tiles})
}

func (f fixture) counted(t *testing.T, snapshot stats.StatsSnapshot) {
	t.Helper()

	require.NoError(t, f.stats.Record(t.Context(), ada, func(stats.Stats) stats.Stats { return snapshot.Restore() }))
}

func (f fixture) raise(t *testing.T, store raise_level_usecase.Levels) (levels.Level, error) {
	t.Helper()

	return raise_level_usecase.New(f.stats, f.accounts, store, f.feed, cptime.NewFixedClock(now)).Execute(t.Context(), ada) //nolint:wrapcheck // the tests read the use case's error.
}

func (f fixture) told(t *testing.T) []levels.Told {
	t.Helper()

	var told []levels.Told
	for len(f.stream) > 0 {
		told = append(told, <-f.stream)
	}
	return told
}

func (f fixture) reached(t *testing.T) []levels.Standing {
	t.Helper()

	var reached []levels.Standing
	for _, told := range f.told(t) {
		if level, ok := told.(levels.Reached); ok {
			reached = append(reached, level.Standing)
		}
	}
	return reached
}

func tiles(xp levels.XP) levels.Gain { return levels.Gain{Source: levels.TilesTaken{}.Name(), XP: xp} }

func (f fixture) kept(t *testing.T) levels.ByAccount {
	t.Helper()

	kept, err := f.levels.Levels(t.Context(), []accounts.AccountID{ada})
	require.NoError(t, err)
	return kept
}

func TestALinkedAccountIsRaisedToTheLevelItsStatsReach(t *testing.T) {
	f := setUp(t, true)
	f.took(t, level3Tiles)

	raised, err := f.raise(t, f.levels)

	require.NoError(t, err)
	assert.Equal(t, levels.Level(3), raised)
	assert.Equal(t, levels.ByAccount{ada: 3}, f.kept(t))
	third := levels.Curve{}.StandingOf(level3Tiles)
	assert.Equal(t, []levels.Told{
		levels.Gained{Gains: []levels.Gain{tiles(level3Tiles)}, Standing: third},
		levels.Reached{Standing: third},
	}, f.told(t), "the XP, then the level it reached")
}

func TestAJumpOfManyLevelsIsOneRaise(t *testing.T) {
	f := setUp(t, true)
	require.NoError(t, f.levels.RaiseAll(t.Context(), levels.ByAccount{ada: 1}, now))
	f.took(t, level9Tiles)

	raised, err := f.raise(t, f.levels)

	require.NoError(t, err)
	assert.Equal(t, levels.Level(9), raised)
	assert.Equal(t, levels.ByAccount{ada: 9}, f.kept(t))
	assert.Equal(t, []levels.Standing{levels.Curve{}.StandingOf(level9Tiles)}, f.reached(t), "one jump is told once")
}

func TestAGuestGetsNoLevel(t *testing.T) {
	f := setUp(t, false)
	f.took(t, level9Tiles)

	raised, err := f.raise(t, f.levels)

	require.NoError(t, err)
	assert.Zero(t, raised)
	assert.Empty(t, f.kept(t))
	assert.Equal(t, []accounts.AccountID{ada}, f.accounts.Asked())
	assert.Empty(t, f.told(t))
}

func TestNoRiseAsksPlayerNothingAndWritesNothing(t *testing.T) {
	f := setUp(t, true)
	require.NoError(t, f.levels.RaiseAll(t.Context(), levels.ByAccount{ada: 3}, now))
	f.took(t, level3Tiles)

	raised, err := f.raise(t, f.levels)

	require.NoError(t, err)
	assert.Zero(t, raised)
	assert.Empty(t, f.accounts.Asked())
	assert.Equal(t, levels.ByAccount{ada: 3}, f.kept(t))
	assert.Empty(t, f.reached(t))
}

func TestAFactThatRaisesNoLevelTellsTheXPItGave(t *testing.T) {
	f := setUp(t, true)
	require.NoError(t, f.levels.RaiseAll(t.Context(), levels.ByAccount{ada: 3}, now))
	f.took(t, level3Tiles)
	f.stream = f.feed.Subscribe(t.Context(), ada, levels.Scale{}.EarnedOf(stats.StatsSnapshot{TilesTaken: level3Tiles}.Restore()))
	f.took(t, level3Tiles+5)

	_, err := f.raise(t, f.levels)

	require.NoError(t, err)
	assert.Equal(t, []levels.Told{levels.Gained{
		Gains:    []levels.Gain{tiles(5)},
		Standing: levels.Curve{}.StandingAt(3, 117),
	}}, f.told(t))
}

func TestAFactToldTwiceGivesItsXPOnce(t *testing.T) {
	f := setUp(t, true)
	require.NoError(t, f.levels.RaiseAll(t.Context(), levels.ByAccount{ada: 3}, now))
	f.took(t, level3Tiles)

	_, err := f.raise(t, f.levels)
	require.NoError(t, err)
	_, err = f.raise(t, f.levels)
	require.NoError(t, err)

	assert.Len(t, f.told(t), 1)
}

func TestTheDaysFirstTakeGivesItsTileAndItsDay(t *testing.T) {
	f := setUp(t, true)
	require.NoError(t, f.levels.RaiseAll(t.Context(), levels.ByAccount{ada: 5}, now))
	yesterday := stats.StatsSnapshot{Account: ada, TilesTaken: 800, DaysPlayed: 1}
	f.counted(t, yesterday)
	f.stream = f.feed.Subscribe(t.Context(), ada, levels.Scale{}.EarnedOf(yesterday.Restore()))
	today := stats.StatsSnapshot{Account: ada, TilesTaken: 801, DaysPlayed: 2}
	f.counted(t, today)

	_, err := f.raise(t, f.levels)

	require.NoError(t, err)
	assert.Equal(t, []levels.Told{levels.Gained{
		Gains:    []levels.Gain{tiles(1), {Source: levels.DaysPlayed{}.Name(), XP: 100}},
		Standing: levels.Curve{}.StandingAt(5, 1_001),
	}}, f.told(t))
}

func TestAnAccountWithNoStatsIsNotRaised(t *testing.T) {
	f := setUp(t, true)

	raised, err := f.raise(t, f.levels)

	require.NoError(t, err)
	assert.Zero(t, raised)
	assert.Empty(t, f.accounts.Asked())
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

func TestARaiseThatFindsAHigherLevelKeptRaisesNothing(t *testing.T) {
	f := setUp(t, true)
	f.took(t, level3Tiles)

	raised, err := f.raise(t, racedLevels{f.levels})

	require.NoError(t, err)
	assert.Zero(t, raised)
	assert.Equal(t, levels.ByAccount{ada: 9}, f.kept(t))
	assert.Empty(t, f.told(t), "the raise that wrote it tells it")
}
