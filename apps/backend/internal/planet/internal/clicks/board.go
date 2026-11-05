package clicks

type Board struct {
	maxIndex uint32
}

func NewBoard(maxIndex uint32) Board {
	return Board{maxIndex: maxIndex}
}

func (b Board) CheckTile(tile uint32) bool {
	return tile > 0 && tile <= b.maxIndex
}

func (b Board) MaxIndex() uint32 {
	return b.maxIndex
}

// A page load reads from tile 1 and clamps its last batch to the map, whatever its batch size.
func (b Board) OffMap(start, end uint32) bool {
	return start == 0 || end > b.maxIndex
}
