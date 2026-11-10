package placements

import (
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

var ErrInvalidPlacement = errors.New("not a placement")

// Placement is a shield taken from a hand and not yet saved on the map. It ends saved there, or back in the hand.
type Placement struct {
	id      clicks.PlacementID
	holder  bonuses.Holder
	takenAt time.Time
	ended   bool
}

type Snapshot struct {
	ID      clicks.PlacementID
	Holder  bonuses.Holder
	TakenAt time.Time
	Ended   bool
}

func New(id clicks.PlacementID, holder bonuses.Holder, takenAt time.Time) (Placement, error) {
	return Snapshot{ID: id, Holder: holder, TakenAt: takenAt}.Restore()
}

func (s Snapshot) Restore() (Placement, error) {
	switch {
	case s.ID == clicks.PlacementID{}:
		return Placement{}, fmt.Errorf("%w: it has no id", ErrInvalidPlacement)
	case s.Holder == bonuses.NoHolder:
		return Placement{}, fmt.Errorf("%w: %s has no holder", ErrInvalidPlacement, s.ID)
	case s.TakenAt.IsZero():
		return Placement{}, fmt.Errorf("%w: %s has no time", ErrInvalidPlacement, s.ID)
	}

	return Placement{id: s.ID, holder: s.Holder, takenAt: s.TakenAt.UTC(), ended: s.Ended}, nil
}

func (p Placement) TakeSnapshot() Snapshot {
	return Snapshot{ID: p.id, Holder: p.holder, TakenAt: p.takenAt, Ended: p.ended}
}

func (p Placement) Holder() bonuses.Holder {
	return p.holder
}

func (p Placement) Ended() Placement {
	p.ended = true

	return p
}
