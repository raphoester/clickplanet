package clicks

import "context"

type Shields interface {
	Shields(tile uint32) int
	Strike(ctx context.Context, tile uint32, owner string) bool
}

func NewShielding(shields Shields) Shielding {
	return Shielding{shields: shields}
}

type Shielding struct {
	shields Shields
}

func (s Shielding) Outcome(tile uint32, owner, flag string) Outcome {
	return OutcomeOf(owner, flag, s.shields.Shields(tile))
}

func (s Shielding) Strike(ctx context.Context, tile uint32, owner, flag string) Outcome {
	outcome := s.Outcome(tile, owner, flag)
	if outcome == Shielded && !s.shields.Strike(ctx, tile, owner) {
		return Taken
	}

	return outcome
}

func ShieldError(owner, country string, shields, most int) error {
	switch {
	case country == "" || owner != country:
		return ErrNotYourTile
	case shields >= most:
		return ErrTileFull
	}

	return nil
}
