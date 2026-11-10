package moments

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
)

type Seasons interface {
	Calendar(ctx context.Context) (season.Calendar, error)
}

type Entries interface {
	Unended(ctx context.Context, at time.Time) ([]schedule.Entry, error)
}

// Rounds is what the gamemaster keeps of its commands to scoring, read in one go.
type Rounds interface {
	Kept(ctx context.Context) (Kept, error)
}

// Arenas is what the gamemaster keeps of the arena battles, read in one go.
type Arenas interface {
	ArenasKept(ctx context.Context) (ArenasKept, error)
}

func NewPlanner(seasons Seasons, entries Entries, rounds Rounds, arenas Arenas, cutoff season.Cutoff) Planner {
	return Planner{seasons: seasons, entries: entries, rounds: rounds, arenas: arenas, cutoff: cutoff}
}

// Planner reads the seasons, the entries that may be a battle of one, and what the gamemaster kept of its commands, and
// plans their books.
type Planner struct {
	seasons Seasons
	entries Entries
	rounds  Rounds
	arenas  Arenas
	cutoff  season.Cutoff
}

func (p Planner) Plan(ctx context.Context) (Plan, error) {
	return p.With(ctx)
}

func (p Planner) With(ctx context.Context, changed ...schedule.Entry) (Plan, error) {
	calendar, err := p.seasons.Calendar(ctx)
	if err != nil {
		return Plan{}, fmt.Errorf("could not read the seasons: %w", err)
	}
	entries, err := p.entries.Unended(ctx, season.Origin)
	if err != nil {
		return Plan{}, fmt.Errorf("could not read the schedule: %w", err)
	}
	kept, err := p.rounds.Kept(ctx)
	if err != nil {
		return Plan{}, fmt.Errorf("could not read what scoring answered: %w", err)
	}
	arenas, err := p.arenas.ArenasKept(ctx)
	if err != nil {
		return Plan{}, fmt.Errorf("could not read the arena battles: %w", err)
	}
	for _, entry := range changed {
		id := entry.TakeSnapshot().ID
		entries = slices.DeleteFunc(entries, func(other schedule.Entry) bool { return other.TakeSnapshot().ID == id })
	}
	return NewPlan(calendar, p.cutoff, kept.Opened, append(entries, changed...)...).With(NewKnown(kept, arenas)), nil
}
