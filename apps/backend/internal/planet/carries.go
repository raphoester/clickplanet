package planet

import (
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/embedded_board"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/eras"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
)

// carriesOf reads the carry-over of a switch of the world's board for its history, from the files this process carries:
// a history across a switch with none is refused rather than drawn on a blank board.
type carriesOf struct {
	registry *embedded_board.Registry
}

func (c carriesOf) Carry(from, to eras.Board) (ledger.Carry, error) {
	return c.registry.Carryover(from.Name, to.Name) //nolint:wrapcheck // the registry names both boards.
}
