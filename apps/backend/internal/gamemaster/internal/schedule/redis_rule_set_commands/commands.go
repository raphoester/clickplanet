package redis_rule_set_commands

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	antibotv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule"
)

type Sender interface {
	Send(ctx context.Context, commands ...proto.Message) error
}

const sendTimeout = 2 * time.Second

func New(commands Sender) Commands {
	return Commands{commands: commands}
}

// Commands tells planet, the antibot and scoring, each in its own words, the set the schedule plays. Scoring is told
// the entry's window too, for a round of its own.
type Commands struct {
	commands Sender
}

func (c Commands) Play(ctx context.Context, turn schedule.Turn, decidedAt time.Time) error {
	ctx, cancel := context.WithTimeout(ctx, sendTimeout)
	defer cancel()

	played := turn.TakeSnapshot()
	set, at := string(played.Set), timestamppb.New(decidedAt)
	scoring := &scoringv1.PlayRuleSet{RuleSet: set, DecidedAt: at}
	if !played.EndsAt.IsZero() {
		scoring.StartsAt, scoring.EndsAt = timestamppb.New(played.StartsAt), timestamppb.New(played.EndsAt)
	}
	if err := c.commands.Send(ctx,
		&planetv1.PlayRuleSet{RuleSet: set, DecidedAt: at},
		&antibotv1.PlayRuleSet{RuleSet: set, DecidedAt: at},
		scoring,
	); err != nil {
		return fmt.Errorf("failed to command the rule set %q: %w", set, err)
	}
	return nil
}
