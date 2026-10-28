package redis_scoring_commands_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/redis_scoring_commands"
)

type sent struct {
	commands []proto.Message
	err      error
}

func (s *sent) Send(_ context.Context, commands ...proto.Message) error {
	s.commands = append(s.commands, commands...)
	return s.err
}

var finale = time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC)

const entry = "0192f5a0-7c00-7000-8000-000000000031"

func TestEachCommandIsOneScoringCommandInScoringsWords(t *testing.T) {
	for _, each := range []struct {
		command moments.Command
		wire    *scoringv1.ScoringCommand
	}{
		{
			moments.Command{Action: moments.OpenBook, Book: "season-0", At: finale},
			&scoringv1.ScoringCommand{Command: &scoringv1.ScoringCommand_OpenBook{OpenBook: &scoringv1.OpenBook{
				BookId: "season-0", StartsAt: timestamppb.New(finale),
			}}},
		},
		{
			moments.Command{Action: moments.OpenRound, Book: "season-0", Round: entry, Times: 3, At: finale},
			&scoringv1.ScoringCommand{Command: &scoringv1.ScoringCommand_OpenRound{OpenRound: &scoringv1.OpenRound{
				RoundId: entry, BookId: "season-0", Times: 3, StartsAt: timestamppb.New(finale),
			}}},
		},
		{
			moments.Command{Action: moments.CloseRound, Book: "season-0", Round: "season-0/day-24", At: finale},
			&scoringv1.ScoringCommand{Command: &scoringv1.ScoringCommand_CloseRound{CloseRound: &scoringv1.CloseRound{
				RoundId: "season-0/day-24", EndsAt: timestamppb.New(finale),
			}}},
		},
		{
			moments.Command{Action: moments.CloseBook, Book: "season-0", Tiebreak: entry, At: finale},
			&scoringv1.ScoringCommand{Command: &scoringv1.ScoringCommand_CloseBook{CloseBook: &scoringv1.CloseBook{
				BookId: "season-0", EndsAt: timestamppb.New(finale), TiebreakRoundId: entry,
			}}},
		},
	} {
		commands := &sent{}

		require.NoError(t, redis_scoring_commands.New(commands).Send(t.Context(), each.command))

		require.Len(t, commands.commands, 1)
		assert.True(t, proto.Equal(each.wire, commands.commands[0]), "%v", commands.commands[0])
	}
}

func TestACommandRedisDidNotTakeIsAnError(t *testing.T) {
	down := errors.New("redis is down")

	err := redis_scoring_commands.New(&sent{err: down}).Send(t.Context(),
		moments.Command{Action: moments.CloseRound, Book: "season-0", Round: "season-0/day-24", At: finale})

	require.ErrorIs(t, err, down)
	assert.Contains(t, err.Error(), "close round season-0/day-24")
}

func TestAnActionScoringHasNoCommandForIsAnErrorAndSendsNothing(t *testing.T) {
	commands := &sent{}

	err := redis_scoring_commands.New(commands).Send(t.Context(), moments.Command{Book: "season-0", At: finale})

	require.Error(t, err)
	assert.Empty(t, commands.commands)
}
