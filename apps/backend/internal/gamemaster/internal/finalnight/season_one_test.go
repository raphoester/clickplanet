package finalnight_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/inmemory_moment_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/usecases/play_scoring_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule/inmemory_entry_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule/usecases/play_schedule_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season/inmemory_season_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season/usecases/play_resets_usecase"
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

type scoringTold struct {
	commands []moments.Command
}

func (s *scoringTold) Send(_ context.Context, command moments.Command) error {
	s.commands = append(s.commands, command)
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

	modules, planet, scoring := &planetAndAntibot{}, &resetsTold{}, &scoringTold{}
	boot := func() (*play_schedule_usecase.UseCase, *play_resets_usecase.UseCase, *play_scoring_usecase.UseCase) {
		return play_schedule_usecase.New(entries, seasons, modules, clock),
			play_resets_usecase.New(seasons, planet, clock),
			play_scoring_usecase.New(moments.NewPlanner(seasons, entries, kept, season.Cutoff(21*time.Hour)), kept, scoring, clock)
	}
	rules, resets, points := boot()
	tick := func() {
		_, _, err := rules.Execute(t.Context())
		require.NoError(t, err)
		_, _, err = resets.Execute(t.Context())
		require.NoError(t, err)
		_, err = points.Execute(t.Context())
		require.NoError(t, err)
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
		{Action: moments.OpenRound, Book: "season-1", Round: "season-1/day-1", Times: 1, At: seasonOneStarts},
	}}, opensAt, "no book or round opens over the break")

	clock.Advance(4 * time.Minute)
	rules, resets, points = boot()
	tick()
	assert.Equal(t, []season.Number{1, 1}, planet.of, "a boot tells it again: planet keeps season-1 and wipes nothing")
}
