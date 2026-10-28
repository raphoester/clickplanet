package redis_map_commands

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
)

type Sender interface {
	Send(ctx context.Context, commands ...proto.Message) error
}

const sendTimeout = 2 * time.Second

func New(commands Sender) Commands {
	return Commands{commands: commands}
}

// Commands asks planet, in its words, to reset the map as a season starts: the reset is named by the season's book.
type Commands struct {
	commands Sender
}

func (c Commands) Reset(ctx context.Context, starts season.Window) error {
	ctx, cancel := context.WithTimeout(ctx, sendTimeout)
	defer cancel()

	id := fmt.Sprintf("season-%d", starts.Number)
	if err := c.commands.Send(ctx, &planetv1.ResetMap{ResetId: id, DecidedAt: timestamppb.New(starts.StartsAt)}); err != nil {
		return fmt.Errorf("failed to command the reset %q: %w", id, err)
	}
	return nil
}
