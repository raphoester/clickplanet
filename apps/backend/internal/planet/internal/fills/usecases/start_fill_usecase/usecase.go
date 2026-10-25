package start_fill_usecase

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/fills"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Fills interface {
	Fill(ctx context.Context, request bonuses.FillID) (fills.Fill, error)
	Save(ctx context.Context, fill fills.Fill) error
}

type Ledger interface {
	Flush(ctx context.Context) error
}

type Runner interface {
	Wake()
}

type In struct {
	Request bonuses.FillID
	Source  fills.Source
}

func New(kept Fills, ledger Ledger, runner Runner, clock cptime.Clock) *UseCase {
	return &UseCase{fills: kept, ledger: ledger, runner: runner, clock: clock}
}

type UseCase struct {
	fills  Fills
	ledger Ledger
	runner Runner
	clock  cptime.Clock

	// One at a time, so a request sent twice at once starts one fill.
	mu sync.Mutex
}

func (u *UseCase) Execute(ctx context.Context, in In) error {
	u.mu.Lock()
	defer u.mu.Unlock()

	kept, err := u.fills.Fill(ctx, in.Request)
	if err != nil {
		return fmt.Errorf("could not tell whether request %q started a fill: %w", in.Request, err)
	}

	fill, err := fills.NewFill(in.Request, in.Source, u.clock.Now(), kept)
	if errors.Is(err, fills.ErrStartedAlready) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("cannot start a fill for request %q: %w", in.Request, err)
	}

	// The fill reads who played from postgres, which holds the ledger's last second only once flushed.
	if err := u.ledger.Flush(ctx); err != nil {
		return fmt.Errorf("failed to flush the ledger before the fill %q: %w", in.Request, err)
	}
	if err := u.fills.Save(ctx, fill); err != nil {
		return fmt.Errorf("could not keep the fill %q: %w", in.Request, err)
	}
	u.runner.Wake()

	return nil
}
