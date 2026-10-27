package rpc_scoring_rule_sets

import (
	"context"
	"fmt"
	"slices"
	"time"

	"connectrpc.com/connect"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule"
)

type Scoring interface {
	GetRuleSetNames(
		ctx context.Context,
		req *connect.Request[scoringv1.GetRuleSetNamesRequest],
	) (*connect.Response[scoringv1.GetRuleSetNamesResponse], error)
}

const askTimeout = 2 * time.Second

func New(scoring Scoring) *RuleSets {
	return &RuleSets{scoring: scoring}
}

type RuleSets struct {
	scoring Scoring
}

var _ schedule.Keeper = (*RuleSets)(nil)

func (r *RuleSets) Keeps(ctx context.Context, set schedule.SetName) error {
	ctx, cancel := context.WithTimeout(ctx, askTimeout)
	defer cancel()

	res, err := r.scoring.GetRuleSetNames(ctx, connect.NewRequest(&scoringv1.GetRuleSetNamesRequest{}))
	if err != nil {
		return fmt.Errorf("failed to ask the scoring module which rule sets it keeps: %w", err)
	}
	if !slices.Contains(res.Msg.GetNames(), string(set)) {
		return fmt.Errorf("scoring %w: %q", schedule.ErrUnknownSet, set)
	}
	return nil
}
