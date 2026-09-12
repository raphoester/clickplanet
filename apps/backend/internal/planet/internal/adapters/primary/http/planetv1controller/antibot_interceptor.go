package planetv1controller

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"connectrpc.com/connect"
	"github.com/prometheus/client_golang/prometheus"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpipscope"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

// This one cannot live in cpconnect like the other three: it reads tile_id
// and country_id out of the message, so it is tied to this contract.
type ClickGuard interface {
	Inspect(click antibot.Click) (drop bool)
	Committed(click antibot.Click)
	Flagged() int
}

// TileOwner reads who holds a tile. The jury is handed that answer from before
// the handler runs, because afterwards the map no longer remembers it.
type TileOwner interface {
	Owner(tile uint32) (string, bool)
}

// Even resolution to 2s: a bot on a ~1s timer hides in a bucket any wider, which is where 0.5/1/2 left it invisible.
var reactionBuckets = []float64{
	0.05, 0.1, 0.2, 0.3, 0.4,
	0.5, 0.6, 0.7, 0.8, 0.9,
	1.0, 1.1, 1.25, 1.5, 1.75,
	2.0, 2.5, 3.0, 4.0, 5.0,
}

func NewAntiBotInterceptor(
	guard ClickGuard,
	owner TileOwner,
	clock cptime.Clock,
	registerer prometheus.Registerer,
) (connect.Interceptor, error) {
	if clock == nil {
		clock = cptime.SystemClock{}
	}

	dropped := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "shadowbanned_clicks",
		Help: "Clicks answered OK and dropped without touching the map",
	})

	flagged := prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "shadowban_flagged",
		Help: "Callers currently banned, whether or not shadowBan.enforce is on",
	}, func() float64 { return float64(guard.Flagged()) })

	for _, collector := range []prometheus.Collector{dropped, flagged} {
		if err := registerer.Register(collector); err != nil {
			return nil, fmt.Errorf("failed to register collector: %w", err)
		}
	}

	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if req.Spec().Procedure != planetv1connect.ClickServiceClickProcedure {
				return next(ctx, req)
			}

			msg, ok := req.Any().(*planetv1.ClickRequest)
			if !ok {
				return next(ctx, req)
			}

			// Scoped to the same unit the throttle is charged to, so a caller
			// cannot serve a ban on one address and click from the next one in
			// its own /64.
			click := antibot.Click{
				Scope:   cpipscope.Of(cpctx.GetSourceIP(ctx)),
				Tile:    msg.GetTileId(),
				Country: msg.GetCountryId(),
				At:      clock.Now(),
			}

			if held, known := owner.Owner(click.Tile); known {
				click.Held = held
				click.NoOp = held == click.Country
			}

			if guard.Inspect(click) {
				dropped.Inc()
				return connect.NewResponse(&planetv1.ClickResponse{}), nil
			}

			res, err := next(ctx, req)

			// Only a click the handler accepted actually reached the map. A
			// refused one recorded as a take is a way to have the next honest
			// clicker of that tile look like it is reacting to something.
			if err == nil {
				guard.Committed(click)
			}

			return res, err
		}
	}), nil
}

// NewAntiBotObserver builds the hooks the guard reports through: a histogram for
// the shape of the reactions, and a log line for who.
func NewAntiBotObserver(
	logger *slog.Logger,
	registerer prometheus.Registerer,
) (antibot.Observer, error) {
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}

	reactions := prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "click_reaction_seconds",
		Help:    "Delay between a tile being taken and another caller taking it back",
		Buckets: reactionBuckets,
	})

	// Counts flags, not callers, and once per watchdog that argued for each one:
	// a caller flagged six times is six here and one on shadowban_flagged, and
	// the gap between the two is the thing to look at.
	flags := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "shadowban_flags",
		Help: "Times a caller has been flagged, counted once per watchdog that argued for it",
	}, []string{"watchdog"})

	for _, collector := range []prometheus.Collector{reactions, flags} {
		if err := registerer.Register(collector); err != nil {
			return antibot.Observer{}, fmt.Errorf("failed to register collector: %w", err)
		}
	}

	onReaction := func(delay time.Duration) { reactions.Observe(delay.Seconds()) }

	// The address goes in the log and never on a label: per-IP labels are
	// unbounded cardinality, and they would put personal data in every scrape.
	onFlag := func(report antibot.Report) {
		fields := make([]any, 0, 8+len(report.Opinions))
		fields = append(fields,
			slog.String("scope", report.Scope),
			slog.Int("flags", report.Flags),
			slog.Int("clicks", report.Clicks),
			slog.Duration("activeFor", report.ActiveFor),
			slog.Duration("longestGap", report.LongestGap),
			slog.String("topCountry", report.TopCountry),
			slog.Int("topCountryClicks", report.TopCountryClicks),
			slog.Any("tiles", report.Tiles),
		)

		// Every watchdog goes in the line, the quiet ones included: what did not
		// fire is half of reading a ban that did.
		// Every watchdog, not only the ones that argued for the ban: what did not
		// fire is half of reading a line that did. How a reading words itself is
		// antibot's; the attribute name and the message are ours.
		for _, opinion := range report.Opinions {
			fields = append(fields, slog.String(opinion.Watchdog, opinion.String()))

			if opinion.Fired() {
				flags.WithLabelValues(opinion.Watchdog).Inc()
			}
		}

		logger.Warn("antibot ban", fields...)
	}

	return antibot.Observer{OnReaction: onReaction, OnFlag: onFlag}, nil
}
