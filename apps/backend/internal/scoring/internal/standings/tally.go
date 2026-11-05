package standings

import "maps"

// A take can be told twice: one below the mark was counted already. Takes come in ledger order, then index order.
type Tally struct {
	Main       Country
	Tiles      map[Country]uint64
	TakesBelow TakeMark
}

func (t Tally) WithTake(take Take) Tally {
	if t.TakesBelow.counted(take) {
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
	return Tally{Main: main, Tiles: tiles, TakesBelow: takeMarkPast(take)}
}

func (t Tally) Empty() bool {
	return t.Main == ""
}
