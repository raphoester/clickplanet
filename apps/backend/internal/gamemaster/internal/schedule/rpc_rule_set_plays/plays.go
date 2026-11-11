package rpc_rule_set_plays

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	antibotv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule"
)

type Planet interface {
	PlayRuleSet(ctx context.Context, req *connect.Request[planetv1.PlayRuleSetRequest],
	) (*connect.Response[planetv1.PlayRuleSetResponse], error)
}

type Antibot interface {
	PlayRuleSet(ctx context.Context, req *connect.Request[antibotv1.PlayRuleSetRequest],
	) (*connect.Response[antibotv1.PlayRuleSetResponse], error)
}

// The antibot takes a set with its running bans restated.
const playTimeout = time.Minute

func New(planet Planet, antibot Antibot) Plays {
	return Plays{planet: planet, antibot: antibot}
}

// Plays tells planet and the antibot, each in its own words, the set the schedule plays.
type Plays struct {
	planet  Planet
	antibot Antibot
}

func (p Plays) Play(ctx context.Context, turn schedule.Turn, decidedAt time.Time) error {
	if err := p.PlayAntibot(ctx, turn, decidedAt); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, playTimeout)
	defer cancel()

	set := string(turn.TakeSnapshot().Set)
	if _, err := p.planet.PlayRuleSet(ctx, connect.NewRequest(&planetv1.PlayRuleSetRequest{
		RuleSet: set, DecidedAt: timestamppb.New(decidedAt),
	})); err != nil {
		return fmt.Errorf("planet did not play the rule set %q: %w", set, err)
	}
	return nil
}

func (p Plays) PlayAntibot(ctx context.Context, turn schedule.Turn, decidedAt time.Time) error {
	ctx, cancel := context.WithTimeout(ctx, playTimeout)
	defer cancel()

	set := string(turn.TakeSnapshot().Set)
	if _, err := p.antibot.PlayRuleSet(ctx, connect.NewRequest(&antibotv1.PlayRuleSetRequest{
		RuleSet: set, DecidedAt: timestamppb.New(decidedAt),
	})); err != nil {
		return fmt.Errorf("the antibot did not play the rule set %q: %w", set, err)
	}
	return nil
}
