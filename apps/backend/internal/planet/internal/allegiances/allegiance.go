package allegiances

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

type Account string

const NoAccount Account = ""

func AccountOf(payer clicks.Payer) Account {
	return Account(payer.Account)
}

var ErrInvalidAccount = errors.New("not an account id")

func AccountFrom(value string) (Account, error) {
	id, err := uuid.Parse(value)
	if err != nil || id == uuid.Nil {
		return NoAccount, fmt.Errorf("%w: %q", ErrInvalidAccount, value)
	}
	return Account(id.String()), nil
}

type Allegiance struct {
	flag   string
	change Change
}

func (a Allegiance) Pledged(flag string, at time.Time, lock Lock) (Allegiance, error) {
	switch a.flag {
	case flag:
		return a, nil
	case "":
		return Allegiance{flag: flag}, nil
	}

	until := a.change.ChangeableAt()
	if lock {
		return Allegiance{}, Refusal{Flag: a.flag, Until: until, Locked: true, Reason: ErrLocked}
	}
	if at.Before(until) {
		return Allegiance{}, Refusal{Flag: a.flag, Until: until, Reason: ErrChangedTooSoon}
	}

	return Allegiance{flag: flag, change: ChangedAt(at)}, nil
}

func (a Allegiance) PlayError(flag string, lock Lock) error {
	switch a.flag {
	case flag:
		return nil
	case "":
		return Refusal{Reason: ErrNoFlag}
	}

	return Refusal{Flag: a.flag, Until: a.change.ChangeableAt(), Locked: bool(lock), Reason: ErrOtherFlag}
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
