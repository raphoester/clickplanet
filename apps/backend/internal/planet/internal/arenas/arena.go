package arenas

import (
	"errors"
	"fmt"
	"time"
	"unicode/utf8"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

var (
	ErrInvalidArena = errors.New("not an arena")
	// ErrRefused is a call that can never apply: made again, it is refused again.
	ErrRefused = errors.New("refused")
)

// ID names an arena as the module that asks for it names it: the battle it is played for.
type ID string

const maxID = 64

func ParseID(id string) (ID, error) {
	if id == "" || utf8.RuneCountInString(id) > maxID || clicks.MapID(id) == clicks.World {
		return "", fmt.Errorf("%w: its id %q is empty, longer than %d, or the world's", ErrInvalidArena, id, maxID)
	}
	return ID(id), nil
}

// Map is the arena's map, named by its id.
func (id ID) Map() clicks.MapID {
	return clicks.MapID(id)
}

// Opening is an arena open: its map played since OpenedAt, for a battle that ends at EndsAt, with no play for Freeze
// after it opened and before it ends.
type Opening struct {
	ID       ID
	EndsAt   time.Time
	OpenedAt time.Time
	Freeze   time.Duration
}

// Freezes is true while the arena takes no play: its freeze after its map played, and from its freeze before the
// battle's end until the world's map is back.
func (o Opening) Freezes(now time.Time) bool {
	return now.Before(o.OpenedAt.Add(o.Freeze)) || !now.Before(o.EndsAt.Add(-o.Freeze))
}

// Closing is an arena closed at a time: its map is deleted and the world's plays again, or it never opened.
type Closing struct {
	ID       ID
	ClosedAt time.Time
}

// Arena is what planet keeps of one: its opening once it opened, and whether it closed.
type Arena struct {
	ID      ID
	Opening Opening
	Opened  bool
	Closed  bool
}

// Played is the map that plays while the arena is as it is: its own while it is open, else the world's.
func (a Arena) Played() clicks.MapID {
	if a.Opened && !a.Closed {
		return a.ID.Map()
	}
	return clicks.World
}

// Open is the arena opened now with its freeze, or false when it is open already. It opens once, never after it closed,
// never at or after its end, and never while another is open: each is refused.
func (a Arena) Open(endsAt, now time.Time, freeze time.Duration, playing Arena) (Opening, bool, error) {
	switch {
	case a.Opened && !a.Closed:
		return Opening{}, false, nil
	case a.Closed:
		return Opening{}, false, fmt.Errorf("%w: arena %s closed already", ErrRefused, a.ID)
	case !now.Before(endsAt):
		return Opening{}, false, fmt.Errorf("%w: arena %s ended at %s", ErrRefused, a.ID, endsAt.UTC())
	case playing.Opened && playing.ID != a.ID:
		return Opening{}, false, fmt.Errorf("%w: arena %s while arena %s is open", ErrRefused, a.ID, playing.ID)
	}
	return Opening{ID: a.ID, EndsAt: endsAt.UTC(), OpenedAt: now.UTC(), Freeze: freeze}, true, nil
}

// Close is the arena closed now, or false when it closed already. One never opened closes too, with no map to bring
// back, so whoever waits for it never waits for ever.
func (a Arena) Close(now time.Time) (Closing, bool) {
	if a.Closed {
		return Closing{}, false
	}
	return Closing{ID: a.ID, ClosedAt: now.UTC()}, true
}
