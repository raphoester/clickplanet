package clicks

type TileUpdate struct {
	Tile     uint32
	Value    string
	Previous string
	Clicked  bool

	Shields int
}

type Blast struct {
	Tile      uint32
	CountryID string
	Point     Vec3
	Radius    float64
	Cleared   []uint32
	// Owners[i] is the flag Cleared[i] wore before the blast: Clear fills it.
	Owners []string

	Struck []uint32
	// Left[i] is the shields Struck[i] kept: Clear fills it.
	Left []int
}

type Change struct {
	Version       MapVersion
	Update        *TileUpdate
	Blast         *Blast
	Fortification *Fortification
	// The rows the writes since the last census changed, each whole: a flag at zero is listed.
	Census *Census
}

// Tiles holds two little-endian bytes per tile, each an index into Codes.
type DenseBatch struct {
	Version MapVersion
	Start   uint32
	Codes   []string
	Tiles   []byte

	Shields []TileShields
}

type TileShields struct {
	Tile    uint32
	Shields int
}

type Restoration struct {
	Tile uint32
	From string
	To   string
}
