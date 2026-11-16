package allegiance_streams

import (
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/allegiances"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
)

// Board is the streams open, each told the mines of the flag its account plays for.
type Board interface {
	Pledged(holder charges.Holder, flag string)
}

func New(board Board) Streams {
	return Streams{board: board}
}

// Streams moves every stream of an account to the flag it pledged.
type Streams struct {
	board Board
}

var _ allegiances.Teller = Streams{}

func (s Streams) Pledged(account allegiances.Account, flag string, _ string) {
	s.board.Pledged(charges.Holder(account), flag)
}
