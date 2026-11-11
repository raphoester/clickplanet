package judge_battles_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/arenas/inmemory_arena_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/arenas/usecases/judge_battles_usecase"
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

type harness struct {
	clock   *cptime.FixedClock
	judged  *inmemory_arena_store.Store
	planet  *planet
	planner moments.Planner
	useCase *judge_battles_usecase.UseCase
}

// A battle of set runs from 21:00 to 23:00.
func newHarness(t *testing.T, set schedule.SetName) *harness {
	t.Helper()
	seasons, entries := inmemory_season_store.New(), inmemory_entry_store.New()
	require.NoError(t, seasons.Save(t.Context(), season.Snapshot{Number: 0, Length: 23*24*time.Hour + 23*time.Hour}.Restore()))
	require.NoError(t, entries.Save(t.Context(), schedule.EntrySnapshot{
		ID: entry, Set: set, Kind: schedule.Battle, StartsAt: starts, EndsAt: starts.Add(2 * time.Hour), Priority: 1, Times: 2,
		MadeAt: starts.Add(-24 * time.Hour),
	}.Restore()))
	h := &harness{
		clock: cptime.NewFixedClock(starts), judged: inmemory_arena_store.New(),
		planet: &planet{arenas: map[schedule.SetName]bool{"arena-battle": true}},
	}
	h.planner = moments.NewPlanner(seasons, entries, inmemory_moment_store.New(), h.judged, season.Cutoff(21*time.Hour))
	h.useCase = judge_battles_usecase.New(h.judged, h.planet, h.clock)
	return h
}

func (h *harness) execute(t *testing.T) ([]moments.Judgement, error) {
	t.Helper()
	plan, err := h.planner.Plan(t.Context())
	require.NoError(t, err)
	return h.useCase.Execute(t.Context(), plan)
}

func TestTheBattleThatRunsIsJudgedOnce(t *testing.T) {
	h := newHarness(t, "arena-battle")

	judged, err := h.execute(t)
	require.NoError(t, err)
	assert.Equal(t, []moments.Judgement{{Round: fought, Mode: moments.Arena}}, judged)

	h.planet.arenas["arena-battle"] = false
	h.clock.Advance(time.Minute)
	judged, err = h.execute(t)
	require.NoError(t, err)
	assert.Empty(t, judged, "a set edited later changes no judgement")
	assert.Equal(t, 1, h.planet.asked)
}

func TestABattleWhoseSetPlaysNoArenaIsPlain(t *testing.T) {
	h := newHarness(t, "urf")

	judged, err := h.execute(t)

	require.NoError(t, err)
	assert.Equal(t, []moments.Judgement{{Round: fought, Mode: moments.Plain}}, judged)
}

func TestAPlanetThatCannotSayJudgesNothingAndIsAskedAgain(t *testing.T) {
	h := newHarness(t, "arena-battle")
	h.planet.err = errors.New("planet is down")

	_, err := h.execute(t)
	require.Error(t, err)

	h.planet.err = nil
	judged, err := h.execute(t)
	require.NoError(t, err)
	assert.Equal(t, []moments.Judgement{{Round: fought, Mode: moments.Arena}}, judged)
}
