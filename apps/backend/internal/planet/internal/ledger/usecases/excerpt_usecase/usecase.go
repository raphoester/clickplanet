package excerpt_usecase

import (
	"context"
	"errors"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

// The ledger keeps 72 hours in memory.
const longest = 72 * time.Hour

var ErrInvalidWindow = errors.New("an excerpt needs a since before its until, at most 72 hours apart")

type Ledger interface {
	Replay(see func(ledger.Event)) ledger.Position
}

type In struct {
	Since time.Time
	Until time.Time
}

type Out struct {
	Lines []ledger.Line
}

func New(ledger Ledger, clock cptime.Clock) *UseCase {
	return &UseCase{ledger: ledger, clock: clock}
}

type UseCase struct {
	ledger Ledger
	clock  cptime.Clock
}

func (u *UseCase) Execute(_ context.Context, in In) (Out, error) {
	until := in.Until
	if until.IsZero() {
		until = u.clock.Now()
	}
	if in.Since.IsZero() || !in.Since.Before(until) || until.Sub(in.Since) > longest {
		return Out{}, ErrInvalidWindow
	}

	excerpt := ledger.NewExcerpt(in.Since, until)
	u.ledger.Replay(excerpt.See)

	return Out{Lines: excerpt.Lines()}, nil
}
