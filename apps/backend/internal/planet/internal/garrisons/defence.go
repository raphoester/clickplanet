package garrisons

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

type Strikes interface {
	Defenders(tile uint32, owner string) int
	Strike(tile uint32, owner string) bool
}

func NewDefence(garrisons Strikes) Defence {
	return Defence{garrisons: garrisons}
}

type Defence struct {
	garrisons Strikes
}

func (d Defence) Outcome(tile uint32, owner, flag string) clicks.Outcome {
	return clicks.OutcomeOf(owner, flag, d.garrisons.Defenders(tile, owner))
}

func (d Defence) Strike(tile uint32, owner, flag string) clicks.Outcome {
	outcome := d.Outcome(tile, owner, flag)
	if outcome == clicks.Defended && !d.garrisons.Strike(tile, owner) {
		return clicks.Taken
	}

	return outcome
}

type Owners interface {
	Owner(tile uint32) (string, bool)
}

type Clearer interface {
	Clear(ctx context.Context, blast clicks.Blast) (clicks.Blast, error)
}

func NewShelter(clearer Clearer, owners Owners, garrisons Strikes) Shelter {
	return Shelter{clearer: clearer, owners: owners, garrisons: garrisons}
}

type Shelter struct {
	clearer   Clearer
	owners    Owners
	garrisons Strikes
}

func (s Shelter) Clear(ctx context.Context, blast clicks.Blast) (clicks.Blast, error) {
	exposed := make([]uint32, 0, len(blast.Cleared))
	for _, tile := range blast.Cleared {
		owner, _ := s.owners.Owner(tile)
		if s.garrisons.Strike(tile, owner) {
			continue
		}
		exposed = append(exposed, tile)
	}
	blast.Cleared = exposed

	return s.clearer.Clear(ctx, blast) //nolint:wrapcheck // a pure delegation: the storage already named what failed.
}
