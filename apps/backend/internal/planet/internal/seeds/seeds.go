// Package seeds holds the patterns an arena's fresh map may start from. The arena knows none of them: each is an
// arenas.Seed, and a rule set names one.
package seeds

import (
	"context"
	"errors"
	"fmt"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/arenas"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
)

// Pattern is the seed a rule set names, with its numbers.
type Pattern interface {
	Validate() error
	// Seed is the pattern laid on the ground of this map.
	Seed(ground Ground) arenas.Seed
}

// Ground is what a pattern seeds from: the board, and the flags with no ground on it that stand on a pin.
type Ground struct {
	Board  *clicks.Ground
	Pinned []clicks.Pinned
}

var Part = cprules.PartOf[*planetv1.Rules, Pattern]{
	Key: "seed",
	Read: func(m *planetv1.Rules) Pattern {
		switch pattern := m.GetSeed().GetPattern().(type) {
		case *planetv1.Seed_Capitals:
			return Capitals{Shields: int(pattern.Capitals.GetShields())}
		case nil:
			return none{}
		}
		return unknown{}
	},
}

// Pins are the flags a player may play now that have no ground and stand on a pin.
type Pins interface {
	Pinned(ctx context.Context) ([]clicks.Pinned, error)
}

func NewCatalog(pins Pins) Catalog {
	return Catalog{pins: pins}
}

// Catalog seeds an arena's map with the pattern its set names, on the board the arena plays.
type Catalog struct {
	pins Pins
}

var _ arenas.Seeds = Catalog{}

func (c Catalog) SeedOf(ctx context.Context, set cprules.Set[*planetv1.Rules], board *clicks.Ground) (arenas.Seed, error) {
	pinned, err := c.pins.Pinned(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read the flags that stand on a pin: %w", err)
	}
	return Part.Of(set).Seed(Ground{Board: board, Pinned: pinned}), nil
}

// A set that names no pattern, and default holds none: no set can play.
type none struct{}

func (none) Validate() error { return errors.New("pattern: none is named") }

func (none) Seed(Ground) arenas.Seed { return blank{} }

// A pattern of a build after this one.
type unknown struct{}

func (unknown) Validate() error { return errors.New("pattern: this build knows no such pattern") }

func (unknown) Seed(Ground) arenas.Seed { return blank{} }

type blank struct{}

func (blank) Tiles() []clicks.StartTile { return nil }

func (blank) Seats() []clicks.Capital { return nil }
