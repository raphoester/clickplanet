package clicks

// Every write of the map counts one in Seq; Epoch is the second the process started, when Seq starts from zero.
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

type Census struct {
	Countries  []FlagTiles
	Landmasses []LandmassTiles
}
