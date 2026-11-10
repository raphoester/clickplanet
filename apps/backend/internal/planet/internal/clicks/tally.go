package clicks

// Every write counts one in Seq, from zero since Epoch: the second the map last started (boot, reset, arena).
type MapVersion struct {
	Epoch uint32
	Seq   uint64
}

type FlagTiles struct {
	Flag  string
	Tiles uint32
}

type LandmassTiles struct {
	Landmass LandmassID
	Flags    []FlagTiles
}

type Tally struct {
	Countries  []FlagTiles
	Landmasses []LandmassTiles
}
