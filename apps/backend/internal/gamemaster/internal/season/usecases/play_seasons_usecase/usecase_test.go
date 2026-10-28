package play_seasons_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season/inmemory_season_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season/usecases/play_seasons_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	seasonZeroEnds = time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC)
	zero           = season.Snapshot{Number: 0, Length: seasonZeroEnds.Sub(season.Origin)}
	one            = season.Snapshot{Number: 1, Break: time.Hour, Length: 30 * 24 * time.Hour}
	zeroWindow     = season.Window{Number: 0, StartsAt: season.Origin, EndsAt: seasonZeroEnds}
	oneWindow      = season.Window{Number: 1, StartsAt: seasonZeroEnds.Add(time.Hour), EndsAt: seasonZeroEnds.Add(time.Hour + 30*24*time.Hour)}
)

type command struct {
	window    season.Window
	decidedAt time.Time
}

type commanded struct {
	sent []command
	err  error
}

func (c *commanded) Play(_ context.Context, window season.Window, decidedAt time.Time) error {
	if c.err != nil {
		return c.err
	}
	c.sent = append(c.sent, command{window: window, decidedAt: decidedAt})
	return nil
}

type harness struct {
	t        *testing.T
	clock    *cptime.FixedClock
	seasons  *inmemory_season_store.Store
	commands *commanded
	useCase  *play_seasons_usecase.UseCase
}

func newHarness(t *testing.T, at time.Time) *harness {
	t.Helper()

	h := &harness{t: t, clock: cptime.NewFixedClock(at), seasons: inmemory_season_store.New(), commands: &commanded{}}
	h.useCase = play_seasons_usecase.New(h.seasons, h.commands, h.clock)
	h.save(zero, one)
	return h
}

func (h *harness) save(snapshots ...season.Snapshot) {
	h.t.Helper()
	for _, snapshot := range snapshots {
		require.NoError(h.t, h.seasons.Save(h.t.Context(), snapshot.Restore()))
	}
}

func (h *harness) execute() int {
	h.t.Helper()
	told, err := h.useCase.Execute(h.t.Context())
	require.NoError(h.t, err)
	return len(told)
}

func TestTheFirstExecuteTellsEveryStartedSeason(t *testing.T) {
	now := oneWindow.StartsAt.Add(time.Minute)
	h := newHarness(t, now)

	assert.Equal(t, 2, h.execute(), "scoring may have missed a command before the boot")

	assert.Equal(t, []command{{window: zeroWindow, decidedAt: now}, {window: oneWindow, decidedAt: now}}, h.commands.sent)
}

func TestASeasonToComeIsNotTold(t *testing.T) {
	h := newHarness(t, seasonZeroEnds.Add(-time.Hour))

	h.execute()

	assert.Equal(t, []command{{window: zeroWindow, decidedAt: seasonZeroEnds.Add(-time.Hour)}}, h.commands.sent)
}

func TestNothingIsSentWhileNoSeasonChanges(t *testing.T) {
	h := newHarness(t, seasonZeroEnds.Add(-time.Hour))
	h.execute()

	for range 3 {
		h.clock.Advance(time.Second)
		assert.Zero(t, h.execute())
	}
}

func TestASeasonIsToldAtItsStartAfterItsBreak(t *testing.T) {
	h := newHarness(t, oneWindow.StartsAt.Add(-time.Second))
	h.execute()

	h.clock.Advance(time.Second)

	assert.Equal(t, 1, h.execute())
	assert.Equal(t, command{window: oneWindow, decidedAt: oneWindow.StartsAt}, h.commands.sent[1])
}

func TestAMovedEndIsToldAgain(t *testing.T) {
	h := newHarness(t, seasonZeroEnds.Add(-time.Hour))
	h.execute()
	moved := zero
	moved.Length += 30 * time.Minute
	h.save(moved)

	h.clock.Advance(time.Second)

	assert.Equal(t, 1, h.execute())
	assert.Equal(t, command{
		window:    season.Window{Number: 0, StartsAt: season.Origin, EndsAt: seasonZeroEnds.Add(30 * time.Minute)},
		decidedAt: seasonZeroEnds.Add(-time.Hour + time.Second),
	}, h.commands.sent[1])
}

func TestACommandRedisDidNotTakeIsSentAtTheNextExecute(t *testing.T) {
	h := newHarness(t, seasonZeroEnds.Add(-time.Hour))
	down := errors.New("redis is down")
	h.commands.err = down

	_, err := h.useCase.Execute(t.Context())
	require.ErrorIs(t, err, down)

	h.commands.err = nil
	assert.Equal(t, 1, h.execute())
}

func TestSeasonsThatCannotBeReadSendNothing(t *testing.T) {
	h := newHarness(t, seasonZeroEnds.Add(-time.Hour))
	down := errors.New("postgres is down")
	h.seasons.FailWith(down)

	_, err := h.useCase.Execute(t.Context())

	require.ErrorIs(t, err, down)
	assert.Empty(t, h.commands.sent)
}
