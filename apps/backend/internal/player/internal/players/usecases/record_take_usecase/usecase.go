// Package record_take_usecase counts a tile an account took, for its stats.
package record_take_usecase

import (
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type Stats interface {
	RecordTake(account players.AccountID, at time.Time)
}

type UseCase struct {
	stats Stats
}

func New(stats Stats) *UseCase {
	return &UseCase{stats: stats}
}

type In struct {
	Account players.AccountID
	At      time.Time
}

func (u *UseCase) Execute(in In) {
	u.stats.RecordTake(in.Account, in.At)
}
