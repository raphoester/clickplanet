package missions

import (
	"context"
	"errors"
	"fmt"
)

var ErrNoBrief = errors.New("no brief was drawn for the account that day")

type Store interface {
	Brief(ctx context.Context, day Day, account AccountID) (Brief, error)
	// The first brief kept for the day and the account wins, and is the answer.
	KeepBrief(ctx context.Context, day Day, account AccountID, brief Brief) (Brief, error)

	Sheet(ctx context.Context, day Day, account AccountID) (Sheet, error)
	ChangeSheet(ctx context.Context, day Day, account AccountID, change func(Sheet) Sheet) error

	DeleteAccount(ctx context.Context, account AccountID) error
}

type Planet interface {
	World(ctx context.Context) (World, error)
}

type Allegiances interface {
	FlagOf(ctx context.Context, account AccountID) (string, error)
}

func NewBriefing(store Store, planet Planet, allegiances Allegiances) Briefing {
	return Briefing{store: store, planet: planet, allegiances: allegiances}
}

type Briefing struct {
	store       Store
	planet      Planet
	allegiances Allegiances
}

// Drawn once a day, for the flag the account plays then.
func (b Briefing) Of(ctx context.Context, day Day, account AccountID) (Brief, error) {
	brief, err := b.store.Brief(ctx, day, account)
	if !errors.Is(err, ErrNoBrief) {
		return brief, err //nolint:wrapcheck // the store named it.
	}

	flag, err := b.allegiances.FlagOf(ctx, account)
	if err != nil {
		return Brief{}, fmt.Errorf("failed to read the flag the account plays for: %w", err)
	}
	if flag == "" {
		return Brief{}, nil
	}
	world, err := b.planet.World(ctx)
	if err != nil {
		return Brief{}, fmt.Errorf("failed to read the map to draw %s's missions: %w", flag, err)
	}
	return b.store.KeepBrief(ctx, day, account, Draw(day, account, flag, world)) //nolint:wrapcheck // the store named it.
}
