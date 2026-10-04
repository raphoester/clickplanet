package standings

import "maps"

type Tally struct {
	Main  Country
	Tiles map[Country]uint64
}

func (t Tally) WithTake(country Country) Tally {
	tiles := maps.Clone(t.Tiles)
	if tiles == nil {
		tiles = map[Country]uint64{}
	}
	tiles[country]++

	main := t.Main
	if t.Empty() || tiles[country] > tiles[main] {
		main = country
	}
	return Tally{Main: main, Tiles: tiles}
}

func (t Tally) Empty() bool {
	return t.Main == ""
}
