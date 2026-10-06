package ledger

import (
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

type Bombing struct {
	Scope   string
	Account string
	At      time.Time
	Blast   clicks.Blast
}

func (b Bombing) Landing() Taking {
	taking := Taking{Tile: b.Blast.Tile, Scope: b.Scope, Account: b.Account, At: b.At, Bombed: true}
	for i, tile := range b.Blast.Cleared {
		if tile == b.Blast.Tile {
			taking.Previous = b.Blast.Owners[i]
		}
	}
	return taking
}

func (b Bombing) Takings() []Taking {
	takings := make([]Taking, len(b.Blast.Cleared))
	for i, tile := range b.Blast.Cleared {
		takings[i] = Taking{
			Tile: tile, Scope: b.Scope, Account: b.Account, Previous: b.Blast.Owners[i], At: b.At, Bombed: true,
		}
	}
	return takings
}
