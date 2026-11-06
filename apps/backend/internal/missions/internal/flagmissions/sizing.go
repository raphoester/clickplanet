package flagmissions

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
)

const (
	goalStep    = 10
	mostAllowed = 100
)

var ErrInvalidSizing = errors.New("these numbers cannot size a flag mission")

type SizingSnapshot struct {
	GoalPercent  uint32
	LeastGoal    uint32
	FewestPoints uint32
	MostPoints   uint32
}

// The numbers a flag mission is drawn with, as the operator set them: its goal is GoalPercent of all the flag did the
// day before, at least LeastGoal, and it is worth MostPoints to a flag with no tile, FewestPoints to the biggest.
type Sizing struct {
	goalPercent  uint32
	leastGoal    uint32
	fewestPoints uint32
	mostPoints   uint32
}

// Every number that cannot play is named, by its name in missions.v1.FlagMissionRules.
func SizingOf(snapshot SizingSnapshot) (Sizing, error) {
	var flaws []error
	if snapshot.GoalPercent < 1 || snapshot.GoalPercent > mostAllowed {
		flaws = append(flaws, fmt.Errorf("goalPercent is %d, and must be 1 to 100", snapshot.GoalPercent))
	}
	if snapshot.LeastGoal < goalStep {
		flaws = append(flaws, fmt.Errorf("leastGoal is %d, and must be %d or more", snapshot.LeastGoal, goalStep))
	}
	if snapshot.FewestPoints < 1 {
		flaws = append(flaws, fmt.Errorf("fewestPoints is %d, and must be 1 or more", snapshot.FewestPoints))
	}
	if snapshot.MostPoints < snapshot.FewestPoints || snapshot.MostPoints > mostAllowed {
		flaws = append(flaws, fmt.Errorf("mostPoints is %d, and must be fewestPoints (%d) to 100", snapshot.MostPoints,
			snapshot.FewestPoints))
	}
	if len(flaws) > 0 {
		return Sizing{}, fmt.Errorf("%w: %w", ErrInvalidSizing, errors.Join(flaws...))
	}
	return Sizing{
		goalPercent: snapshot.GoalPercent, leastGoal: snapshot.LeastGoal, fewestPoints: snapshot.FewestPoints,
		mostPoints: snapshot.MostPoints,
	}, nil
}

func (s Sizing) Snapshot() SizingSnapshot {
	return SizingSnapshot{
		GoalPercent: s.goalPercent, LeastGoal: s.leastGoal, FewestPoints: s.fewestPoints, MostPoints: s.mostPoints,
	}
}

// Where the operator's numbers are kept. A draw reads them, so a save plays from the next draw on.
type Sizings interface {
	Sizing(ctx context.Context) (Sizing, error)
}

// A tile taken and a shield broken are a click each, so both kinds ask a part of all the flag did.
func (s Sizing) goalOf(yesterday Day) uint32 {
	goal := max(uint64(s.leastGoal), (yesterday.takes+yesterday.strikes)*uint64(s.goalPercent)/100)
	goal -= goal % goalStep
	return uint32(min(goal, uint64(^uint32(0)))) //nolint:gosec // capped just above.
}

// mostPoints for a flag with no tiles, fewestPoints for the biggest, and in proportion between, rounded.
func (s Sizing) pointsOf(flag string, world missions.World) uint32 {
	var held, biggest uint64
	for _, territory := range world.Territories {
		biggest = max(biggest, uint64(territory.Tiles))
		if territory.Flag == flag {
			held = uint64(territory.Tiles)
		}
	}
	if biggest == 0 {
		return s.mostPoints
	}
	spread := uint64(s.mostPoints - s.fewestPoints)
	return s.mostPoints - uint32((spread*held*2+biggest)/(2*biggest)) //nolint:gosec // at most mostPoints - fewestPoints.
}
