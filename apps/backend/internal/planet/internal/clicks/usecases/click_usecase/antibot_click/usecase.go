// Package antibot_click is the shadow ban, as a decorator over the click use
// case rather than an interceptor over the procedure.
//
// It sits here and not at the edge because none of what it does is about HTTP:
// it reads who held the tile before the write, it reports the take afterwards,
// and a flagged caller is answered OK with nothing written. All three are about
// the click, and the middle one only works adjacent to the write — afterwards
// the map no longer remembers who held the tile.
package antibot_click

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpipscope"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

// ClickGuard is the part of antibot.Guard this decorator uses.
type ClickGuard interface {
	Inspect(click antibot.Click) (drop bool)
	Committed(click antibot.Click)
	Flagged() int
}

// TileOwner reads who holds a tile. The jury is handed that answer from before
// the write, because afterwards the map no longer remembers it.
type TileOwner interface {
	Owner(tile uint32) (string, bool)
}

func New(
	implementation click_usecase.IUseCase,
	guard ClickGuard,
	owner TileOwner,
	clock cptime.Clock,
	registerer prometheus.Registerer,
) *UseCase {
	if clock == nil {
		clock = cptime.SystemClock{}
	}

	factory := promauto.With(registerer)

	dropped := factory.NewCounter(prometheus.CounterOpts{
		Name: "shadowbanned_clicks",
		Help: "Clicks answered OK and dropped without touching the map",
	})

	factory.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "shadowban_flagged",
		Help: "Callers currently banned, whether or not shadowBan.enforce is on",
	}, func() float64 { return float64(guard.Flagged()) })

	return &UseCase{
		implementation: implementation,
		guard:          guard,
		owner:          owner,
		clock:          clock,
		dropped:        dropped,
	}
}

type UseCase struct {
	implementation click_usecase.IUseCase
	guard          ClickGuard
	owner          TileOwner
	clock          cptime.Clock
	dropped        prometheus.Counter
}

// Execute answers a flagged caller exactly as it answers an accepted one: nil.
// That is the whole point — a refusal names the check that tripped and the
// author fixes it in an afternoon, while a silent no-op names nothing.
func (u *UseCase) Execute(ctx context.Context, in click_usecase.In) (click_usecase.Out, error) {
	// Scoped to the same unit the throttle is charged to, so a caller cannot
	// serve a ban on one address and click from the next one in its own /64.
	// The account rides along so a ban falls on it too, and a linked one is signed in, so its ban leaves the scope alone.
	observed := antibot.Click{
		Scope:    cpipscope.Of(cpctx.GetSourceIP(ctx)),
		Account:  cpctx.GetAccount(ctx),
		SignedIn: cpctx.GetLinked(ctx),
		Tile:     in.TileID,
		Country:  in.CountryID,
		At:       u.clock.Now(),
	}

	if held, known := u.owner.Owner(observed.Tile); known {
		observed.Held = held
		observed.NoOp = held == observed.Country
	}

	if u.guard.Inspect(observed) {
		u.dropped.Inc()
		return click_usecase.Out{}, nil
	}

	out, err := u.implementation.Execute(ctx, in)

	// Only a click the use case accepted actually reached the map. A refused one
	// recorded as a take is a way to have the next honest clicker of that tile
	// look like it is reacting to something.
	if err == nil {
		u.guard.Committed(observed)
	}

	return out, err
}
