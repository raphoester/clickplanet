package announce_start_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/pings"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Store interface {
	StartPinged(ctx context.Context, key pings.StartKey) (bool, error)
	SaveStartPing(ctx context.Context, key pings.StartKey, at time.Time) error
}

type Herald interface {
	CallToBattle(ctx context.Context, start pings.Start) error
}

const timeout = 10 * time.Second

func New(clock cptime.Clock, store Store, herald Herald, transactioner cpatomic.Transactioner) *UseCase {
	return &UseCase{clock: clock, store: store, herald: herald, transactioner: transactioner}
}

type UseCase struct {
	clock         cptime.Clock
	store         Store
	herald        Herald
	transactioner cpatomic.Transactioner
}

// Told once: a post Discord refused is kept for nobody, and the event comes again until it is Fresh no more.
func (u *UseCase) Execute(ctx context.Context, start pings.Start) error {
	now := u.clock.Now()
	if start.Stale(now) {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	pinged, err := u.store.StartPinged(ctx, start.Key)
	if err != nil {
		return fmt.Errorf("failed to read whether %s was told: %w", start.Key, err)
	}
	if pinged {
		return nil
	}
	return u.transactioner.InTx(ctx, func(ctx context.Context) error { //nolint:wrapcheck // each step named its own.
		if err := u.store.SaveStartPing(ctx, start.Key, now); err != nil {
			return fmt.Errorf("failed to keep the ping: %w", err)
		}
		return u.herald.CallToBattle(ctx, start)
	})
}
