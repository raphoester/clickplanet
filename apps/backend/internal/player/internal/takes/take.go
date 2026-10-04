package takes

import (
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

type Position uint64

type Take struct {
	position Position
	account  players.AccountID
	country  string
	at       time.Time
	reverted bool
}

func TakeOf(position Position, account players.AccountID, country string, at time.Time, reverted bool) Take {
	return Take{position: position, account: account, country: country, at: at, reverted: reverted}
}

func (t Take) Position() Position { return t.position }

func (t Take) Countable() bool {
	return t.account != cpsession.NoAccount && t.country != "" && !t.reverted
}
