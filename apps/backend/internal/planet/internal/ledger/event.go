package ledger

import (
	"errors"
	"fmt"
	"time"
)

var ErrUnknownKind = errors.New("an event of a kind this ledger does not know")

type Event interface {
	Replay(see func(Taking))
	Show(see func(Scene))
	Entry() (Entry, error)
}

type Entry struct {
	Kind     string
	Tile     uint32
	Scope    string
	Account  string
	Country  string
	Previous string
	At       time.Time
	Payload  []byte
}

var kinds = map[string]func(Entry) (Event, error){
	kindTake:    takingOf,
	kindStrike:  strikingOf,
	kindSpread:  spreadingOf,
	kindEnclose: enclosingOf,
	kindBomb:    bombingOf,
	kindShield:  shieldingOf,
	kindFortify: fortifyingOf,
}

func EventOf(entry Entry) (Event, error) {
	of, ok := kinds[entry.Kind]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnknownKind, entry.Kind)
	}
	return of(entry)
}
