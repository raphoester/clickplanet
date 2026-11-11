// Package seeds holds the patterns an arena's fresh map may start from. The arena knows none of them: each is an
// arenas.Seed, and a rule set names one.
package seeds

import (
	"errors"

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

// Ground is what the map gives a pattern to seed from.
type Ground struct {
	Capitals *clicks.Capitals
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

// Catalog seeds an arena's map with the pattern its set names.
type Catalog struct {
	ground Ground
}

var _ arenas.Seeds = Catalog{}

func NewCatalog(ground Ground) Catalog {
	return Catalog{ground: ground}
}

func (c Catalog) SeedOf(set cprules.Set[*planetv1.Rules]) arenas.Seed {
	return Part.Of(set).Seed(c.ground)
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
