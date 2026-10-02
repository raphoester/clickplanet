package titles

import (
	"slices"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type ID string

type Title interface {
	ID() ID
	Name() string
	EarnedBy(career Career) bool
}

type IDs []ID

func (i IDs) Without(held IDs) IDs {
	return slices.DeleteFunc(slices.Clone(i), func(id ID) bool { return slices.Contains(held, id) })
}

type Grants map[players.AccountID]IDs

type Catalog []Title

func (c Catalog) EarnedBy(career Career) IDs {
	var earned IDs
	for _, title := range c {
		if title.EarnedBy(career) {
			earned = append(earned, title.ID())
		}
	}
	return earned
}

func (c Catalog) GrantsFor(careers []Career) Grants {
	grants := Grants{}
	for _, career := range careers {
		if earned := c.EarnedBy(career); len(earned) > 0 {
			grants[career.Stats.Account] = earned
		}
	}
	return grants
}

func (c Catalog) Of(held IDs) []Title {
	var titles []Title
	for _, title := range c {
		if slices.Contains(held, title.ID()) {
			titles = append(titles, title)
		}
	}
	return titles
}

func (c Catalog) Without(ids IDs) Catalog {
	return slices.DeleteFunc(slices.Clone(c), func(title Title) bool { return slices.Contains(ids, title.ID()) })
}

func (c Catalog) IDs() IDs {
	ids := make(IDs, 0, len(c))
	for _, title := range c {
		ids = append(ids, title.ID())
	}
	return ids
}
