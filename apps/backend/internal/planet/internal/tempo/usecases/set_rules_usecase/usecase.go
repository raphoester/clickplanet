package set_rules_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/tempo"
)

type Switches interface {
	Set(rules tempo.Rules)
}

type Executor interface {
	Execute(ctx context.Context, in In) error
}

type In struct {
	RefillMultiplier float64
	BoxInterval      time.Duration

	GiftTag        tempo.GiftTag
	GiftMadeBefore time.Time

	Frozen bool
}

func New(switches Switches) *UseCase {
	return &UseCase{switches: switches}
}

type UseCase struct {
	switches Switches
}

var _ Executor = (*UseCase)(nil)

func (u *UseCase) Execute(_ context.Context, in In) error {
	rules, err := tempo.NewRules(in.RefillMultiplier, in.BoxInterval, in.Frozen)
	if err != nil {
		return fmt.Errorf("cannot set the rules: %w", err)
	}

	if in.GiftTag != "" {
		gift, err := tempo.GiftOf(in.GiftTag, in.GiftMadeBefore)
		if err != nil {
			return fmt.Errorf("cannot set the rules: %w", err)
		}
		rules = rules.WithGift(gift)
	}

	u.switches.Set(rules)

	return nil
}
