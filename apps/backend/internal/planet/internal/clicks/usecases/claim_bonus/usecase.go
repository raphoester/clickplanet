// Package claim_bonus redeems a box and starts the boost it is worth.
package claim_bonus

import (
	"context"
	"errors"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/bonus"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

// ErrNoSuchBonus covers every way a claim can fail and tells nobody which:
// unknown, spent, lapsed and somebody else's are the same answer, because the
// difference is what a script guessing tokens would measure.
var ErrNoSuchBonus = errors.New("no bonus to claim")

type Registry interface {
	Claim(token string, scope string) (bonus.Reward, bool)
	Publish(taken bonus.Taken)
	Multiplier() float64
}

// Booster widens a caller's allowance. It is the same limiter the throttle
// spends: a bonus that did not move that bucket would not be a bonus.
type Booster interface {
	Boost(key string, multiplier float64, until time.Time) cpratelimit.State
}

type In struct {
	Token     string
	CountryID string
}

type Out struct {
	Budget   cpratelimit.State
	Kind     bonus.Kind
	Duration time.Duration
}

func New(registry Registry, booster Booster, clock cptime.Clock) *UseCase {
	if clock == nil {
		clock = cptime.SystemClock{}
	}

	return &UseCase{registry: registry, booster: booster, clock: clock}
}

type UseCase struct {
	registry Registry
	booster  Booster
	clock    cptime.Clock
}

// Execute derives the scope the way the throttle derives its bucket key, which
// is what ties the offer, the claim and the boosted bucket to one caller by
// construction rather than by agreement.
func (u *UseCase) Execute(ctx context.Context, in In) (Out, error) {
	scope := cpctx.RateLimitKey(ctx)

	reward, ok := u.registry.Claim(in.Token, scope)
	if !ok {
		return Out{}, ErrNoSuchBonus
	}

	state := u.booster.Boost(scope, u.registry.Multiplier(), u.clock.Now().Add(reward.Duration))

	// Only once the boost has landed: a catch announced to the planet that then
	// failed to apply is the one lie this could tell.
	u.registry.Publish(bonus.Taken{CountryID: in.CountryID, Kind: reward.Kind})

	return Out{Budget: state, Kind: reward.Kind, Duration: reward.Duration}, nil
}
