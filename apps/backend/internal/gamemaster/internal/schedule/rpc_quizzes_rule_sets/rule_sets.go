package rpc_quizzes_rule_sets

import (
	"context"
	"fmt"
	"slices"
	"time"

	"connectrpc.com/connect"

	quizzesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/quizzes/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule"
)

type Quizzes interface {
	GetRuleSetNames(
		ctx context.Context,
		req *connect.Request[quizzesv1.GetRuleSetNamesRequest],
	) (*connect.Response[quizzesv1.GetRuleSetNamesResponse], error)
}

const askTimeout = 2 * time.Second

func New(quizzes Quizzes) *RuleSets {
	return &RuleSets{quizzes: quizzes}
}

// RuleSets asks the quizzes which sets they keep: an entry plays its set in every module.
type RuleSets struct {
	quizzes Quizzes
}

var _ schedule.Keeper = (*RuleSets)(nil)

func (r *RuleSets) Keeps(ctx context.Context, set schedule.SetName) error {
	ctx, cancel := context.WithTimeout(ctx, askTimeout)
	defer cancel()

	res, err := r.quizzes.GetRuleSetNames(ctx, connect.NewRequest(&quizzesv1.GetRuleSetNamesRequest{}))
	if err != nil {
		return fmt.Errorf("failed to ask the quizzes module which rule sets it keeps: %w", err)
	}
	if !slices.Contains(res.Msg.GetNames(), string(set)) {
		return fmt.Errorf("the quizzes module %w: %q", schedule.ErrUnknownSet, set)
	}
	return nil
}
