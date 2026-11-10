package redis_arena_commands

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
)

type Sender interface {
	Send(ctx context.Context, commands ...proto.Message) error
}

const sendTimeout = 2 * time.Second

func New(commands Sender) Commands {
	return Commands{commands: commands}
}

// Commands asks planet, in its words, on its one arena stream: an open and a close are handled in the order sent.
type Commands struct {
	commands Sender
}

func (c Commands) Send(ctx context.Context, command moments.ArenaCommand) error {
	asked, err := commandOf(command)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, sendTimeout)
	defer cancel()

	if err := c.commands.Send(ctx, asked); err != nil {
		return fmt.Errorf("failed to %s %s: %w", command.Action, command.Arena, err)
	}
	return nil
}

func commandOf(command moments.ArenaCommand) (*planetv1.ArenaCommand, error) {
	switch command.Action {
	case moments.OpenArena:
		return &planetv1.ArenaCommand{Command: &planetv1.ArenaCommand_OpenArena{OpenArena: &planetv1.OpenArena{
			ArenaId: string(command.Arena), EndsAt: timestamppb.New(command.EndsAt), RuleSet: string(command.Set),
		}}}, nil
	case moments.CloseArena:
		return &planetv1.ArenaCommand{Command: &planetv1.ArenaCommand_CloseArena{CloseArena: &planetv1.CloseArena{
			ArenaId: string(command.Arena),
		}}}, nil
	}
	return nil, fmt.Errorf("planet has no arena command to %s", command.Action)
}
