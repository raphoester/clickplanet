package antibot_click

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpipscope"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type ClickGuard interface {
	Inspect(click antibot.Click) (drop bool)
	Committed(click antibot.Click)
	Flagged() int
}

type TileOwner interface {
	Owner(tile uint32) (string, bool)
}

type Rule interface {
	Outcome(tile uint32, owner, flag string) clicks.Outcome
}

func New(
	implementation click_usecase.IUseCase,
	guard ClickGuard,
	owner TileOwner,
	rule Rule,
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
		rule:           rule,
		clock:          clock,
		dropped:        dropped,
	}
}

type UseCase struct {
	implementation click_usecase.IUseCase
	guard          ClickGuard
	owner          TileOwner
	rule           Rule
	clock          cptime.Clock
	dropped        prometheus.Counter
}

func (u *UseCase) Execute(ctx context.Context, in click_usecase.In) (click_usecase.Out, error) {
	observed := antibot.Click{
		Scope:    cpipscope.Of(cpctx.GetSourceIP(ctx)),
		Account:  cpctx.GetAccount(ctx),
		SignedIn: cpctx.GetLinked(ctx),
		Tile:     in.TileID,
		Country:  in.CountryID,
		At:       u.clock.Now(),
	}

	if held, known := u.owner.Owner(observed.Tile); known {
		outcome := u.rule.Outcome(observed.Tile, held, observed.Country)
		observed.Held = held
		observed.NoOp = outcome == clicks.Unchanged
		observed.Shielded = outcome == clicks.Shielded
	}

	if u.guard.Inspect(observed) {
		// Answer like an accepted click: a refusal would tell the bot it was caught.
		u.dropped.Inc()
		return click_usecase.Out{}, nil
	}

	out, err := u.implementation.Execute(ctx, in)

	if err == nil {
		u.guard.Committed(observed)
	}

	return out, err
}
