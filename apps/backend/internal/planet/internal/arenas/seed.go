package arenas

import (
	"context"
	"errors"
	"fmt"
	"time"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/eras"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
)

// Seed is how an arena's fresh map starts: the tiles it holds before any take, and the seats it gives the flags with no
// ground, which its map plays as capitals. The arena knows no pattern; each pattern is a Seed.
type Seed interface {
	Tiles() []clicks.StartTile
	Seats() []clicks.Capital
}

// Seeds is the seed a rule set names, laid on a board's ground.
type Seeds interface {
	SeedOf(ctx context.Context, set cprules.Set[*planetv1.Rules], ground *clicks.Ground) (Seed, error)
}

// Grounds are the boards an arena may play: those this process carries and planet keeps a row of.
type Grounds interface {
	GroundOf(id clicks.BoardID) (*clicks.Ground, error)
	Named(name clicks.BoardName) (eras.Board, *clicks.Ground, error)
}

// Setup is how an arena opens, read from the battle's set: the freeze around its map, the board it plays and the seed it
// starts from, laid on that board.
type Setup struct {
	Freeze time.Duration
	Board  clicks.BoardID
	Seed   Seed
}

type Sets interface {
	Set(ctx context.Context, name cprules.SetName) (cprules.Set[*planetv1.Rules], error)
}

// Setups reads an arena's setup from the rule set it is opened for, never from the set in force: the battle's set plays
// only once its arena opened.
type Setups struct {
	sets    Sets
	seeds   Seeds
	grounds Grounds
}

func NewSetups(sets Sets, seeds Seeds, grounds Grounds) Setups {
	return Setups{sets: sets, seeds: seeds, grounds: grounds}
}

// Setup is refused for a set planet keeps none of, one that cannot play, or one whose board this process does not carry
// or planet keeps no row of: asked again, it never would. A set that names no board plays world, the board the world
// plays.
func (s Setups) Setup(ctx context.Context, name cprules.SetName, world clicks.BoardID) (Setup, error) {
	set, err := s.sets.Set(ctx, name)
	if errors.Is(err, cprules.ErrUnknownSet) || errors.Is(err, cprules.ErrInvalidSet) {
		return Setup{}, fmt.Errorf("%w: rule set %q: %w", ErrRefused, name, err)
	}
	if err != nil {
		return Setup{}, fmt.Errorf("failed to read rule set %q: %w", name, err)
	}

	rules := Part.Of(set)
	board, ground, err := s.board(rules.Board, world)
	if errors.Is(err, eras.ErrNotCarried) {
		return Setup{}, fmt.Errorf("%w: rule set %q: %w", ErrRefused, name, err)
	}
	if err != nil {
		return Setup{}, fmt.Errorf("failed to read the board of rule set %q: %w", name, err)
	}
	seed, err := s.seeds.SeedOf(ctx, set, ground)
	if err != nil {
		return Setup{}, fmt.Errorf("failed to seed rule set %q: %w", name, err)
	}
	return Setup{Freeze: rules.Freeze, Board: board, Seed: seed}, nil
}

func (s Setups) board(named clicks.BoardName, world clicks.BoardID) (clicks.BoardID, *clicks.Ground, error) {
	if named == "" {
		ground, err := s.grounds.GroundOf(world)
		return world, ground, err //nolint:wrapcheck // the grounds name the board.
	}
	board, ground, err := s.grounds.Named(named)
	return board.ID, ground, err //nolint:wrapcheck // the grounds name the board.
}
