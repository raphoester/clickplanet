package inprocess_level_feed_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/levels"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/levels/inprocess_level_feed"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats"
)

var (
	ada = accounts.AccountID{15: 1}
	bob = accounts.AccountID{15: 2}

	fifth = levels.Curve{}.StandingOf(900)
)

func received(t *testing.T, told <-chan levels.Told) levels.Told {
	t.Helper()

	select {
	case level := <-told:
		return level
	case <-time.After(time.Second):
		require.Fail(t, "nothing was received")
		return nil
	}
}

func silent(t *testing.T, told <-chan levels.Told) {
	t.Helper()

	select {
	case level := <-told:
		assert.Fail(t, "something was told", "%v", level)
	default:
	}
}

func earned(snapshot stats.StatsSnapshot) levels.Earned {
	return levels.Scale{}.EarnedOf(snapshot.Restore())
}

func tiles(xp levels.XP) levels.Gain { return levels.Gain{Source: levels.TilesTaken{}.Name(), XP: xp} }

func TestEveryStreamOfTheAccountGetsItsLevelAndNoOtherAccountDoes(t *testing.T) {
	feed := inprocess_level_feed.New()
	first, second := feed.Subscribe(t.Context(), ada, nil), feed.Subscribe(t.Context(), ada, nil)
	other := feed.Subscribe(t.Context(), bob, nil)

	feed.Publish(ada, fifth)

	assert.Equal(t, levels.Reached{Standing: fifth}, received(t, first))
	assert.Equal(t, levels.Reached{Standing: fifth}, received(t, second))
	silent(t, other)
}

func TestAFactToldTwiceOrReadLateTellsNothing(t *testing.T) {
	feed := inprocess_level_feed.New()
	told := feed.Subscribe(t.Context(), ada, nil)
	feed.Count(ada, earned(stats.StatsSnapshot{TilesTaken: 10}), fifth)
	received(t, told)

	feed.Count(ada, earned(stats.StatsSnapshot{TilesTaken: 10}), fifth)
	feed.Count(ada, earned(stats.StatsSnapshot{TilesTaken: 8}), fifth)

	silent(t, told)
	feed.Count(ada, earned(stats.StatsSnapshot{TilesTaken: 11}), fifth)
	assert.Equal(t, levels.Gained{Gains: []levels.Gain{tiles(1)}, Standing: fifth}, received(t, told))
}

func TestASecondStreamKeepsWhatTheFirstKnows(t *testing.T) {
	feed := inprocess_level_feed.New()
	first := feed.Subscribe(t.Context(), ada, earned(stats.StatsSnapshot{TilesTaken: 20}))
	second := feed.Subscribe(t.Context(), ada, earned(stats.StatsSnapshot{TilesTaken: 10}))

	feed.Count(ada, earned(stats.StatsSnapshot{TilesTaken: 21}), fifth)

	assert.Equal(t, levels.Gained{Gains: []levels.Gain{tiles(1)}, Standing: fifth}, received(t, first))
	assert.Equal(t, levels.Gained{Gains: []levels.Gain{tiles(1)}, Standing: fifth}, received(t, second))
}

func TestXPCountedWithNoStreamOpenIsForgotten(t *testing.T) {
	feed := inprocess_level_feed.New()
	ctx, cancel := context.WithCancel(t.Context())
	closed := feed.Subscribe(ctx, ada, earned(stats.StatsSnapshot{TilesTaken: 10}))
	cancel()
	for range closed {
	}
	feed.Count(ada, earned(stats.StatsSnapshot{TilesTaken: 30}), fifth)

	told := feed.Subscribe(t.Context(), ada, earned(stats.StatsSnapshot{TilesTaken: 25}))
	feed.Count(ada, earned(stats.StatsSnapshot{TilesTaken: 31}), fifth)

	assert.Equal(t, levels.Gained{Gains: []levels.Gain{tiles(6)}, Standing: fifth}, received(t, told),
		"from what the new stream read, not from what was counted while none was open")
}

func TestTheGainAndTheLevelComeInTheOrderTheyWereTold(t *testing.T) {
	feed := inprocess_level_feed.New()
	told := feed.Subscribe(t.Context(), ada, nil)

	feed.Count(ada, earned(stats.StatsSnapshot{TilesTaken: 900}), fifth)
	feed.Publish(ada, fifth)

	assert.IsType(t, levels.Gained{}, received(t, told))
	assert.Equal(t, levels.Reached{Standing: fifth}, received(t, told))
}

func TestAClosedStreamIsForgottenAndItsChannelClosed(t *testing.T) {
	feed := inprocess_level_feed.New()
	ctx, cancel := context.WithCancel(t.Context())
	told := feed.Subscribe(ctx, ada, nil)

	cancel()

	require.Eventually(t, func() bool {
		select {
		case _, open := <-told:
			return !open
		default:
			return false
		}
	}, time.Second, time.Millisecond)
	feed.Publish(ada, fifth)
	feed.Count(ada, earned(stats.StatsSnapshot{TilesTaken: 1}), fifth)
}

func TestAFullStreamDropsRatherThanBlocks(t *testing.T) {
	feed := inprocess_level_feed.New()
	told := feed.Subscribe(t.Context(), ada, nil)

	for range 50 {
		feed.Publish(ada, fifth)
	}

	assert.Len(t, told, 32)
}
