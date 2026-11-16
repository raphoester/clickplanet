//go:build testing

package flagmissions

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/days"
)

// The numbers migration 20261106180000_rules seeds.
func TestingSizing() Sizing {
	sizing, err := SizingOf(SizingSnapshot{GoalPercent: 30, LeastGoal: 250, FewestPoints: 3, MostPoints: 10})
	if err != nil {
		panic(err)
	}
	return sizing
}

// FixedSizings answers the same numbers at each draw.
type FixedSizings struct {
	sizing Sizing
}

func NewFixedSizings(sizing Sizing) FixedSizings {
	return FixedSizings{sizing: sizing}
}

func (f FixedSizings) Sizing(context.Context) (Sizing, error) {
	return f.sizing, nil
}

// TestingStanding is a season's table with every round before the day scored.
func TestingStanding(points map[string]uint32) Standing {
	return StandingOf(StandingSnapshot{Points: points, Settled: true})
}

// FixedStandings answers the same table at each draw.
type FixedStandings struct {
	standing Standing
}

func NewFixedStandings(standing Standing) FixedStandings {
	return FixedStandings{standing: standing}
}

func (f FixedStandings) Standing(context.Context, days.Day) (Standing, error) {
	return f.standing, nil
}
