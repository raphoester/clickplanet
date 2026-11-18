package ledger

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

const kindBoardSwitched = "board_switched"

// Switching is a map moved onto another board, its land carried over (docs/boards.md): no take, no strike and no
// scene, so nobody gains a tile, XP or a mission's progress, and no replay draws it. It names the new board's stage: the
// readers start that board's era at its time.
type Switching struct {
	From, To           clicks.BoardName
	FromTiles, ToTiles uint32
	At                 time.Time
}

func (s Switching) Replay(func(Taking)) {}

func (s Switching) Strikes(func(Strike)) {}

func (s Switching) Show(func(Scene)) {}

type switchingPayload struct {
	From      clicks.BoardName `json:"from"`
	To        clicks.BoardName `json:"to"`
	FromTiles uint32           `json:"fromTiles"`
	ToTiles   uint32           `json:"toTiles"`
}

func (s Switching) Entry() (Entry, error) {
	payload, err := json.Marshal(switchingPayload{From: s.From, To: s.To, FromTiles: s.FromTiles, ToTiles: s.ToTiles})
	if err != nil {
		return Entry{}, fmt.Errorf("failed to write down the switch from %s to %s: %w", s.From, s.To, err)
	}
	return Entry{Kind: kindBoardSwitched, At: s.At, Payload: payload}, nil
}

func switchingOf(entry Entry) (Event, error) {
	var payload switchingPayload
	if err := json.Unmarshal(entry.Payload, &payload); err != nil {
		return nil, fmt.Errorf("failed to read the switch of board: %w", err)
	}
	return Switching{
		From: payload.From, To: payload.To, FromTiles: payload.FromTiles, ToTiles: payload.ToTiles, At: entry.At,
	}, nil
}
