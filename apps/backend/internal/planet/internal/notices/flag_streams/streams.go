package flag_streams

import "github.com/raphoester/clickplanet.lol-backend/internal/shared/cpflags"

type Board interface {
	PublishFlagsChanged()
}

func New(board Board) Streams {
	return Streams{board: board}
}

// Streams tells every stream open that a flag was opened or closed.
type Streams struct {
	board Board
}

var _ cpflags.Teller = Streams{}

func (s Streams) FlagsChanged() {
	s.board.PublishFlagsChanged()
}
