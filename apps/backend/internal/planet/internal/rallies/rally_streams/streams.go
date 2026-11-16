package rally_streams

import (
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/notices"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rallies"
)

type Board interface {
	PublishRally(called notices.RallyCalled)
}

func New(board Board) Streams {
	return Streams{board: board}
}

// Streams tells every stream open of a rally as it is called, with the whole of its time left.
type Streams struct {
	board Board
}

var _ rallies.Teller = Streams{}

func (s Streams) Called(rally rallies.Rally) {
	kept := rally.TakeSnapshot()
	s.board.PublishRally(notices.RallyCalled{Rally: kept, Left: kept.EndsAt.Sub(kept.CalledAt)})
}
