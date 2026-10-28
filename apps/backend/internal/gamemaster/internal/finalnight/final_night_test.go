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
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/inmemory_moment_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/outbox_moment_events"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/usecases/hear_book_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/usecases/hear_round_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/usecases/play_scoring_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule/inmemory_entry_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule/usecases/play_schedule_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season/inmemory_season_store"
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

func (p *planetAndAntibot) Play(_ context.Context, turn schedule.Turn, decidedAt time.Time) error {
	p.played = append(p.played, played{set: turn.TakeSnapshot().Set, at: decidedAt})
	return nil
}

// scoring scores each round and closes each book as it is commanded, and tells the gamemaster at once.
type scoring struct {
	t        *testing.T
	commands []moments.Command
	rounds   *hear_round_usecase.Hearing
	books    *hear_book_usecase.Hearing
}

func (s *scoring) Send(ctx context.Context, command moments.Command) error {
	s.commands = append(s.commands, command)
	switch command.Action {
	case moments.CloseRound:
		heard, err := s.rounds.Execute(ctx, moments.ScoredRound{Round: command.Round, Book: command.Book, EndedAt: command.At})
		require.NoError(s.t, err)
		require.Equal(s.t, hear_round_usecase.Told, heard, "%q", command.Round)
	case moments.CloseBook:
		heard, err := s.books.Execute(ctx, moments.ClosedBook{Book: command.Book, EndedAt: command.At})
		require.NoError(s.t, err)
		require.Equal(s.t, hear_book_usecase.Told, heard)
	case moments.OpenBook, moments.OpenRound:
	}
	return nil
}

func TestSeasonZerosLastDayAndItsFinalePlaySecondBySecond(t *testing.T) {
	clock := cptime.NewFixedClock(finaleStarts.Add(-24*time.Hour - time.Second))
	seasons, entries, kept := inmemory_season_store.New(), inmemory_entry_store.New(), inmemory_moment_store.New()
	require.NoError(t, seasons.Save(t.Context(), seasonZero.Restore()))
	planner := moments.NewPlanner(seasons, entries, kept, season.Cutoff(21*time.Hour))
	outbox := &cpatomic.FakeOutbox{}
	told := outbox_moment_events.New(outbox)

	modules := &planetAndAntibot{}
	rules := play_schedule_usecase.New(entries, seasons, modules, clock)
	scored := &scoring{
		t:      t,
		rounds: hear_round_usecase.New(planner, kept, told, cpatomic.FakeTransactioner{}),
		books:  hear_book_usecase.New(planner, kept, told, cpatomic.FakeTransactioner{}),
	}
	points := play_scoring_usecase.New(planner, kept, scored, clock)

	tick := func() {
		_, _, err := rules.Execute(t.Context())
		require.NoError(t, err)
		_, err = points.Execute(t.Context())
		require.NoError(t, err)
	}
	tick()
	scored.commands, outbox = nil, &cpatomic.FakeOutbox{}
	told = outbox_moment_events.New(outbox)
	scored.rounds = hear_round_usecase.New(planner, kept, told, cpatomic.FakeTransactioner{})
	scored.books = hear_book_usecase.New(planner, kept, told, cpatomic.FakeTransactioner{})

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
			{Action: moments.OpenRound, Book: book, Round: "season-0/day-24", Times: 1, At: finaleStarts.Add(-24 * time.Hour)},
		},
		finaleStarts: {
			{Action: moments.CloseRound, Book: book, Round: "season-0/day-24", At: finaleStarts},
			{Action: moments.OpenRound, Book: book, Round: "season-0/finale", Times: 3, At: finaleStarts},
		},
		seasonEnds: {
			{Action: moments.CloseRound, Book: book, Round: "season-0/finale", At: seasonEnds},
			{Action: moments.CloseBook, Book: book, Tiebreak: "season-0/finale", At: seasonEnds},
		},
	}, commandedAt, "the finale is a round × 3 from 21:00, closed at 23:00 before the book it breaks a tie of")

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
	recorded := outbox.Recorded()
	require.Len(t, recorded, len(want))
	for i := range want {
		assert.True(t, proto.Equal(want[i], recorded[i]), "%d: %v", i, recorded[i])
	}
}
