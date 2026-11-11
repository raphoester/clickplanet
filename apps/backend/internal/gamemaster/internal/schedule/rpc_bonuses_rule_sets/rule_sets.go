package rpc_bonuses_rule_sets

import (
	"context"
	"fmt"
	"slices"
	"time"

	"connectrpc.com/connect"

	bonusesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/bonuses/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule"
)

type Bonuses interface {
	GetRuleSetNames(
		ctx context.Context,
		req *connect.Request[bonusesv1.GetRuleSetNamesRequest],
	) (*connect.Response[bonusesv1.GetRuleSetNamesResponse], error)
}

const askTimeout = 2 * time.Second

func New(bonuses Bonuses) *RuleSets {
	return &RuleSets{bonuses: bonuses}
}

// RuleSets asks the offers (the boxes and the quizzes) which sets they keep: an entry plays its set in every module.
type RuleSets struct {
	bonuses Bonuses
}

var _ schedule.Keeper = (*RuleSets)(nil)

func (r *RuleSets) Keeps(ctx context.Context, set schedule.SetName) error {
	ctx, cancel := context.WithTimeout(ctx, askTimeout)
	defer cancel()

	res, err := r.bonuses.GetRuleSetNames(ctx, connect.NewRequest(&bonusesv1.GetRuleSetNamesRequest{}))
	if err != nil {
		return fmt.Errorf("failed to ask the bonuses module which rule sets it keeps: %w", err)
	}
	if !slices.Contains(res.Msg.GetNames(), string(set)) {
		return fmt.Errorf("the bonuses module %w: %q", schedule.ErrUnknownSet, set)
	}
	return nil
}
