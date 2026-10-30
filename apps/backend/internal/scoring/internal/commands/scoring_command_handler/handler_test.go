package scoring_command_handler_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
)

func TestARoundOfABookNotOpenIsRefusedAndDropped(t *testing.T) {
	stack := newStack()

	require.NoError(t, stack.handler.Handle(t.Context(), openRound("season-0/day-1", "season-0", 1, utc(8, 0))))

	_, found, err := stack.rounds.Find(t.Context(), "season-0/day-1")
	require.NoError(t, err)
	assert.False(t, found)
}

func TestARoundOfABookClosedIsRefusedAndDropped(t *testing.T) {
	stack := newStack()
	require.NoError(t, stack.handler.Handle(t.Context(), openBook("season-0", utc(8, 0))))
	require.NoError(t, stack.handler.Handle(t.Context(), closeBook("season-0", "", utc(31, 23))))

	require.NoError(t, stack.handler.Handle(t.Context(), openRound(battle, "season-0", 3, utc(31, 23))))

	_, found, err := stack.rounds.Find(t.Context(), battle)
	require.NoError(t, err)
	assert.False(t, found)
}

func TestACommandThatCanNeverApplyIsDropped(t *testing.T) {
	stack := newStack()
	require.NoError(t, stack.handler.Handle(t.Context(), openBook("season-0", utc(8, 0))))

	for _, command := range []*scoringv1.ScoringCommand{
		{},
		openBook("", utc(8, 0)),
		{Command: &scoringv1.ScoringCommand_OpenBook{OpenBook: &scoringv1.OpenBook{BookId: "season-1"}}},
		openRound("season-0/day-1", "season-0", 0, utc(8, 0)),
		openRound("season-0/day-1", "season-0", 11, utc(8, 0)),
		closeRound("", utc(8, 21)),
		closeBook("season-0", string(make([]byte, 65)), utc(31, 23)),
	} {
		require.NoError(t, stack.handler.Handle(t.Context(), command), "%v", command)
	}
	_, found, err := stack.books.Find(t.Context(), "season-1")
	require.NoError(t, err)
	assert.False(t, found)
	_, found, err = stack.rounds.Find(t.Context(), "season-0/day-1")
	require.NoError(t, err)
	assert.False(t, found)
}

func TestACloseOfARoundNeverOpenedScoresNothing(t *testing.T) {
	stack := newStack()
	stack.territory.held = map[books.Country]uint32{"fr": 1}

	require.NoError(t, stack.handler.Handle(t.Context(), closeRound("season-0/day-1", utc(8, 21))))
	require.NoError(t, stack.handler.Handle(t.Context(), closeBook("season-0", "", utc(31, 23))))

	assert.Empty(t, stack.outbox.Recorded())
}

func TestAFailureIsDeliveredAgainAndScoresTheRoundThen(t *testing.T) {
	stack := newStack()
	require.NoError(t, stack.handler.Handle(t.Context(), openBook("season-0", utc(8, 0))))
	require.NoError(t, stack.handler.Handle(t.Context(), openRound("season-0/day-1", "season-0", 1, utc(8, 0))))
	down := errors.New("the planet module is down")
	stack.territory.err = down

	require.ErrorIs(t, stack.handler.Handle(t.Context(), closeRound("season-0/day-1", utc(8, 21))), down)
	assert.Empty(t, stack.outbox.Recorded())

	stack.territory.err, stack.territory.held = nil, map[books.Country]uint32{"fr": 1}
	require.NoError(t, stack.handler.Handle(t.Context(), closeRound("season-0/day-1", utc(8, 21))))
	assert.Len(t, stack.outbox.Recorded(), 1)
}

func TestABookClosedCountsTheGrantsItHolds(t *testing.T) {
	stack := newStack()
	require.NoError(t, stack.handler.Handle(t.Context(), openBook("season-0", utc(8, 0))))
	require.NoError(t, stack.handler.Handle(t.Context(), openRound("season-0/day-1", "season-0", 1, utc(8, 0))))
	stack.territory.held = map[books.Country]uint32{"fr": 1}
	require.NoError(t, stack.handler.Handle(t.Context(), closeRound("season-0/day-1", utc(8, 21))))
	for _, grant := range []books.Grant{
		{ID: "before", Country: "fr", Points: 3, At: utc(7, 0)},
		{ID: "held", Country: "de", Points: 2, At: utc(9, 0)},
	} {
		_, err := stack.grants.Keep(t.Context(), grant)
		require.NoError(t, err)
	}

	require.NoError(t, stack.handler.Handle(t.Context(), closeBook("season-0", "", utc(31, 23))))

	recorded := stack.outbox.Recorded()
	require.Len(t, recorded, 2)
	assert.True(t, proto.Equal(&scoringv1.BookClosed{BookId: "season-0", EndedAt: timestamppb.New(utc(31, 23)), Table: []*scoringv1.Place{
		{Country: "fr", Rank: 1, Points: 25, RoundsWon: 1},
		{Country: "de", Rank: 2, Points: 2},
	}}, recorded[1]), recorded[1])
}

func TestABookOpenedLaterKeepsItsFirstStart(t *testing.T) {
	stack := newStack()

	require.NoError(t, stack.handler.Handle(t.Context(), &scoringv1.ScoringCommand{Command: &scoringv1.ScoringCommand_OpenBook{
		OpenBook: &scoringv1.OpenBook{BookId: "season-0", StartsAt: timestamppb.New(utc(8, 0))},
	}}))
	require.NoError(t, stack.handler.Handle(t.Context(), openBook("season-0", utc(9, 0))))

	book, _, err := stack.books.Find(t.Context(), "season-0")
	require.NoError(t, err)
	assert.Equal(t, books.NewBook("season-0", utc(8, 0)), book)
}
