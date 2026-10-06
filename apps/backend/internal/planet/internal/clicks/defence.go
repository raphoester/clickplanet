package clicks

import "context"

type Defenders interface {
	Defenders(tile uint32) int
	Strike(ctx context.Context, tile uint32, owner string) bool
}

func NewDefence(defenders Defenders) Defence {
	return Defence{defenders: defenders}
}

type Defence struct {
	defenders Defenders
}

func (d Defence) Outcome(tile uint32, owner, flag string) Outcome {
	return OutcomeOf(owner, flag, d.defenders.Defenders(tile))
}

func (d Defence) Strike(ctx context.Context, tile uint32, owner, flag string) Outcome {
	outcome := d.Outcome(tile, owner, flag)
	if outcome == Defended && !d.defenders.Strike(ctx, tile, owner) {
		return Taken
	}

	return outcome
}

func ReinforceError(owner, country string, defenders, most int) error {
	switch {
	case country == "" || owner != country:
		return ErrNotYourTile
	case defenders >= most:
		return ErrTileFull
	}

	return nil
}
