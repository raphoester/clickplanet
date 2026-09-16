// Package claim_bonus_usecase redeems a box and starts the boost it is worth.
package claim_bonus_usecase

import (
	"context"
	"errors"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

// ErrNoSuchBonus covers every way a claim can fail and tells nobody which:
// unknown, spent, lapsed and somebody else's are the same answer, because the
// difference is what a script guessing tokens would measure.
var ErrNoSuchBonus = errors.New("no bonus to claim")

type Registry interface {
	Claim(token string, scope string) (bonuses.Reward, bool)
	Publish(taken bonuses.Taken)
	Multiplier() float64
}

// Booster widens a caller's allowance. It is the same limiter the throttle
// spends: a bonus that did not move that bucket would not be a bonus.
type Booster interface {
	Boost(key string, multiplier float64, until time.Time) cpratelimit.State
	Peek(key cpratelimit.Key) cpratelimit.State
}

type Pricer interface {
	Price(country string) clicks.Price
}

// Spreader starts a spread bonus, which the click chain then reads on every click.
type Spreader interface {
	Grant(scope string, until time.Time)
}

// Bomber hands a caller the bomb a box was worth, for drop_bomb to spend.
type Bomber interface {
	Grant(scope string, until time.Time)
}

// Encloser starts an enclose bonus, which the click chain then spends.
type Encloser interface {
	Grant(scope string, until time.Time, shapes int, maxTiles int)
}

type In struct {
	Token     string
	CountryID string
}

type Out struct {
	Budget   clicks.Budget
	Kind     bonuses.Kind
	Duration time.Duration

	// BlastRadius is set for a bomb only, in radians of arc.
	BlastRadius float64
	// For an enclose bonus only.
	Enclosures        int
	EnclosureMaxTiles int
}

func New(
	registry Registry,
	booster Booster,
	pricer Pricer,
	spreader Spreader,
	bomber Bomber,
	blastRadius float64,
	encloser Encloser,
	buckets clicks.Buckets,
	clock cptime.Clock,
) *UseCase {
	if clock == nil {
		clock = cptime.SystemClock{}
	}

	return &UseCase{
		registry:    registry,
		booster:     booster,
		pricer:      pricer,
		spreader:    spreader,
		bomber:      bomber,
		blastRadius: blastRadius,
		encloser:    encloser,
		buckets:     buckets,
		clock:       clock,
	}
}

type UseCase struct {
	registry    Registry
	booster     Booster
	pricer      Pricer
	spreader    Spreader
	bomber      Bomber
	blastRadius float64
	encloser    Encloser
	buckets     clicks.Buckets
	clock       cptime.Clock
}

// Execute derives the payer the way the throttle does, which ties the offer and the claim to one scope,
// and the boost to the bucket that scope's click spends first, by construction rather than by agreement.
func (u *UseCase) Execute(ctx context.Context, in In) (Out, error) {
	payer := clicks.PayerOf(ctx)

	reward, ok := u.registry.Claim(in.Token, payer.Scope)
	if !ok {
		return Out{}, ErrNoSuchBonus
	}

	state := u.apply(payer, reward)

	// Only once the boost has landed: a catch announced to the planet that then
	// failed to apply is the one lie this could tell.
	u.registry.Publish(bonuses.Taken{CountryID: in.CountryID, Kind: reward.Kind})

	out := Out{
		Budget:            clicks.BudgetOf(state, u.pricer.Price(in.CountryID)),
		Kind:              reward.Kind,
		Duration:          reward.Duration,
		Enclosures:        reward.Enclosures,
		EnclosureMaxTiles: reward.EnclosureMaxTiles,
	}
	if reward.Kind == bonuses.KindBomb {
		out.BlastRadius = u.blastRadius
	}

	return out, nil
}

// apply starts what the reward is worth, and answers the allowance as it stands
// afterwards: the tighter bucket, as a click reports it. A triple widens the account's bucket and never
// the scope's, which the scope's other players share.
func (u *UseCase) apply(payer clicks.Payer, reward bonuses.Reward) cpratelimit.State {
	until := u.clock.Now().Add(reward.Duration)

	switch reward.Kind {
	case bonuses.KindSpreadClicks:
		u.spreader.Grant(payer.Scope, until)
	case bonuses.KindBomb:
		u.bomber.Grant(payer.Scope, until)
	case bonuses.KindEncloseClicks:
		u.encloser.Grant(payer.Scope, until, reward.Enclosures, reward.EnclosureMaxTiles)
	case bonuses.KindTripleClicks:
		u.booster.Boost(u.buckets.Boosted(payer), u.registry.Multiplier(), until)
	}

	keys := u.buckets.Keys(payer)
	states := make([]cpratelimit.State, len(keys))
	for i, key := range keys {
		states[i] = u.booster.Peek(key)
	}

	return clicks.Tightest(states)
}
