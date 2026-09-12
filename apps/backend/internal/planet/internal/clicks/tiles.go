package clicks

// TileUpdate is one tile changing hands, as it is published to every open feed.
type TileUpdate struct {
	Tile     uint32
	Value    string
	Previous string
}

// DenseBatch is a span of the map with the country codes interned into Codes
// and two bytes per tile in Tiles indexing into it. Tile ids are implicit in
// the position, which is what keeps a full map at ~516 KB.
type DenseBatch struct {
	Start uint32
	Codes []string
	Tiles []byte
}
