package finalnight_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/arenas/inmemory_arena_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/arenas/usecases/judge_battles_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/arenas/usecases/play_arenas_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/inmemory_moment_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/outbox_moment_events"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/usecases/play_maps_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/usecases/play_plan_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/usecases/play_scoring_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/usecases/tell_starts_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule/inmemory_entry_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule/usecases/play_schedule_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season/inmemory_season_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season/usecases/play_resets_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var seasonOneStarts = seasonEnds.Add(16 * time.Hour)

type resetsTold struct {
	at []time.Time
	of []season.Number
}

func (r *resetsTold) Reset(_ context.Context, starts season.Window) error {
	r.at, r.of = append(r.at, starts.StartsAt), append(r.of, starts.Number)
	return nil
}

func TestTheMapIsFrozenOverTheBreakAndResetAsSeasonOneStarts(t *testing.T) {
	clock := cptime.NewFixedClock(seasonEnds.Add(-time.Second))
	seasons, entries, kept := inmemory_season_store.New(), inmemory_entry_store.New(), inmemory_moment_store.New()
	require.NoError(t, seasons.Save(t.Context(), seasonZero.Restore()))
	require.NoError(t, seasons.Save(t.Context(), season.Snapshot{Number: 1, Break: 16 * time.Hour, Length: 28 * 24 * time.Hour}.Restore()))
	require.NoError(t, entries.Save(t.Context(), schedule.EntrySnapshot{
		ID: schedule.EntryID(uuid.UUID{15: 1}), Set: "frozen", Kind: schedule.Plain, StartsAt: seasonEnds, EndsAt: seasonOneStarts,
		Priority: 1, MadeAt: seasonEnds.Add(-24 * time.Hour),
	}.Restore()))

	modules, planet, scoring := &planetAndAntibot{}, &resetsTold{}, &scoring{}
	judged := inmemory_arena_store.New()
	_, err := judged.KeepJudged(t.Context(), moments.Judgement{Round: moments.FinaleRoundOf(0), Mode: moments.Plain})
	require.NoError(t, err, "the finale was judged as it started, before this test's last second of it")
	told := outbox_moment_events.New(&cpatomic.FakeOutbox{})
	boot := func() (*play_schedule_usecase.UseCase, *play_plan_usecase.UseCase, *play_maps_usecase.UseCase) {
		planner := moments.NewPlanner(seasons, entries, kept, judged, season.Cutoff(21*time.Hour))
		return play_schedule_usecase.New(entries, seasons, planner, modules, clock, time.Minute),
			play_plan_usecase.New(planner, judge_battles_usecase.New(judged, rulesOf{}, clock),
				play_scoring_usecase.New(kept, scoring, told, cpatomic.FakeTransactioner{}, clock, time.Minute),
				tell_starts_usecase.New(kept, told, cpatomic.FakeTransactioner{}, clock)),
			play_maps_usecase.New(planner,
				play_resets_usecase.New(kept, planet, clock, time.Minute),
				play_arenas_usecase.New(judged, &arenaCommands{}, clock, time.Minute))
	}
	rules, plan, maps := boot()
	tick := func() {
		_, _, err := rules.Execute(t.Context())
		require.NoError(t, err)
		require.NoError(t, plan.Execute(t.Context()))
		require.NoError(t, maps.Execute(t.Context()))
	}
	tick()
	scoring.commands = nil

	opensAt := map[time.Time][]moments.Command{}
	for clock.Now().Before(seasonOneStarts.Add(time.Minute)) {
		clock.Advance(time.Second)
		before := len(scoring.commands)
		tick()
		for _, command := range scoring.commands[before:] {
			if command.Action == moments.OpenBook || command.Action == moments.OpenRound {
				opensAt[clock.Now()] = append(opensAt[clock.Now()], command)
			}
		}
	}

	assert.Equal(t, []played{
		{set: "rush", at: seasonEnds.Add(-time.Second)},
		{set: "frozen", at: seasonEnds},
		{set: schedule.Default, at: seasonOneStarts},
	}, modules.played, "the finale, then the freeze from 23:00, then the rules of season 1 from 15:00")
	assert.Equal(t, []season.Number{1}, planet.of, "the map is reset once, as season 1 starts")
	assert.Equal(t, []time.Time{seasonOneStarts}, planet.at)
	assert.Equal(t, map[time.Time][]moments.Command{seasonOneStarts: {
		{Action: moments.OpenBook, Book: "season-1", At: seasonOneStarts},
		{Action: moments.OpenRound, Book: "season-1", Round: "season-1/day-1", Times: 1, At: seasonOneStarts, Ranking: moments.RankedByHeldAndGain},
	}}, opensAt, "no book or round opens over the break")

	clock.Advance(4 * time.Minute)
	rules, plan, maps = boot()
	tick()
	assert.Equal(t, []season.Number{1}, planet.of, "a boot asks no reset planet answered")
}
