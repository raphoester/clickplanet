package finalnight_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	gamemasterv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1"
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

var (
	finaleStarts = time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC)
	seasonEnds   = time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC)
	// Season 0 as migrations 20261028120000_seasons and 20261028150000_season_finales seed it.
	seasonZero = season.Snapshot{
		Number: 0, Length: 23*24*time.Hour + 23*time.Hour,
		Finale: season.Finale{Length: 2 * time.Hour, Set: "rush", Description: "Triple points.", Times: 3},
	}
)

type played struct {
	set schedule.SetName
	at  time.Time
}

type planetAndAntibot struct {
	played []played
}

func (p *planetAndAntibot) PlayAntibot(context.Context, schedule.Turn, time.Time) error { return nil }

func (p *planetAndAntibot) Play(_ context.Context, turn schedule.Turn, decidedAt time.Time) error {
	if len(p.played) > 0 && !decidedAt.After(p.played[len(p.played)-1].at) {
		return nil
	}
	p.played = append(p.played, played{set: turn.TakeSnapshot().Set, at: decidedAt})
	return nil
}

// scoring scores each round and closes each book as it is called, and answers at once.
type scoring struct {
	commands []moments.Command
}

func (s *scoring) Ask(_ context.Context, command moments.Command) (moments.Answer, error) {
	s.commands = append(s.commands, command)
	return moments.Answer{EndedAt: command.At}, nil
}

// rulesOf answers whether a set makes an arena battle: none of season 0's does.
type rulesOf map[schedule.SetName]bool

func (r rulesOf) Arena(_ context.Context, set schedule.SetName) (bool, error) {
	return r[set], nil
}

type arenaCommands struct {
	commands []moments.ArenaCommand
}

func (a *arenaCommands) Ask(_ context.Context, command moments.ArenaCommand) error {
	a.commands = append(a.commands, command)
	return nil
}

func TestSeasonZerosLastDayAndItsFinalePlaySecondBySecond(t *testing.T) {
	clock := cptime.NewFixedClock(finaleStarts.Add(-24*time.Hour - time.Second))
	seasons, entries, kept := inmemory_season_store.New(), inmemory_entry_store.New(), inmemory_moment_store.New()
	require.NoError(t, seasons.Save(t.Context(), seasonZero.Restore()))
	judged := inmemory_arena_store.New()
	planner := moments.NewPlanner(seasons, entries, kept, judged, season.Cutoff(21*time.Hour))
	outbox := &cpatomic.FakeOutbox{}
	told := outbox_moment_events.New(outbox)

	modules := &planetAndAntibot{}
	rules := play_schedule_usecase.New(entries, seasons, planner, modules, clock, time.Minute)
	scored, planet := &scoring{}, &arenaCommands{}
	plan := play_plan_usecase.New(planner, judge_battles_usecase.New(judged, rulesOf{}, clock),
		play_scoring_usecase.New(kept, scored, told, cpatomic.FakeTransactioner{}, clock, time.Minute),
		tell_starts_usecase.New(kept, outbox_moment_events.New(&cpatomic.FakeOutbox{}), cpatomic.FakeTransactioner{}, clock))
	maps := play_maps_usecase.New(planner,
		play_resets_usecase.New(kept, &resetsTold{}, clock, time.Minute),
		play_arenas_usecase.New(judged, planet, clock, time.Minute))

	tick := func() {
		_, _, err := rules.Execute(t.Context())
		require.NoError(t, err)
		require.NoError(t, plan.Execute(t.Context()))
		require.NoError(t, maps.Execute(t.Context()))
	}
	tick()
	scored.commands = nil
	before := len(outbox.Recorded())

	commandedAt := map[time.Time][]moments.Command{}
	for clock.Now().Before(seasonEnds.Add(time.Minute)) {
		clock.Advance(time.Second)
		before := len(scored.commands)
		tick()
		if sent := scored.commands[before:]; len(sent) > 0 {
			commandedAt[clock.Now()] = sent
		}
	}

	assert.Equal(t, []played{
		{set: schedule.Default, at: finaleStarts.Add(-24*time.Hour - time.Second)},
		{set: "rush", at: finaleStarts},
		{set: schedule.Default, at: seasonEnds},
	}, modules.played, "the finale's set plays from 21:00 to 23:00, and nothing else is told")

	const book = moments.BookID("season-0")
	assert.Equal(t, map[time.Time][]moments.Command{
		finaleStarts.Add(-24 * time.Hour): {
			{Action: moments.CloseRound, Book: book, Round: "season-0/day-23", At: finaleStarts.Add(-24 * time.Hour)},
			{Action: moments.OpenRound, Book: book, Round: "season-0/day-24", Times: 1, At: finaleStarts.Add(-24 * time.Hour), Ranking: moments.RankedByHeldAndGain},
		},
		finaleStarts: {
			{Action: moments.CloseRound, Book: book, Round: "season-0/day-24", At: finaleStarts},
			{Action: moments.OpenRound, Book: book, Round: "season-0/finale", Times: 3, At: finaleStarts, Ranking: moments.RankedByGain},
		},
		seasonEnds: {
			{Action: moments.CloseRound, Book: book, Round: "season-0/finale", At: seasonEnds},
			{Action: moments.CloseBook, Book: book, Tiebreak: "season-0/finale", At: seasonEnds},
		},
	}, commandedAt, "the finale is a round × 3 from 21:00, closed at 23:00 before the book it breaks a tie of")

	assert.Empty(t, planet.commands, "the Final Battle is no arena battle: the map stays")

	finale := &gamemasterv1.BattleEnded{
		Season: 0, Number: 1, Times: 3, EndsSeason: true, RoundId: "season-0/finale", EndedAt: timestamppb.New(seasonEnds),
	}
	want := []proto.Message{
		&gamemasterv1.DayEnded{Season: 0, Day: 23, RoundId: "season-0/day-23", EndedAt: timestamppb.New(finaleStarts.Add(-24 * time.Hour))},
		&gamemasterv1.DayEnded{Season: 0, Day: 24, RoundId: "season-0/day-24", EndedAt: timestamppb.New(finaleStarts)},
		finale,
		&gamemasterv1.SeasonEnded{
			Season: 0, BookId: "season-0", EndedAt: timestamppb.New(seasonEnds), FinaleRoundId: "season-0/finale", FinaleBattle: finale,
		},
	}
	recorded := outbox.Recorded()[before:]
	require.Len(t, recorded, len(want))
	for i := range want {
		assert.True(t, proto.Equal(want[i], recorded[i]), "%d: %v", i, recorded[i])
	}
}
