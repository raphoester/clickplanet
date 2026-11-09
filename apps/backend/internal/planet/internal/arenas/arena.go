package arenas

import (
	"errors"
	"fmt"
	"time"
	"unicode/utf8"
)

var (
	ErrInvalidArena = errors.New("not an arena")
	// ErrRefused is a command that can never apply: delivered again, it would hold back the ones after it.
	ErrRefused = errors.New("refused")
)

// ID names an arena as the module that asks for it names it: the battle it is played for.
type ID string

const maxID = 64

func ParseID(id string) (ID, error) {
	if id == "" || utf8.RuneCountInString(id) > maxID {
		return "", fmt.Errorf("%w: its id %q is empty or longer than %d", ErrInvalidArena, id, maxID)
	}
	return ID(id), nil
}

// Opening is an arena open: the map blank since OpenedAt, for a battle that ends at EndsAt.
type Opening struct {
	ID       ID
	EndsAt   time.Time
	OpenedAt time.Time
}

// Freezes is true while the arena takes no play: freeze after the map went blank, and from freeze before the battle's
// end until the map is back.
func (o Opening) Freezes(now time.Time, freeze time.Duration) bool {
	return now.Before(o.OpenedAt.Add(freeze)) || !now.Before(o.EndsAt.Add(-freeze))
}

// Closing is an arena closed at a time: the map from before it is back, or it never opened.
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

// Open is the arena opened now, or false when it is open already. It opens once, never after it closed, never at or
// after its end, and never while another is open: each is refused.
func (a Arena) Open(endsAt, now time.Time, playing Arena) (Opening, bool, error) {
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
	return Opening{ID: a.ID, EndsAt: endsAt.UTC(), OpenedAt: now.UTC()}, true, nil
}

// Close is the arena closed now, or false when it closed already. One never opened closes too, with no map to bring
// back, so whoever waits for it never waits for ever.
func (a Arena) Close(now time.Time) (Closing, bool) {
	if a.Closed {
		return Closing{}, false
	}
	return Closing{ID: a.ID, ClosedAt: now.UTC()}, true
}
