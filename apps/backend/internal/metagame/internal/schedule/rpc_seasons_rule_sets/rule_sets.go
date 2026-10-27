package rpc_seasons_rule_sets

import (
	"context"
	"fmt"
	"slices"
	"time"

	"connectrpc.com/connect"

	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/metagame/internal/schedule"
)

type Seasons interface {
	GetRuleSetNames(
		ctx context.Context,
		req *connect.Request[seasonsv1.GetRuleSetNamesRequest],
	) (*connect.Response[seasonsv1.GetRuleSetNamesResponse], error)
}

const askTimeout = 2 * time.Second

func New(seasons Seasons) *RuleSets {
	return &RuleSets{seasons: seasons}
}

type RuleSets struct {
	seasons Seasons
}

var _ schedule.Keeper = (*RuleSets)(nil)

func (r *RuleSets) Keeps(ctx context.Context, set schedule.SetName) error {
	ctx, cancel := context.WithTimeout(ctx, askTimeout)
	defer cancel()

	res, err := r.seasons.GetRuleSetNames(ctx, connect.NewRequest(&seasonsv1.GetRuleSetNamesRequest{}))
	if err != nil {
		return fmt.Errorf("failed to ask the seasons module which rule sets it keeps: %w", err)
	}
	if !slices.Contains(res.Msg.GetNames(), string(set)) {
		return fmt.Errorf("seasons %w: %q", schedule.ErrUnknownSet, set)
	}
	return nil
}
