package moments

import (
	"context"
	"fmt"
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

type Opened interface {
	Opened(ctx context.Context) ([]RoundID, error)
}

func NewPlanner(seasons Seasons, entries Entries, opened Opened, cutoff season.Cutoff) Planner {
	return Planner{seasons: seasons, entries: entries, opened: opened, cutoff: cutoff}
}

// Planner reads the seasons and the entries that may be a battle of one, and plans their books.
type Planner struct {
	seasons Seasons
	entries Entries
	opened  Opened
	cutoff  season.Cutoff
}

func (p Planner) Plan(ctx context.Context) (Plan, error) {
	calendar, err := p.seasons.Calendar(ctx)
	if err != nil {
		return Plan{}, fmt.Errorf("could not read the seasons: %w", err)
	}
	entries, err := p.entries.Unended(ctx, season.Origin)
	if err != nil {
		return Plan{}, fmt.Errorf("could not read the schedule: %w", err)
	}
	opened, err := p.opened.Opened(ctx)
	if err != nil {
		return Plan{}, fmt.Errorf("could not read the rounds opened: %w", err)
	}
	snapshots := make([]schedule.EntrySnapshot, 0, len(entries))
	for _, entry := range entries {
		snapshots = append(snapshots, entry.TakeSnapshot())
	}
	return NewPlan(calendar, p.cutoff, opened, snapshots...), nil
}
