package play_arenas_usecase_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/arenas/inmemory_arena_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/arenas/usecases/play_arenas_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/inmemory_moment_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule/inmemory_entry_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season/inmemory_season_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	starts = time.Date(2026, 10, 30, 21, 0, 0, 0, time.UTC)
	ends   = starts.Add(2 * time.Hour)
	entry  = schedule.EntryID(uuid.MustParse("0192f5a0-7c00-7000-8000-000000000032"))
	fought = moments.BattleRoundOf(entry)
)

type planet struct {
	asked   []moments.ArenaCommand
	down    error
	refuses moments.ArenaAction
}

func (p *planet) Ask(_ context.Context, command moments.ArenaCommand) error {
	p.asked = append(p.asked, command)
	if command.Action == p.refuses {
		return fmt.Errorf("%w: closed already", moments.ErrRefused)
	}
	return p.down
}

type harness struct {
	clock   *cptime.FixedClock
	kept    *inmemory_moment_store.Store
	judged  *inmemory_arena_store.Store
	planet  *planet
	planner moments.Planner
	useCase *play_arenas_usecase.UseCase
}

// The battle runs from 21:00 to 23:00 and is judged an arena battle.
func newHarness(t *testing.T, set schedule.SetName) *harness {
	t.Helper()
	seasons, entries := inmemory_season_store.New(), inmemory_entry_store.New()
	require.NoError(t, seasons.Save(t.Context(), season.Snapshot{Number: 0, Length: 23*24*time.Hour + 23*time.Hour}.Restore()))
	require.NoError(t, entries.Save(t.Context(), schedule.EntrySnapshot{
		ID: entry, Set: set, Kind: schedule.Battle, StartsAt: starts, EndsAt: ends, Priority: 1, Times: 2,
		MadeAt: starts.Add(-24 * time.Hour),
	}.Restore()))
	h := &harness{
		clock: cptime.NewFixedClock(starts), kept: inmemory_moment_store.New(), judged: inmemory_arena_store.New(),
		planet: &planet{},
	}
	_, err := h.judged.KeepJudged(t.Context(), moments.Judgement{Round: fought, Mode: moments.Arena})
	require.NoError(t, err)
	h.planner = moments.NewPlanner(seasons, entries, h.kept, h.judged, season.Cutoff(21*time.Hour))
	h.boot()
	return h
}

func (h *harness) boot() {
	h.useCase = play_arenas_usecase.New(h.judged, h.planet, h.clock, time.Minute)
}

func (h *harness) execute(t *testing.T) ([]play_arenas_usecase.Call, error) {
	t.Helper()
	plan, err := h.planner.Plan(t.Context())
	require.NoError(t, err)
	return h.useCase.Execute(t.Context(), plan)
}

func (h *harness) scored(t *testing.T, round moments.RoundID, at time.Time) {
	t.Helper()
	_, err := h.kept.KeepRound(t.Context(), moments.ScoredRound{Round: round, Book: "season-0", EndedAt: at})
	require.NoError(t, err)
}

func TestTheArenaOpensOnceAndClosesOnceAndEachAnswerIsKept(t *testing.T) {
	h := newHarness(t, "arena-battle")
	_, err := h.execute(t)
	require.NoError(t, err)
	assert.Empty(t, h.planet.asked, "day 23 is not scored yet")

	h.scored(t, "season-0/day-23", starts)
	for range 3 {
		h.clock.Advance(time.Second)
		_, err = h.execute(t)
		require.NoError(t, err)
	}
	open := moments.ArenaCommand{Action: moments.OpenArena, Arena: fought, At: starts, EndsAt: ends, Set: "arena-battle"}
	assert.Equal(t, []moments.ArenaCommand{open}, h.planet.asked)

	h.clock.Advance(ends.Sub(h.clock.Now()))
	h.scored(t, fought, ends)
	for range 3 {
		_, err = h.execute(t)
		require.NoError(t, err)
		h.clock.Advance(time.Minute)
	}
	closing := moments.ArenaCommand{Action: moments.CloseArena, Arena: fought, At: ends, EndsAt: ends}
	assert.Equal(t, []moments.ArenaCommand{open, closing}, h.planet.asked)

	h.boot()
	_, err = h.execute(t)
	require.NoError(t, err)
	kept, err := h.judged.ArenasKept(t.Context())
	require.NoError(t, err)
	assert.Equal(t, []moments.RoundID{fought}, kept.Opened)
	assert.Equal(t, []moments.RoundID{fought}, kept.Closed)
	assert.Len(t, h.planet.asked, 2, "a boot calls nothing planet answered")
}

func TestAnArenaWhoseRoundScoringMissedCloses(t *testing.T) {
	h := newHarness(t, "arena-battle")
	_, err := h.execute(t)
	require.NoError(t, err)

	h.clock.Advance(ends.Sub(h.clock.Now()))
	_, err = h.kept.KeepMissedRound(t.Context(), moments.MissedRound{Round: fought, EndedAt: ends})
	require.NoError(t, err)
	_, err = h.execute(t)
	require.NoError(t, err)

	assert.Equal(t, []moments.ArenaCommand{{Action: moments.CloseArena, Arena: fought, At: ends, EndsAt: ends}}, h.planet.asked)
}

func TestACallPlanetDidNotAnswerIsMadeAgainAMinuteLater(t *testing.T) {
	h := newHarness(t, "arena-battle")
	h.scored(t, "season-0/day-23", starts)
	down := errors.New("planet is down")
	h.planet.down = down

	_, err := h.execute(t)
	require.ErrorIs(t, err, down)
	h.planet.down = nil
	h.clock.Advance(59 * time.Second)
	_, err = h.execute(t)
	require.NoError(t, err)
	assert.Len(t, h.planet.asked, 1, "the call waits a minute")

	h.clock.Advance(time.Second)
	calls, err := h.execute(t)
	require.NoError(t, err)
	open := moments.ArenaCommand{Action: moments.OpenArena, Arena: fought, At: starts, EndsAt: ends, Set: "arena-battle"}
	assert.Equal(t, []play_arenas_usecase.Call{{Command: open}}, calls)
	assert.Equal(t, []moments.ArenaCommand{open, open}, h.planet.asked)
}

func TestAnOpenPlanetRefusesIsNotCalledAgainAndOpensNothing(t *testing.T) {
	h := newHarness(t, "arena-battle")
	h.scored(t, "season-0/day-23", starts)
	h.planet.refuses = moments.OpenArena

	calls, err := h.execute(t)
	require.NoError(t, err)
	require.Len(t, calls, 1)
	require.ErrorIs(t, calls[0].Refusal, moments.ErrRefused)

	h.clock.Advance(time.Hour)
	_, err = h.execute(t)
	require.NoError(t, err)
	assert.Len(t, h.planet.asked, 1)
	kept, err := h.judged.ArenasKept(t.Context())
	require.NoError(t, err)
	assert.Empty(t, kept.Opened)
}
