package rpc_antibot_rule_sets

import (
	"context"
	"fmt"
	"slices"
	"time"

	"connectrpc.com/connect"

	antibotv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/schedule"
)

type Antibot interface {
	GetRuleSetNames(
		ctx context.Context,
		req *connect.Request[antibotv1.GetRuleSetNamesRequest],
	) (*connect.Response[antibotv1.GetRuleSetNamesResponse], error)
}

const askTimeout = 2 * time.Second

func New(antibot Antibot) *RuleSets {
	return &RuleSets{antibot: antibot}
}

type RuleSets struct {
	antibot Antibot
}

var _ schedule.Keeper = (*RuleSets)(nil)

// An antibot that is off serves no route, and judges with no set: it lacks none.
func (r *RuleSets) Keeps(ctx context.Context, set schedule.SetName) error {
	ctx, cancel := context.WithTimeout(ctx, askTimeout)
	defer cancel()

	res, err := r.antibot.GetRuleSetNames(ctx, connect.NewRequest(&antibotv1.GetRuleSetNamesRequest{}))
	if connect.CodeOf(err) == connect.CodeUnimplemented {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to ask the antibot which rule sets it keeps: %w", err)
	}
	if !slices.Contains(res.Msg.GetNames(), string(set)) {
		return fmt.Errorf("the antibot %w: %q", schedule.ErrUnknownSet, set)
	}
	return nil
}
