//go:build testing

package flagmissions

import "context"

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
