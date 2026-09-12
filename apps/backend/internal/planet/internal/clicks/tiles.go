package clicks

// TileUpdate is one tile changing hands, as it is published to every open feed.
type TileUpdate struct {
	Tile     uint32
	Value    string
	Previous string
}

// Blast is a bomb landing, as one event so a client can hold the clear back until the blast hits.
type Blast struct {
	// Tile is 0 for a bomb that fell in the sea.
	Tile      uint32
	CountryID string
	Point     Vec3
	Radius    float64
	Cleared   []uint32
}

// Change is one thing that happened to the map; one feed keeps blasts and updates in order.
type Change struct {
	Update *TileUpdate
	Blast  *Blast
}

// DenseBatch is a span of the map with the country codes interned into Codes
// and two bytes per tile in Tiles indexing into it. Tile ids are implicit in
// the position, which is what keeps a full map at ~516 KB.
type DenseBatch struct {
	Start uint32
	Codes []string
	Tiles []byte
}
