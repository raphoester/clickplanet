package fills_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/fills"
)

var at = time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC)

func TestAFillStartsOncePerRequest(t *testing.T) {
	started, err := fills.NewFill("season-0:finale:fill", fills.SourceInternal, at, fills.Fill{})
	require.NoError(t, err)
	assert.Equal(t, fills.Snapshot{ID: "season-0:finale:fill", Source: fills.SourceInternal, StartedAt: at},
		started.TakeSnapshot())
	assert.True(t, started.Running())

	again, err := fills.NewFill("season-0:finale:fill", fills.SourceOperator, at.Add(time.Hour), started)
	require.ErrorIs(t, err, fills.ErrStartedAlready)
	assert.Equal(t, started, again, "the fill sent again is the one kept, not a new start")
}

func TestAFillNeedsARequestOfAtMost128Characters(t *testing.T) {
	_, err := fills.NewFill("", fills.SourceOperator, at, fills.Fill{})
	require.ErrorIs(t, err, fills.ErrNoRequest)

	_, err = fills.NewFill(bonuses.FillID(strings.Repeat("é", 129)), fills.SourceOperator, at, fills.Fill{})
	require.ErrorIs(t, err, fills.ErrLongRequest)

	_, err = fills.NewFill(bonuses.FillID(strings.Repeat("é", 128)), fills.SourceOperator, at, fills.Fill{})
	require.NoError(t, err)
}

func TestAFillGoesThroughThePlayersBeforeItPageByPageUntilAPageIsNotFull(t *testing.T) {
	fill, err := fills.NewFill("a1b2", fills.SourceOperator, at, fills.Fill{})
	require.NoError(t, err)

	first := fill.NextPage(2)
	assert.Equal(t, fills.Page{Before: at, After: "", Size: 2}, first)
	fill = fill.Past(first, []bonuses.Holder{"a", "b"}, at.Add(time.Second))
	assert.True(t, fill.Running(), "a full page may have more after it")

	second := fill.NextPage(2)
	assert.Equal(t, fills.Page{Before: at, After: "b", Size: 2}, second)
	fill = fill.Past(second, []bonuses.Holder{"c"}, at.Add(2*time.Second))

	assert.False(t, fill.Running())
	assert.Equal(t, fills.Snapshot{
		ID: "a1b2", Source: fills.SourceOperator, StartedAt: at,
		Reached: "c", Filled: 3, FinishedAt: at.Add(2 * time.Second),
	}, fill.TakeSnapshot())
}

func TestAFillOfNobodyIsFinishedAtItsFirstPage(t *testing.T) {
	fill, err := fills.NewFill("a1b2", fills.SourceOperator, at, fills.Fill{})
	require.NoError(t, err)

	fill = fill.Past(fill.NextPage(500), nil, at)

	assert.False(t, fill.Running())
	assert.Zero(t, fill.TakeSnapshot().Filled)
}

func TestAKeptFillIsRestoredOnlyWhenItIsOne(t *testing.T) {
	for _, kept := range []fills.Snapshot{
		{ID: "a1b2", Source: "seasons", StartedAt: at},
		{ID: "a1b2", Source: fills.SourceOperator},
		{ID: "a1b2", Source: fills.SourceOperator, StartedAt: at, Filled: -1},
	} {
		_, err := kept.Restore()

		require.ErrorIs(t, err, fills.ErrInvalidFill, "%+v", kept)
	}
}
