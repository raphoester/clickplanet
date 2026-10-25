package antibot_click

import (
	"context"
	"math"
	"time"

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
	Inspect(ctx context.Context, click antibot.Click)
	Committed(click antibot.Click)
	Flagged(ctx context.Context) (int, error)
}

const flaggedTimeout = time.Second

type TileOwner interface {
	Owner(tile uint32) (string, bool)
}

type Rule interface {
	Outcome(tile uint32, owner, flag string) clicks.Outcome
}

type Tempo interface {
	Tempo() float64
}

func New(
	implementation click_usecase.IUseCase,
	guard ClickGuard,
	owner TileOwner,
	rule Rule,
	tempo Tempo,
	clock cptime.Clock,
	registerer prometheus.Registerer,
) *UseCase {
	if clock == nil {
		clock = cptime.SystemClock{}
	}

	promauto.With(registerer).NewGaugeFunc(prometheus.GaugeOpts{
		Name: "banned_callers",
		Help: "Callers currently banned, whether or not bans.enforce is on",
	}, func() float64 {
		ctx, cancel := context.WithTimeout(context.Background(), flaggedTimeout)
		defer cancel()

		flagged, err := guard.Flagged(ctx)
		if err != nil {
			return math.NaN()
		}
		return float64(flagged)
	})

	return &UseCase{
		implementation: implementation,
		guard:          guard,
		owner:          owner,
		rule:           rule,
		tempo:          tempo,
		clock:          clock,
	}
}

type UseCase struct {
	implementation click_usecase.IUseCase
	guard          ClickGuard
	owner          TileOwner
	rule           Rule
	tempo          Tempo
	clock          cptime.Clock
}

func (u *UseCase) Execute(ctx context.Context, in click_usecase.In) (click_usecase.Out, error) {
	observed := antibot.Click{
		Scope:    cpipscope.Of(cpctx.GetSourceIP(ctx)),
		Account:  cpctx.GetAccount(ctx),
		SignedIn: cpctx.GetLinked(ctx),
		Tile:     in.TileID,
		Country:  in.CountryID,
		At:       u.clock.Now(),
		Tempo:    u.tempo.Tempo(),
	}

	if held, known := u.owner.Owner(observed.Tile); known {
		outcome := u.rule.Outcome(observed.Tile, held, observed.Country)
		observed.Held = held
		observed.NoOp = outcome == clicks.Unchanged
		observed.Shielded = outcome == clicks.Shielded
	}

	u.guard.Inspect(ctx, observed)

	out, err := u.implementation.Execute(ctx, in)

	if err == nil {
		u.guard.Committed(observed)
	}

	return out, err
}
