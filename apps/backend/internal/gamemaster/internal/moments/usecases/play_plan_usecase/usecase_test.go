package play_plan_usecase_test

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
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/usecases/play_maps_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/usecases/play_plan_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/usecases/play_scoring_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule/inmemory_entry_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season/inmemory_season_store"
)

var (
	starts = time.Date(2026, 10, 30, 21, 0, 0, 0, time.UTC)
	now    = starts.Add(time.Second)
	entry  = schedule.EntryID(uuid.MustParse("0192f5a0-7c00-7000-8000-000000000032"))
	fought = moments.BattleRoundOf(entry)
)

type steps struct {
	ran      []string
	unjudged []bool
	judged   []moments.Judgement
	err      error
}

func (s *steps) saw(step string, plan moments.Plan) {
	s.ran = append(s.ran, step)
	_, unjudged := plan.Unjudged(now)
	s.unjudged = append(s.unjudged, unjudged)
}

type judge struct{ *steps }

func (j judge) Execute(_ context.Context, plan moments.Plan) ([]moments.Judgement, error) {
	j.saw("judge", plan)
	return j.judged, nil
}

type resets struct{ *steps }

func (r resets) Execute(_ context.Context, plan moments.Plan) (season.Window, bool, error) {
	r.saw("resets", plan)
	return season.Window{}, false, r.err
}

type arenas struct{ *steps }

func (a arenas) Execute(_ context.Context, plan moments.Plan) ([]play_arenas_usecase.Call, error) {
	a.saw("arenas", plan)
	return nil, nil
}

type scoring struct{ *steps }

func (s scoring) Execute(_ context.Context, plan moments.Plan) ([]play_scoring_usecase.Call, error) {
	s.saw("scoring", plan)
	return nil, s.err
}

type starting struct{ *steps }

func (s starting) Execute(_ context.Context, plan moments.Plan) ([]moments.Moment, error) {
	s.saw("starts", plan)
	return nil, nil
}

type counted struct {
	planner moments.Planner
	reads   int
}

func (c *counted) Plan(ctx context.Context) (moments.Plan, error) {
	c.reads++
	return c.planner.Plan(ctx) //nolint:wrapcheck // a count, not a sentence.
}

type harness struct {
	seasons *inmemory_season_store.Store
	planner *counted
	steps   *steps
	plays   *play_plan_usecase.UseCase
	maps    *play_maps_usecase.UseCase
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	seasons, entries := inmemory_season_store.New(), inmemory_entry_store.New()
	require.NoError(t, seasons.Save(t.Context(), season.Snapshot{Number: 0, Length: 23*24*time.Hour + 23*time.Hour}.Restore()))
	require.NoError(t, entries.Save(t.Context(), schedule.EntrySnapshot{
		ID: entry, Set: "arena-battle", Kind: schedule.Battle, StartsAt: starts, EndsAt: starts.Add(2 * time.Hour),
		Priority: 1, Times: 2, MadeAt: starts.Add(-24 * time.Hour),
	}.Restore()))
	h := &harness{
		seasons: seasons,
		planner: &counted{planner: moments.NewPlanner(seasons, entries, inmemory_moment_store.New(),
			inmemory_arena_store.New(), season.Cutoff(21*time.Hour))},
		steps: &steps{},
	}
	h.plays = play_plan_usecase.New(h.planner, judge{h.steps}, scoring{h.steps}, starting{h.steps})
	h.maps = play_maps_usecase.New(h.planner, resets{h.steps}, arenas{h.steps})
	return h
}

func TestEachStepGetsThePlanReadOnceThisTickInOrder(t *testing.T) {
	h := newHarness(t)

	require.NoError(t, h.plays.Execute(t.Context()))
	assert.Equal(t, 1, h.planner.reads)
	require.NoError(t, h.maps.Execute(t.Context()))
	assert.Equal(t, 2, h.planner.reads, "the map swaps read their own plan")

	assert.Equal(t, []string{"judge", "scoring", "starts", "resets", "arenas"}, h.steps.ran)
}

func TestTheStepsAfterTheJudgeGetTheBattleItJudged(t *testing.T) {
	h := newHarness(t)
	h.steps.judged = []moments.Judgement{{Round: fought, Mode: moments.Arena}}

	require.NoError(t, h.plays.Execute(t.Context()))

	assert.Equal(t, []bool{true, false, false}, h.steps.unjudged)
	assert.Equal(t, 1, h.planner.reads, "the judgement is not read back")
}

func TestAStepThatFailsStopsNoneAfterIt(t *testing.T) {
	h := newHarness(t)
	down := errors.New("scoring is down")
	h.steps.err = down

	plays := h.plays.Execute(t.Context())
	maps := h.maps.Execute(t.Context())

	require.ErrorIs(t, plays, down)
	require.NotErrorIs(t, plays, play_plan_usecase.ErrUnplanned)
	require.ErrorIs(t, maps, down)
	assert.Equal(t, []string{"judge", "scoring", "starts", "resets", "arenas"}, h.steps.ran)
}

func TestAPlanThatCannotBeReadRunsNoStep(t *testing.T) {
	h := newHarness(t)
	down := errors.New("postgres is down")
	h.seasons.FailWith(down)

	plays := h.plays.Execute(t.Context())
	maps := h.maps.Execute(t.Context())

	require.ErrorIs(t, plays, play_plan_usecase.ErrUnplanned)
	require.ErrorIs(t, plays, down)
	require.ErrorIs(t, maps, play_plan_usecase.ErrUnplanned)
	assert.Empty(t, h.steps.ran)
}
