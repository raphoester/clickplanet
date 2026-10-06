package clicks

type TileUpdate struct {
	Tile     uint32
	Value    string
	Previous string
	Clicked  bool
}

type Blast struct {
	Tile      uint32
	CountryID string
	Point     Vec3
	Radius    float64
	Cleared   []uint32
	// Owners[i] is the flag Cleared[i] wore before the blast: Clear fills it.
	Owners []string
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
}

type Restoration struct {
	Tile uint32
	From string
	To   string
}
