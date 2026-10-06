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

	Struck []uint32
}

type Change struct {
	Update *TileUpdate
	Blast  *Blast
}

// Tiles holds two little-endian bytes per tile, each an index into Codes.
type DenseBatch struct {
	Start uint32
	Codes []string
	Tiles []byte

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
