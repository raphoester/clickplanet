// Package claim_bonus_usecase redeems a box and hands over the charge it is worth.
package claim_bonus_usecase

import (
	"context"
	"errors"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

// ErrNoSuchBonus covers every way a claim can fail and tells nobody which:
// unknown, spent, lapsed and somebody else's are the same answer, because the
// difference is what a script guessing tokens would measure.
var ErrNoSuchBonus = errors.New("no bonus to claim")

type Registry interface {
	Claim(token string, scope string) (bonuses.Reward, bool)
	Publish(taken bonuses.Taken)
}

// Charger hands a caller the charge a box was worth, for the click chain, drop_bomb and use_refill to spend.
type Charger interface {
	Grant(holder bonuses.Holder, kind bonuses.Kind, amount int)
	Held(holder bonuses.Holder) bonuses.Held
}

type In struct {
	Token     string
	CountryID string
}

type Out struct {
	Kind bonuses.Kind

	// How much the box gave: enclosures or spread clicks. One for a refill or a bomb.
	Amount int

	// What the caller holds once the charge is granted.
	Held bonuses.Held
}

func New(registry Registry, charger Charger) *UseCase {
	return &UseCase{registry: registry, charger: charger}
}

type UseCase struct {
	registry Registry
	charger  Charger
}

// Execute derives the payer the way the throttle does, which ties the offer and the claim to one scope,
// and the charge to the account that scope's click spends, by construction rather than by agreement.
func (u *UseCase) Execute(ctx context.Context, in In) (Out, error) {
	payer := clicks.PayerOf(ctx)

	reward, ok := u.registry.Claim(in.Token, payer.Scope)
	if !ok {
		return Out{}, ErrNoSuchBonus
	}

	holder := bonuses.HolderOf(payer)
	u.charger.Grant(holder, reward.Kind, reward.Amount)

	// Only once the charge is held: a catch announced to the planet that then failed to apply is the one
	// lie this could tell.
	u.registry.Publish(bonuses.Taken{CountryID: in.CountryID, Kind: reward.Kind})

	return Out{Kind: reward.Kind, Amount: reward.Amount, Held: u.charger.Held(holder)}, nil
}
