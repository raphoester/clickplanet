package set_raid_mode_usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/moderation"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/raids"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Store interface {
	Current(ctx context.Context) (raids.RaidMode, error)
	Save(ctx context.Context, mode raids.RaidMode) error
}

type In struct {
	// 0 ends the raid mode that runs.
	Duration time.Duration
	Reason   string
	By       moderation.Operator
}

type Executor interface {
	// The raid mode that runs once the call is done, if one does.
	Execute(ctx context.Context, in In) (raids.RaidMode, bool, error)
}

const timeout = 5 * time.Second

func New(store Store, clock cptime.Clock) *UseCase {
	return &UseCase{store: store, clock: clock}
}

type UseCase struct {
	store Store
	clock cptime.Clock
}

var _ Executor = (*UseCase)(nil)

func (u *UseCase) Execute(ctx context.Context, in In) (raids.RaidMode, bool, error) {
	duration, err := raids.DurationOf(in.Duration)
	if err != nil {
		return raids.RaidMode{}, false, err //nolint:wrapcheck // the domain named it.
	}
	reason, err := moderation.ReasonOf(in.Reason)
	if err != nil {
		return raids.RaidMode{}, false, err //nolint:wrapcheck // the domain named it.
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	now := u.clock.Now()
	if duration == 0 {
		return raids.RaidMode{}, false, u.end(ctx, now)
	}

	mode, err := raids.Start(now, duration, reason)
	if err != nil {
		return raids.RaidMode{}, false, fmt.Errorf("failed to start raid mode: %w", err)
	}
	if err := u.store.Save(ctx, mode); err != nil {
		return raids.RaidMode{}, false, fmt.Errorf("failed to keep the raid mode: %w", err)
	}
	return mode, true, nil
}

// One that does not run is left: ending nothing changes nothing.
func (u *UseCase) end(ctx context.Context, now time.Time) error {
	current, err := u.store.Current(ctx)
	if errors.Is(err, raids.ErrNoRaidMode) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to read the raid mode: %w", err)
	}
	if !current.RunsAt(now) {
		return nil
	}
	if err := u.store.Save(ctx, current.Ended(now)); err != nil {
		return fmt.Errorf("failed to end the raid mode: %w", err)
	}
	return nil
}
