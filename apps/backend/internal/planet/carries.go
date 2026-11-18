package planet

import (
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/eras"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
)

var errNoCarry = errors.New("no carry-over is read between these boards")

// carriesUnread refuses a history across a switch of the world's board until the switch reads its carry-over files.
type carriesUnread struct{}

func (carriesUnread) Carry(from, to eras.Board) (ledger.Carry, error) {
	return nil, fmt.Errorf("%w: %s to %s", errNoCarry, from.Name, to.Name)
}
