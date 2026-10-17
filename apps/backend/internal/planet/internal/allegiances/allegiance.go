package allegiances

import (
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

type Account string

const NoAccount Account = ""

func AccountOf(payer clicks.Payer) Account {
	return Account(payer.Account)
}

type Allegiance struct {
	flag   string
	change Change
}

func (a Allegiance) Pledged(flag string, at time.Time) (Allegiance, error) {
	switch a.flag {
	case flag:
		return a, nil
	case "":
		return Allegiance{flag: flag}, nil
	}

	if until := a.change.ChangeableAt(); at.Before(until) {
		return Allegiance{}, Refusal{Flag: a.flag, Until: until}
	}

	return Allegiance{flag: flag, change: ChangedAt(at)}, nil
}

func (a Allegiance) TakeSnapshot() Snapshot {
	return Snapshot{Flag: a.flag, ChangedAt: a.change.at}
}

type Snapshot struct {
	Flag      string
	ChangedAt time.Time
}

func (s Snapshot) Restore() Allegiance {
	return Allegiance{flag: s.Flag, change: ChangedAt(s.ChangedAt)}
}
