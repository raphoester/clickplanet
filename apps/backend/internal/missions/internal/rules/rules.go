package rules

import (
	"context"
	"errors"
	"fmt"

	missionsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/missions/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/flagmissions"
)

var ErrInvalidRules = errors.New("these rules cannot play")

// The numbers missions draws with, one document the operator edits.
type Rules struct {
	flagMissions flagmissions.Sizing
}

// The snapshot is the wire's message: postgres keeps it as protojson, and the operator's calls carry it.
func RulesOf(snapshot *missionsv1.Rules) (Rules, error) {
	flags := snapshot.GetFlagMissions()
	sizing, err := flagmissions.SizingOf(flagmissions.SizingSnapshot{
		GoalPercent: flags.GetGoalPercent(), LeastGoal: flags.GetLeastGoal(), FewestPoints: flags.GetFewestPoints(),
		MostPoints: flags.GetMostPoints(),
	})
	if err != nil {
		return Rules{}, fmt.Errorf("%w: flagMissions: %w", ErrInvalidRules, err)
	}
	return Rules{flagMissions: sizing}, nil
}

func (r Rules) Snapshot() *missionsv1.Rules {
	flags := r.flagMissions.Snapshot()
	return &missionsv1.Rules{FlagMissions: &missionsv1.FlagMissionRules{
		GoalPercent: flags.GoalPercent, LeastGoal: flags.LeastGoal, FewestPoints: flags.FewestPoints,
		MostPoints: flags.MostPoints,
	}}
}

func (r Rules) FlagMissions() flagmissions.Sizing {
	return r.flagMissions
}

type Store interface {
	Rules(ctx context.Context) (Rules, error)
	Save(ctx context.Context, rules Rules) error
}
