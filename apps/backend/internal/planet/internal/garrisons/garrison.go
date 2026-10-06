package garrisons

type Garrison struct {
	Tile    uint32
	Country string

	Defenders int
}

func (g Garrison) Standing(owner string) int {
	if owner == "" || g.Country != owner {
		return 0
	}

	return g.Defenders
}

func (g Garrison) Full(country string, most int) bool {
	return g.Standing(country) >= most
}

func (g Garrison) Reinforced(country string, most int) Garrison {
	if g.Country != country {
		g = Garrison{Tile: g.Tile, Country: country}
	}
	g.Defenders = min(g.Defenders+1, most)

	return g
}

func (g Garrison) Struck(owner string) (Garrison, bool) {
	if g.Standing(owner) == 0 {
		return Garrison{Tile: g.Tile}, false
	}
	g.Defenders--

	return g, true
}

func (g Garrison) Empty() bool {
	return g.Defenders <= 0
}
