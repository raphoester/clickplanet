// Package record_allegiance_usecase counts a tile taken for the flag of the account and the scope that took it.
package record_allegiance_usecase

import (
	"time"
)

type Allegiances interface {
	Record(account string, scope string, country string, at time.Time)
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
	u.allegiances.Record(in.Account, in.Scope, in.Country, in.At)
}
