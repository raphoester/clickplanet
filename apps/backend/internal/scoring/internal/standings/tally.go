package standings

import "maps"

// A take can be told twice: one below the mark was counted already. Takes come in ledger order.
type Tally struct {
	Main       Country
	Tiles      map[Country]uint64
	TakesBelow uint64
}

func (t Tally) WithTake(take Take) Tally {
	if take.Position < t.TakesBelow {
		return t
	}

	tiles := maps.Clone(t.Tiles)
	if tiles == nil {
		tiles = map[Country]uint64{}
	}
	tiles[take.Country]++

	main := t.Main
	if t.Empty() || tiles[take.Country] > tiles[main] {
		main = take.Country
	}
	return Tally{Main: main, Tiles: tiles, TakesBelow: take.Position + 1}
}

func (t Tally) Empty() bool {
	return t.Main == ""
}
