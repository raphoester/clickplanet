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

// Kept is what the gamemaster keeps of the rounds it opened and scoring scored.
type Kept interface {
	Opened(ctx context.Context) ([]RoundID, error)
	Scored(ctx context.Context) ([]RoundID, error)
}

// Judged is what the gamemaster keeps of the arena battles: each battle judged, and each arena planet closed.
type Judged interface {
	Judged(ctx context.Context) ([]Judgement, error)
	Closed(ctx context.Context) ([]RoundID, error)
}

func NewPlanner(seasons Seasons, entries Entries, kept Kept, judged Judged, cutoff season.Cutoff) Planner {
	return Planner{seasons: seasons, entries: entries, kept: kept, judged: judged, cutoff: cutoff}
}

// Planner reads the seasons, the entries that may be a battle of one and the arena battles, and plans their books.
type Planner struct {
	seasons Seasons
	entries Entries
	kept    Kept
	judged  Judged
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
	opened, err := p.kept.Opened(ctx)
	if err != nil {
		return Plan{}, fmt.Errorf("could not read the rounds opened: %w", err)
	}
	known, err := p.arenas(ctx)
	if err != nil {
		return Plan{}, err
	}
	for _, entry := range changed {
		id := entry.TakeSnapshot().ID
		entries = slices.DeleteFunc(entries, func(kept schedule.Entry) bool { return kept.TakeSnapshot().ID == id })
	}
	return NewPlan(calendar, p.cutoff, opened, append(entries, changed...)...).With(known), nil
}

func (p Planner) arenas(ctx context.Context) (Arenas, error) {
	scored, err := p.kept.Scored(ctx)
	if err != nil {
		return Arenas{}, fmt.Errorf("could not read the rounds scored: %w", err)
	}
	judged, err := p.judged.Judged(ctx)
	if err != nil {
		return Arenas{}, fmt.Errorf("could not read the battles judged: %w", err)
	}
	closed, err := p.judged.Closed(ctx)
	if err != nil {
		return Arenas{}, fmt.Errorf("could not read the arenas closed: %w", err)
	}
	return NewArenas(judged, scored, closed), nil
}
