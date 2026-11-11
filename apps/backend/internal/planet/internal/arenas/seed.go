package arenas

import (
	"context"
	"errors"
	"fmt"
	"time"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
)

// Seed is how an arena's fresh map starts: the tiles it holds before any take. The arena knows no pattern; each pattern
// is a Seed.
type Seed interface {
	Tiles() []clicks.StartTile
}

// Seeds is the seed a rule set names.
type Seeds interface {
	SeedOf(set cprules.Set[*planetv1.Rules]) Seed
}

// Setup is how an arena opens: the freeze around its map and the seed it starts from, both read from the battle's set.
type Setup struct {
	Freeze time.Duration
	Seed   Seed
}

type Sets interface {
	Set(ctx context.Context, name cprules.SetName) (cprules.Set[*planetv1.Rules], error)
}

// Setups reads an arena's setup from the rule set it is opened for, never from the set in force: the battle's set plays
// only once its arena opened.
type Setups struct {
	sets  Sets
	seeds Seeds
}

func NewSetups(sets Sets, seeds Seeds) Setups {
	return Setups{sets: sets, seeds: seeds}
}

// Setup is refused for a set planet keeps none of, or one that cannot play: asked again, it never would.
func (s Setups) Setup(ctx context.Context, name cprules.SetName) (Setup, error) {
	set, err := s.sets.Set(ctx, name)
	if errors.Is(err, cprules.ErrUnknownSet) || errors.Is(err, cprules.ErrInvalidSet) {
		return Setup{}, fmt.Errorf("%w: rule set %q: %w", ErrRefused, name, err)
	}
	if err != nil {
		return Setup{}, fmt.Errorf("failed to read rule set %q: %w", name, err)
	}

	return Setup{Freeze: Part.Of(set).Freeze, Seed: s.seeds.SeedOf(set)}, nil
}
