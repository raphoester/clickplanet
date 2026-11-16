package flagmissions

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/days"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var ErrNoDay = errors.New("no day was drawn for the flag")

type Store interface {
	Day(ctx context.Context, day days.Day, flag string) (Day, error)
	// The first day kept for the day and the flag wins, and is the answer. A day kept with no mission takes the
	// mission of the first one kept after, and keeps its own counts.
	KeepDay(ctx context.Context, day days.Day, flag string, drawn Day) (Day, error)
	// A day never kept is ErrNoDay.
	ChangeDay(ctx context.Context, day days.Day, flag string, change func(Day) Day) error
}

type Calendar interface {
	DayBefore(ctx context.Context, day days.Day) (days.Day, error)
}

func NewBriefing(
	store Store, calendar Calendar, planet missions.Planet, standings Standings, allegiances missions.Allegiances,
	sizings Sizings, clock cptime.Clock,
) Briefing {
	return Briefing{
		store: store, calendar: calendar, planet: planet, standings: standings, allegiances: allegiances,
		sizings: sizings, clock: clock,
	}
}

type Briefing struct {
	store       Store
	calendar    Calendar
	planet      missions.Planet
	standings   Standings
	allegiances missions.Allegiances
	sizings     Sizings
	clock       cptime.Clock
}

// Drawn once a day, from the map then, the season's table and what the flag did the day before. A day kept with no
// mission, as one an operator's fix cleared, is drawn again. Early in the day, a table that lacks a round before it is
// ErrUnsettled, and nothing is drawn.
func (b Briefing) Of(ctx context.Context, day days.Day, flag string) (Day, error) {
	kept, err := b.store.Day(ctx, day, flag)
	if err == nil && kept.mission != nil || err != nil && !errors.Is(err, ErrNoDay) {
		return kept, err //nolint:wrapcheck // the store named it.
	}

	yesterday, err := b.dayBefore(ctx, day, flag)
	if err != nil {
		return Day{}, fmt.Errorf("failed to read what %s did the day before: %w", flag, err)
	}
	standing, err := b.standings.Standing(ctx, day)
	if err != nil {
		return Day{}, fmt.Errorf("failed to read the season's table to draw %s's mission: %w", flag, err)
	}
	if !standing.Settled() && b.clock.Now().Before(day.Snapshot().StartsAt.Add(settleWithin)) {
		return Day{}, fmt.Errorf("%w: %s's mission waits for its points", ErrUnsettled, flag)
	}
	world, err := b.planet.World(ctx)
	if err != nil {
		return Day{}, fmt.Errorf("failed to read the map to draw %s's mission: %w", flag, err)
	}
	sizing, err := b.sizings.Sizing(ctx)
	if err != nil {
		return Day{}, fmt.Errorf("failed to read the numbers to draw %s's mission with: %w", flag, err)
	}
	return b.store.KeepDay(ctx, day, flag, Draw(day, flag, world, standing, yesterday, sizing)) //nolint:wrapcheck // the store named it.
}

// A season's first day, or a day the flag did nothing on, has a zero day before.
func (b Briefing) dayBefore(ctx context.Context, day days.Day, flag string) (Day, error) {
	before, err := b.calendar.DayBefore(ctx, day)
	if errors.Is(err, days.ErrNoDay) {
		return Day{}, nil
	}
	if err != nil {
		return Day{}, err //nolint:wrapcheck // named by the caller.
	}
	kept, err := b.store.Day(ctx, before, flag)
	if errors.Is(err, ErrNoDay) {
		return Day{}, nil
	}
	return kept, err //nolint:wrapcheck // named by the caller.
}

// The day of the flag the account plays now; an account with no flag has none.
func (b Briefing) OfPlayer(ctx context.Context, day days.Day, account missions.AccountID) (Day, error) {
	flag, err := b.allegiances.FlagOf(ctx, account)
	if err != nil {
		return Day{}, fmt.Errorf("failed to read the flag the account plays for: %w", err)
	}
	if flag == "" {
		return Day{}, nil
	}
	return b.Of(ctx, day, flag)
}
