package planetv1controller

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/domain/bonus"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpconnect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
)

// BonusRegistry is the local port onto internal/planet/internal/domain/bonus, declared
// here the way DenseMapReader is: the controller names what it needs, and the
// domain does not know it is being served over Connect.
type BonusRegistry interface {
	Attend(scope string) (<-chan bonus.Event, func())
	Clicked(scope string)
	Claim(token string, scope string) (bonus.Reward, bool)
	Publish(taken bonus.Taken)
	Multiplier() float64
}

// ClickBooster widens a caller's allowance for as long as a bonus runs. It is
// the same limiter the throttle spends, because a bonus that did not move the
// bucket the throttle reads would not be a bonus at all.
type ClickBooster interface {
	Boost(key string, multiplier float64, until time.Time) cpratelimit.State
}

// ErrNoSuchBonus covers every way a claim can fail, and says which one to
// nobody. A caller learns that the box is not theirs, never whether the token
// was unknown, spent, lapsed, or somebody else's — the difference is exactly
// what a script guessing tokens would measure.
var ErrNoSuchBonus = errors.New("no bonus to claim")

// ClaimBonus redeems a box and starts the boost.
//
// The scope is derived the same way the throttle derives its bucket key, which
// is what ties the three together: the caller drawn for the offer, the caller
// allowed to claim it, and the bucket that gets widened are one and the same by
// construction rather than by agreement.
func (s *ClickService) ClaimBonus(
	ctx context.Context,
	req *connect.Request[planetv1.ClaimBonusRequest],
) (*connect.Response[planetv1.ClaimBonusResponse], error) {
	if s.bonuses == nil || s.booster == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, ErrNoSuchBonus)
	}

	scope := cpconnect.RateLimitKey(ctx)

	reward, ok := s.bonuses.Claim(req.Msg.GetToken(), scope)
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, ErrNoSuchBonus)
	}

	state := s.booster.Boost(scope, s.bonuses.Multiplier(), s.clock.Now().Add(reward.Duration))

	// Only after the boost is applied: a catch announced to the planet that
	// then failed to land would be the one lie this feature could tell.
	s.bonuses.Publish(bonus.Taken{
		CountryID: req.Msg.GetCountryId(),
		Kind:      reward.Kind,
	})

	return connect.NewResponse(&planetv1.ClaimBonusResponse{
		Budget:          EncodeBudget(state),
		Kind:            encodeKind(reward.Kind),
		DurationSeconds: uint32(reward.Duration / time.Second),
	}), nil
}

func encodeKind(kind bonus.Kind) planetv1.BonusKind {
	switch kind {
	case bonus.KindTripleClicks:
		return planetv1.BonusKind_BONUS_KIND_TRIPLE_CLICKS
	default:
		return planetv1.BonusKind_BONUS_KIND_UNSPECIFIED
	}
}

func bonusOfferedEvent(offer *bonus.Offer) *planetv1.PlanetEvent {
	return &planetv1.PlanetEvent{
		Event: &planetv1.PlanetEvent_BonusOffered{
			BonusOffered: &planetv1.BonusOffered{
				Token:           offer.Token,
				Seed:            offer.Seed,
				Kind:            encodeKind(offer.Kind),
				DurationSeconds: uint32(offer.Duration / time.Second),
				ExpiresAtUnixMs: offer.ExpiresAt.UnixMilli(),
			},
		},
	}
}

func bonusTakenEvent(taken *bonus.Taken) *planetv1.PlanetEvent {
	return &planetv1.PlanetEvent{
		Event: &planetv1.PlanetEvent_BonusTaken{
			BonusTaken: &planetv1.BonusTaken{
				CountryId: taken.CountryID,
				Kind:      encodeKind(taken.Kind),
			},
		},
	}
}

// bonusEvent turns whichever half of the event is set into a frame, and returns
// nil for one that is neither — a case added to bonus.Event and forgotten here
// drops rather than panics.
func bonusEvent(event bonus.Event) *planetv1.PlanetEvent {
	switch {
	case event.Offer != nil:
		return bonusOfferedEvent(event.Offer)
	case event.Taken != nil:
		return bonusTakenEvent(event.Taken)
	default:
		return nil
	}
}

// Compile-time proof that the registry satisfies the port the controller
// declares, without the controller importing it for anything else.
var _ BonusRegistry = (*bonus.Registry)(nil)
