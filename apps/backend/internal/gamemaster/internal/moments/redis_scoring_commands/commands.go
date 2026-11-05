package redis_scoring_commands

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
)

type Sender interface {
	Send(ctx context.Context, commands ...proto.Message) error
}

const sendTimeout = 2 * time.Second

func New(commands Sender) Commands {
	return Commands{commands: commands}
}

// Commands asks scoring, in its words, on its one command stream.
type Commands struct {
	commands Sender
}

func (c Commands) Send(ctx context.Context, command moments.Command) error {
	asked, err := commandOf(command)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, sendTimeout)
	defer cancel()

	if err := c.commands.Send(ctx, asked); err != nil {
		subject := string(command.Book)
		if command.Round != "" {
			subject = string(command.Round)
		}
		return fmt.Errorf("failed to %s %s: %w", command.Action, subject, err)
	}
	return nil
}

func commandOf(command moments.Command) (*scoringv1.ScoringCommand, error) {
	at := timestamppb.New(command.At)
	switch command.Action {
	case moments.OpenBook:
		return &scoringv1.ScoringCommand{Command: &scoringv1.ScoringCommand_OpenBook{OpenBook: &scoringv1.OpenBook{
			BookId: string(command.Book), StartsAt: at,
		}}}, nil
	case moments.OpenRound:
		return &scoringv1.ScoringCommand{Command: &scoringv1.ScoringCommand_OpenRound{OpenRound: &scoringv1.OpenRound{
			RoundId: string(command.Round), BookId: string(command.Book), Times: uint32(command.Times), StartsAt: at,
		}}}, nil
	case moments.CloseRound:
		return &scoringv1.ScoringCommand{Command: &scoringv1.ScoringCommand_CloseRound{CloseRound: &scoringv1.CloseRound{
			RoundId: string(command.Round), EndsAt: at,
		}}}, nil
	case moments.CloseBook:
		return &scoringv1.ScoringCommand{Command: &scoringv1.ScoringCommand_CloseBook{CloseBook: &scoringv1.CloseBook{
			BookId: string(command.Book), EndsAt: at, TiebreakRoundId: string(command.Tiebreak),
		}}}, nil
	}
	return nil, fmt.Errorf("scoring has no command to %s", command.Action)
}
