package clicks

// BoardID is a board's row in planet's boards: what each tile, fortify lock, mine, virus and event of the ledger names
// its board by. A tile id means a place only on its board.
type BoardID int16

// BoardName names a board everywhere else: its directory in /map/boards, and the wire. "earth-1".
type BoardName string

// Stage is a map on the board it plays: where a write lands, and what each event of the ledger names.
type Stage struct {
	Map   MapID
	Board BoardID
}

// On is the map on a board.
func (m MapID) On(board BoardID) Stage {
	return Stage{Map: m, Board: board}
}
