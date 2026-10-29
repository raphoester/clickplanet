package scoring_command_handler_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books/inmemory_book_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books/inmemory_grant_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books/inmemory_round_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books/usecases/close_book_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books/usecases/close_round_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books/usecases/open_book_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books/usecases/open_round_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/commands/log_command"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/commands/scoring_command_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/subscribers/log_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

type stubTerritory struct {
	held map[books.Country]uint32
	err  error
}

func (s *stubTerritory) Snapshot(context.Context) (books.Snapshot, error) {
	tiles := uint32(0)
	for _, held := range s.held {
		tiles += held
	}
	return books.Snapshot{Tiles: tiles, Held: s.held}, s.err
}

type stack struct {
	handler   scoring_command_handler.Handler[*scoringv1.ScoringCommand]
	books     *inmemory_book_store.Store
	rounds    *inmemory_round_store.Store
	grants    *inmemory_grant_store.Store
	outbox    *cpatomic.FakeOutbox
	territory *stubTerritory
}

func newStack() stack {
	shelf, rounds, grants := inmemory_book_store.New(), inmemory_round_store.New(), inmemory_grant_store.New()
	outbox, territory := &cpatomic.FakeOutbox{}, &stubTerritory{}
	logger := slog.New(slog.DiscardHandler)
	tx := cpatomic.FakeTransactioner{}
	return stack{
		handler: scoring_command_handler.New(log_subscriber.New(scoring_command_handler.NewDispatcher(scoring_command_handler.UseCases{
			OpenBook:   log_command.New("open_book", open_book_usecase.New(shelf), logger),
			OpenRound:  log_command.New("open_round", open_round_usecase.New(shelf, rounds), logger),
			CloseRound: log_command.New("close_round", close_round_usecase.New(tx, territory, rounds, outbox), logger),
			CloseBook:  log_command.New("close_book", close_book_usecase.New(tx, shelf, rounds, grants, outbox), logger),
		}), logger)),
		books: shelf, rounds: rounds, grants: grants, outbox: outbox, territory: territory,
	}
}

func utc(day, hour int) time.Time {
	return time.Date(2026, 10, day, hour, 0, 0, 0, time.UTC)
}

const battle = "0192f5a0-7c00-7000-8000-000000000031"

func openBook(id string, at time.Time) *scoringv1.ScoringCommand {
	return &scoringv1.ScoringCommand{Command: &scoringv1.ScoringCommand_OpenBook{OpenBook: &scoringv1.OpenBook{
		BookId: id, StartsAt: timestamppb.New(at),
	}}}
}

func openRound(id, book string, times uint32, at time.Time) *scoringv1.ScoringCommand {
	return &scoringv1.ScoringCommand{Command: &scoringv1.ScoringCommand_OpenRound{OpenRound: &scoringv1.OpenRound{
		RoundId: id, BookId: book, Times: times, StartsAt: timestamppb.New(at),
	}}}
}

func closeRound(id string, at time.Time) *scoringv1.ScoringCommand {
	return &scoringv1.ScoringCommand{Command: &scoringv1.ScoringCommand_CloseRound{CloseRound: &scoringv1.CloseRound{
		RoundId: id, EndsAt: timestamppb.New(at),
	}}}
}

func closeBook(id, tiebreak string, at time.Time) *scoringv1.ScoringCommand {
	return &scoringv1.ScoringCommand{Command: &scoringv1.ScoringCommand_CloseBook{CloseBook: &scoringv1.CloseBook{
		BookId: id, EndsAt: timestamppb.New(at), TiebreakRoundId: tiebreak,
	}}}
}

// Season 0's end, Sat 31 Oct 2026: day 24 closes at 21:00, the Final Battle ×3 runs from 21:00 to 23:00 and is the
// season's tiebreak, then the book closes, all on the one stream in the gamemaster's order.
func TestTheLastDayThenTheFinalBattleThenTheSeasonAreScoredOnceInTheirOrder(t *testing.T) {
	stack := newStack()
	handle := func(commands ...*scoringv1.ScoringCommand) {
		for _, command := range commands {
			require.NoError(t, stack.handler.Handle(t.Context(), command))
		}
	}

	handle(openBook("season-0", utc(8, 0)), openRound("season-0/day-24", "season-0", 1, utc(30, 21)))
	stack.territory.held = map[books.Country]uint32{"fr": 3, "de": 1}
	handle(closeRound("season-0/day-24", utc(31, 21)), openRound(battle, "season-0", 3, utc(31, 21)))
	stack.territory.held = map[books.Country]uint32{"de": 4, "fr": 1}
	handle(closeRound(battle, utc(31, 23)), closeBook("season-0", battle, utc(31, 23)))

	want := []proto.Message{
		&scoringv1.RoundScored{RoundId: "season-0/day-24", BookId: "season-0", EndedAt: timestamppb.New(utc(31, 21)), Results: []*scoringv1.RoundResult{
			{Country: "fr", Rank: 1, Points: 25, Share: 0.75}, {Country: "de", Rank: 2, Points: 18, Share: 0.25},
		}},
		&scoringv1.RoundScored{RoundId: battle, BookId: "season-0", EndedAt: timestamppb.New(utc(31, 23)), Results: []*scoringv1.RoundResult{
			{Country: "de", Rank: 1, Points: 75, Share: 0.8}, {Country: "fr", Rank: 2, Points: 54, Share: 0.2},
		}},
		&scoringv1.BookClosed{BookId: "season-0", EndedAt: timestamppb.New(utc(31, 23)), Table: []*scoringv1.Place{
			{Country: "de", Rank: 1, Points: 93, RoundsWon: 1, TiebreakPoints: 75},
			{Country: "fr", Rank: 2, Points: 79, RoundsWon: 1, TiebreakPoints: 54},
		}},
	}
	assertRecorded(t, want, stack.outbox.Recorded())

	book, _, err := stack.books.Find(t.Context(), "season-0")
	require.NoError(t, err)
	scored, err := stack.rounds.ScoredIn(t.Context(), "season-0")
	require.NoError(t, err)

	stack.territory.held = map[books.Country]uint32{"it": 9}
	handle(
		openBook("season-0", utc(9, 0)),
		openRound("season-0/day-24", "season-0", 1, utc(30, 21)),
		closeRound("season-0/day-24", utc(31, 21)),
		openRound(battle, "season-0", 3, utc(31, 21)),
		closeRound(battle, utc(31, 23)),
		closeBook("season-0", battle, utc(31, 23)),
	)

	assertRecorded(t, want, stack.outbox.Recorded())
	again, _, err := stack.books.Find(t.Context(), "season-0")
	require.NoError(t, err)
	assert.Equal(t, book, again)
	scoredAgain, err := stack.rounds.ScoredIn(t.Context(), "season-0")
	require.NoError(t, err)
	assert.Equal(t, scored, scoredAgain, "a second delivery of each command changes nothing")
}

func assertRecorded(t *testing.T, want, recorded []proto.Message) {
	t.Helper()
	require.Len(t, recorded, len(want))
	for i := range want {
		assert.True(t, proto.Equal(want[i], recorded[i]), "event %d: %v", i, recorded[i])
	}
}
