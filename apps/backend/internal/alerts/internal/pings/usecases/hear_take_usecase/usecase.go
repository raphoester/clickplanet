package hear_take_usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/nations"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/pings"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/roles"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Store interface {
	AttackPingsOf(ctx context.Context, victim nations.Flag, since time.Time) (pings.History, error)
	SaveAttackPing(ctx context.Context, victim nations.Flag, at time.Time) error
}

type Herald interface {
	CallToDefend(ctx context.Context, attack pings.Attack) error
}

const timeout = 10 * time.Second

func New(watch *pings.Watch, clock cptime.Clock, store Store, herald Herald, transactioner cpatomic.Transactioner) *UseCase {
	return &UseCase{watch: watch, clock: clock, store: store, herald: herald, transactioner: transactioner}
}

type UseCase struct {
	watch         *pings.Watch
	clock         cptime.Clock
	store         Store
	herald        Herald
	transactioner cpatomic.Transactioner
}

// The takes of a batch, heard in order: each victim is pinged once at most, for the first attack the batch made on it,
// as the quiet time would have it. A ping Discord refused is not tried again: the watch already counted the attack.
func (u *UseCase) Execute(ctx context.Context, takes ...pings.Take) error {
	var attacks []pings.Attack
	attacked := cpcolls.NewSet[nations.Flag]()
	for _, take := range takes {
		attack, heard := u.watch.Hear(take)
		if !heard || attacked.Contains(attack.Victim) {
			continue
		}
		attacked.Add(attack.Victim)
		attacks = append(attacks, attack)
	}

	for _, attack := range attacks {
		if err := u.callToDefend(ctx, attack); err != nil {
			return err
		}
	}
	return nil
}

func (u *UseCase) callToDefend(ctx context.Context, attack pings.Attack) error {
	now := u.clock.Now()
	if attack.Stale(now) {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	history, err := u.store.AttackPingsOf(ctx, attack.Victim, now.Add(-pings.Day))
	if err != nil {
		return fmt.Errorf("failed to read the pings of %s: %w", attack.Victim, err)
	}
	if !history.Allows(now) {
		return nil
	}

	err = u.transactioner.InTx(ctx, func(ctx context.Context) error {
		if err := u.store.SaveAttackPing(ctx, attack.Victim, now); err != nil {
			return fmt.Errorf("failed to keep the ping: %w", err)
		}
		return u.herald.CallToDefend(ctx, attack)
	})
	if errors.Is(err, roles.ErrNoRole) {
		return nil
	}
	return err //nolint:wrapcheck // each step named its own.
}
