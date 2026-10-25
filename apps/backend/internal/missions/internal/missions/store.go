package missions

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var ErrNoBrief = errors.New("no brief was drawn for the flag that day")

type Store interface {
	Brief(ctx context.Context, day Day, flag string) (Brief, error)
	// The first brief kept for the day and the flag wins, and is the answer.
	KeepBrief(ctx context.Context, day Day, flag string, brief Brief) (Brief, error)

	Sheet(ctx context.Context, day Day, account AccountID) (Sheet, error)
	ChangeSheet(ctx context.Context, day Day, account AccountID, change func(Sheet) Sheet) error
	// Keeps the collection only if pay succeeds; ErrCollected when the slot was collected already.
	Collect(ctx context.Context, day Day, account AccountID, mission MissionID, at time.Time,
		pay func(context.Context) error) error

	DeleteAccount(ctx context.Context, account AccountID) error
}

type Planet interface {
	World(ctx context.Context) (World, error)
}

func NewBriefing(store Store, planet Planet) Briefing {
	return Briefing{store: store, planet: planet}
}

type Briefing struct {
	store  Store
	planet Planet
}

func (b Briefing) Of(ctx context.Context, day Day, flag string) (Brief, error) {
	brief, err := b.store.Brief(ctx, day, flag)
	if !errors.Is(err, ErrNoBrief) {
		return brief, err //nolint:wrapcheck // the store named it.
	}

	world, err := b.planet.World(ctx)
	if err != nil {
		return Brief{}, fmt.Errorf("failed to read the map to draw %s's missions: %w", flag, err)
	}
	return b.store.KeepBrief(ctx, day, flag, Draw(day, flag, world)) //nolint:wrapcheck // the store named it.
}
