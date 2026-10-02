package players

import "slices"

type TitleID string

type Title interface {
	ID() TitleID
	Name() string
	EarnedBy(career Career) bool
}

type TitleIDs []TitleID

func (t TitleIDs) Without(held TitleIDs) TitleIDs {
	return slices.DeleteFunc(slices.Clone(t), func(id TitleID) bool { return slices.Contains(held, id) })
}

type Grants map[AccountID]TitleIDs

type Catalog []Title

func (c Catalog) EarnedBy(career Career) TitleIDs {
	var earned TitleIDs
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

func (c Catalog) Of(held TitleIDs) []Title {
	var titles []Title
	for _, title := range c {
		if slices.Contains(held, title.ID()) {
			titles = append(titles, title)
		}
	}
	return titles
}

func (c Catalog) Without(ids TitleIDs) Catalog {
	return slices.DeleteFunc(slices.Clone(c), func(title Title) bool { return slices.Contains(ids, title.ID()) })
}

func (c Catalog) IDs() TitleIDs {
	ids := make(TitleIDs, 0, len(c))
	for _, title := range c {
		ids = append(ids, title.ID())
	}
	return ids
}
