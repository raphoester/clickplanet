package rpc_boxes_rule_sets

import (
	"context"
	"fmt"
	"slices"
	"time"

	"connectrpc.com/connect"

	boxesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/boxes/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule"
)

type Boxes interface {
	GetRuleSetNames(
		ctx context.Context,
		req *connect.Request[boxesv1.GetRuleSetNamesRequest],
	) (*connect.Response[boxesv1.GetRuleSetNamesResponse], error)
}

const askTimeout = 2 * time.Second

func New(boxes Boxes) *RuleSets {
	return &RuleSets{boxes: boxes}
}

// RuleSets asks the boxes which sets they keep: an entry plays its set in every module.
type RuleSets struct {
	boxes Boxes
}

var _ schedule.Keeper = (*RuleSets)(nil)

func (r *RuleSets) Keeps(ctx context.Context, set schedule.SetName) error {
	ctx, cancel := context.WithTimeout(ctx, askTimeout)
	defer cancel()

	res, err := r.boxes.GetRuleSetNames(ctx, connect.NewRequest(&boxesv1.GetRuleSetNamesRequest{}))
	if err != nil {
		return fmt.Errorf("failed to ask the boxes module which rule sets it keeps: %w", err)
	}
	if !slices.Contains(res.Msg.GetNames(), string(set)) {
		return fmt.Errorf("the boxes module %w: %q", schedule.ErrUnknownSet, set)
	}
	return nil
}
