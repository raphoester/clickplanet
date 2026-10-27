package redis_season_commands

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
)

type Sender interface {
	Send(ctx context.Context, commands ...proto.Message) error
}

const sendTimeout = 2 * time.Second

func New(commands Sender) Commands {
	return Commands{commands: commands}
}

// Commands tells scoring, in its words, a season's window.
type Commands struct {
	commands Sender
}

func (c Commands) Play(ctx context.Context, played season.Season, decidedAt time.Time) error {
	ctx, cancel := context.WithTimeout(ctx, sendTimeout)
	defer cancel()

	snapshot := played.TakeSnapshot()
	if err := c.commands.Send(ctx, &scoringv1.PlaySeason{
		Number: uint32(snapshot.Number), StartsAt: timestamppb.New(snapshot.StartsAt), EndsAt: timestamppb.New(snapshot.EndsAt),
		DecidedAt: timestamppb.New(decidedAt),
	}); err != nil {
		return fmt.Errorf("failed to command season %d: %w", snapshot.Number, err)
	}
	return nil
}
