// Package record_allegiance_usecase counts a tile taken for the flag of the account and the scope that took it.
package record_allegiance_usecase

import (
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

type Allegiances interface {
	Record(country string, at time.Time, keys ...clicks.AllegianceKey)
}

type In struct {
	Account string
	Scope   string
	Country string
	At      time.Time
}

func New(allegiances Allegiances) *UseCase {
	return &UseCase{allegiances: allegiances}
}

type UseCase struct {
	allegiances Allegiances
}

func (u *UseCase) Execute(in In) {
	u.allegiances.Record(in.Country, in.At, clicks.Payer{Scope: in.Scope, Account: in.Account}.AllegianceKeys()...)
}
