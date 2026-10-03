// Package forget_allegiances_usecase deletes the tallies with no take since they faded.
package forget_allegiances_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Allegiances interface {
	DeleteAllegiancesBefore(ctx context.Context, cutoff time.Time) (int64, error)
}

// Executor is the forget, as the runner calls it and a decorator wraps it.
type Executor interface {
	Execute(ctx context.Context) (int64, error)
}

const timeout = 30 * time.Second

func New(clock cptime.Clock, allegiances Allegiances) *UseCase {
	return &UseCase{clock: clock, allegiances: allegiances}
}

type UseCase struct {
	clock       cptime.Clock
	allegiances Allegiances
}

var _ Executor = (*UseCase)(nil)

// Execute says how many tallies went.
func (u *UseCase) Execute(ctx context.Context) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	deleted, err := u.allegiances.DeleteAllegiancesBefore(ctx, clicks.FadedBefore(u.clock.Now()))
	if err != nil {
		return deleted, fmt.Errorf("failed to forget the faded allegiances: %w", err)
	}

	return deleted, nil
}
