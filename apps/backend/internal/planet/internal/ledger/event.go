package ledger

import (
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

var ErrUnknownKind = errors.New("an event of a kind this ledger does not know")

type Event interface {
	Replay(see func(Taking))
	Strikes(see func(Strike))
	Show(see func(Scene))
	Entry() (Entry, error)
}

// Entry is an event as it is kept: its stage is the map it was made on and that map's board, which its tiles are ids of.
type Entry struct {
	Stage    clicks.Stage
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
	kindTake:     takingOf,
	kindStrike:   strikingOf,
	kindSpread:   spreadingOf,
	kindEnclose:  enclosingOf,
	kindBomb:     bombingOf,
	kindShield:   shieldingOf,
	kindFortify:  fortifyingOf,
	kindKnockOut: knockingOutOf,
	kindVirus:    infectingOf,
	kindMine:     miningOf,
}

func EventOf(entry Entry) (Event, error) {
	of, ok := kinds[entry.Kind]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnknownKind, entry.Kind)
	}
	return of(entry)
}
