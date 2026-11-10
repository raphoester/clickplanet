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

// Standing says which flags are still in the game: every one outside an arena, and in one each flag with a capital
// not knocked out.
type Standing interface {
	Stands(flag string) bool
}

// Pledged is the flag the account plays for after it asks for flag. A flag that does not stand is refused; an account
// whose flag does not stand changes at once, past the lock and the wait.
func (a Allegiance) Pledged(flag string, at time.Time, lock Lock, standing Standing) (Allegiance, error) {
	switch {
	case a.flag == flag:
		return a, nil
	case !standing.Stands(flag):
		return Allegiance{}, fmt.Errorf("%w: %q", ErrDown, flag)
	case a.flag == "":
		return Allegiance{flag: flag}, nil
	case !standing.Stands(a.flag):
		return Allegiance{flag: flag, change: ChangedAt(at)}, nil
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

func (a Allegiance) Released() Allegiance {
	return Allegiance{flag: a.flag}
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
