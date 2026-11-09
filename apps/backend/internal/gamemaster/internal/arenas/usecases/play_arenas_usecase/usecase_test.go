package play_arenas_usecase_test

import (
	"context"
	"errors"
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
	arenas map[schedule.SetName]bool
	asked  int
	err    error
}

func (p *planet) Arena(_ context.Context, set schedule.SetName) (bool, error) {
	p.asked++
	return p.arenas[set], p.err
}

type sent struct {
	commands []moments.ArenaCommand
	err      error
}

func (s *sent) Send(_ context.Context, command moments.ArenaCommand) error {
	if s.err != nil {
		return s.err
	}
	s.commands = append(s.commands, command)
	return nil
}

type harness struct {
	clock   *cptime.FixedClock
	kept    *inmemory_moment_store.Store
	judged  *inmemory_arena_store.Store
	planet  *planet
	sent    *sent
	planner moments.Planner
	useCase *play_arenas_usecase.UseCase
}

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
		planet: &planet{arenas: map[schedule.SetName]bool{"arena-battle": true}}, sent: &sent{},
	}
	h.planner = moments.NewPlanner(seasons, entries, h.kept, h.judged, season.Cutoff(21*time.Hour))
	h.boot()
	return h
}

func (h *harness) boot() {
	h.useCase = play_arenas_usecase.New(h.planner, h.judged, h.planet, h.sent, h.clock)
}

func (h *harness) scored(t *testing.T, round moments.RoundID, at time.Time) {
	t.Helper()
	_, err := h.kept.KeepRound(t.Context(), moments.ScoredRound{Round: round, Book: "season-0", EndedAt: at})
	require.NoError(t, err)
}

func TestTheBattleThatRunsIsJudgedOnce(t *testing.T) {
	h := newHarness(t, "arena-battle")

	tick, err := h.useCase.Execute(t.Context())
	require.NoError(t, err)
	assert.Equal(t, []moments.Judgement{{Round: fought, Mode: moments.Arena}}, tick.Judged)

	h.planet.arenas["arena-battle"] = false
	h.clock.Advance(time.Minute)
	tick, err = h.useCase.Execute(t.Context())
	require.NoError(t, err)
	assert.Empty(t, tick.Judged, "a set edited later changes no judgement")
	assert.Equal(t, 1, h.planet.asked)
}

func TestABattleWhoseSetPlaysNoArenaIsPlainAndOpensNone(t *testing.T) {
	h := newHarness(t, "urf")
	h.scored(t, "season-0/day-23", starts)

	tick, err := h.useCase.Execute(t.Context())

	require.NoError(t, err)
	assert.Equal(t, []moments.Judgement{{Round: fought, Mode: moments.Plain}}, tick.Judged)
	assert.Empty(t, h.sent.commands)
}

func TestAPlanetThatCannotSayJudgesNothingAndIsAskedAgain(t *testing.T) {
	h := newHarness(t, "arena-battle")
	h.planet.err = errors.New("planet is down")

	_, err := h.useCase.Execute(t.Context())
	require.Error(t, err)

	h.planet.err = nil
	tick, err := h.useCase.Execute(t.Context())
	require.NoError(t, err)
	assert.Equal(t, []moments.Judgement{{Round: fought, Mode: moments.Arena}}, tick.Judged)
}

func TestTheArenaOpensOnceAndClosesOncePerProcessAndAgainAfterABoot(t *testing.T) {
	h := newHarness(t, "arena-battle")
	_, err := h.useCase.Execute(t.Context())
	require.NoError(t, err)
	assert.Empty(t, h.sent.commands, "day 23 is not scored yet")

	h.scored(t, "season-0/day-23", starts)
	for range 3 {
		h.clock.Advance(time.Second)
		_, err = h.useCase.Execute(t.Context())
		require.NoError(t, err)
	}
	open := moments.ArenaCommand{Action: moments.OpenArena, Arena: fought, At: starts, EndsAt: ends}
	assert.Equal(t, []moments.ArenaCommand{open}, h.sent.commands)

	h.clock.Advance(ends.Sub(h.clock.Now()))
	h.scored(t, fought, ends)
	_, err = h.useCase.Execute(t.Context())
	require.NoError(t, err)
	closing := moments.ArenaCommand{Action: moments.CloseArena, Arena: fought, At: ends, EndsAt: ends}
	assert.Equal(t, []moments.ArenaCommand{open, closing}, h.sent.commands)

	h.boot()
	_, err = h.useCase.Execute(t.Context())
	require.NoError(t, err)
	assert.Equal(t, []moments.ArenaCommand{open, closing, closing}, h.sent.commands, "a boot closes again until planet tells it")
}

func TestACommandRedisRefusedIsSentAgain(t *testing.T) {
	h := newHarness(t, "arena-battle")
	h.scored(t, "season-0/day-23", starts)
	_, err := h.useCase.Execute(t.Context())
	require.NoError(t, err)
	h.sent.commands = nil
	h.boot()
	h.sent.err = errors.New("redis is down")

	_, err = h.useCase.Execute(t.Context())
	require.Error(t, err)

	h.sent.err = nil
	_, err = h.useCase.Execute(t.Context())
	require.NoError(t, err)
	assert.Equal(t, []moments.ArenaCommand{{Action: moments.OpenArena, Arena: fought, At: starts, EndsAt: ends}}, h.sent.commands)
}
