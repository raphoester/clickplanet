package players

import "slices"

type TitleID string

type Title interface {
	ID() TitleID
	Name() string
	EarnedBy(stats Stats) bool
}

type TitleIDs []TitleID

func (t TitleIDs) Without(held TitleIDs) TitleIDs {
	return slices.DeleteFunc(slices.Clone(t), func(id TitleID) bool { return slices.Contains(held, id) })
}

type Grants map[AccountID]TitleIDs

type Catalog []Title

func (c Catalog) EarnedBy(stats Stats) TitleIDs {
	var earned TitleIDs
	for _, title := range c {
		if title.EarnedBy(stats) {
			earned = append(earned, title.ID())
		}
	}
	return earned
}

func (c Catalog) GrantsFor(page []Stats) Grants {
	grants := Grants{}
	for _, stats := range page {
		if earned := c.EarnedBy(stats); len(earned) > 0 {
			grants[stats.Account] = earned
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
