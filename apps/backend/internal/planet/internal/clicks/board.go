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
