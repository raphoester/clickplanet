package rallies

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/allegiances"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Countries interface {
	CheckCountry(country string) bool
}

type Allegiances interface {
	PlayError(ctx context.Context, payer clicks.Payer, flag string) error
}

type Refusals interface {
	Refusal(ctx context.Context, payer clicks.Payer) (clicks.Refusal, bool, error)
}

type Calls interface {
	Last(ctx context.Context, account cpsession.AccountID) (Rally, error)
}

func NewGate(countries Countries, allegiances Allegiances, refusals Refusals, calls Calls, clock cptime.Clock) Gate {
	return Gate{countries: countries, allegiances: allegiances, refusals: refusals, calls: calls, clock: clock}
}

// Gate refuses a rally before it leaves the hand: a banned caller, a flag it does not play for, and a rally of the caller
// still on. Any point of the globe takes one, land or sea, whoever holds the tiles around it. The call checks the last
// again under its lock.
type Gate struct {
	countries   Countries
	allegiances Allegiances
	refusals    Refusals
	calls       Calls
	clock       cptime.Clock
}

// A failed read of the refusals, of the flag or of the last rally refuses nothing, as for a click.
func (g Gate) Call(ctx context.Context, payer clicks.Payer, account cpsession.AccountID, country string) error {
	if refusal, refused, err := g.refusals.Refusal(ctx, payer); err == nil && refused {
		return refusal
	}
	if !g.countries.CheckCountry(country) {
		return fmt.Errorf("%w: %q", clicks.ErrUnknownCountry, country)
	}
	if err := g.allegiances.PlayError(ctx, payer, country); err != nil && !errors.Is(err, allegiances.ErrUnread) {
		return err //nolint:wrapcheck // the book names the flag the caller plays for.
	}
	if last, err := g.calls.Last(ctx, account); err == nil && last.On(g.clock.Now()) {
		return ErrRallyOn
	}

	return nil
}
