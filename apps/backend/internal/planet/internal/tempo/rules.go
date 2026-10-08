package tempo

import (
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

var (
	ErrFrozen = errors.New("the map is frozen")

	ErrInvalidRules = errors.New("rules nobody can play by")
)

type GiftTag string

type Gift struct {
	tag        GiftTag
	madeBefore time.Time
}

func GiftOf(tag GiftTag, madeBefore time.Time) (Gift, error) {
	if tag == "" {
		return Gift{}, fmt.Errorf("%w: a gift has no tag", ErrInvalidRules)
	}

	return Gift{tag: tag, madeBefore: madeBefore}, nil
}

func (g Gift) Tag() GiftTag { return g.tag }

func (g Gift) MadeBefore() time.Time { return g.madeBefore }

// An account that does not say when it was made is older than the ids that do.
func (g Gift) OwedTo(payer clicks.Payer) bool {
	if payer.Account == "" {
		return false
	}

	return g.madeBefore.IsZero() || payer.Created.IsZero() || payer.Created.Before(g.madeBefore)
}

type Rules struct {
	refillMultiplier float64
	boxInterval      time.Duration
	gift             Gift
	frozen           bool
}

func Plain() Rules {
	return Rules{refillMultiplier: 1}
}

func NewRules(refillMultiplier float64, boxInterval time.Duration, frozen bool) (Rules, error) {
	if refillMultiplier == 0 {
		refillMultiplier = 1
	}
	if math.IsNaN(refillMultiplier) || math.IsInf(refillMultiplier, 0) || refillMultiplier < 1 {
		return Rules{}, fmt.Errorf("%w: a refill multiplier of %v", ErrInvalidRules, refillMultiplier)
	}
	if boxInterval < 0 {
		return Rules{}, fmt.Errorf("%w: a box every %s", ErrInvalidRules, boxInterval)
	}

	return Rules{refillMultiplier: refillMultiplier, boxInterval: boxInterval, frozen: frozen}, nil
}

func (r Rules) WithGift(gift Gift) Rules {
	r.gift = gift
	return r
}

func (r Rules) RefillMultiplier() float64 { return max(r.refillMultiplier, 1) }

func (r Rules) BoxInterval() (time.Duration, bool) { return r.boxInterval, r.boxInterval > 0 }

func (r Rules) Gift() (Gift, bool) { return r.gift, r.gift.tag != "" }

func (r Rules) Frozen() bool { return r.frozen }

func (r Rules) FrozenError() error {
	if r.frozen {
		return ErrFrozen
	}
	return nil
}
