package planetv1controller

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/prometheus/client_golang/prometheus"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/ctxutil"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/ipscope"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/logging"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/logging/lf"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/xtime"
)

// This one cannot live in connectutil like the other three: it reads tile_id
// and country_id out of the message, so it is tied to this contract.
type ClickJury interface {
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
	jury ClickJury,
	owner TileOwner,
	timeProvider xtime.Provider,
	registerer prometheus.Registerer,
) (connect.Interceptor, error) {
	if timeProvider == nil {
		timeProvider = xtime.ActualProvider{}
	}

	dropped := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "shadowbanned_clicks",
		Help: "Clicks answered OK and dropped without touching the map",
	})

	flagged := prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "shadowban_flagged",
		Help: "Callers currently banned, whether or not shadowBan.enforce is on",
	}, func() float64 { return float64(jury.Flagged()) })

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
				Scope:   ipscope.Of(ctxutil.GetSourceIP(ctx)),
				Tile:    msg.GetTileId(),
				Country: msg.GetCountryId(),
				At:      timeProvider.Now(),
			}

			if held, known := owner.Owner(click.Tile); known {
				click.Held = held
				click.NoOp = held == click.Country
			}

			if jury.Inspect(click) {
				dropped.Inc()
				return connect.NewResponse(&planetv1.ClickResponse{}), nil
			}

			res, err := next(ctx, req)

			// Only a click the handler accepted actually reached the map. A
			// refused one recorded as a take is a way to have the next honest
			// clicker of that tile look like it is reacting to something.
			if err == nil {
				jury.Committed(click)
			}

			return res, err
		}
	}), nil
}

// NewAntiBotReporter builds the hooks the jury reports through: a histogram for
// the shape of the reactions, and a log line for who.
func NewAntiBotReporter(
	logger logging.Logger,
	registerer prometheus.Registerer,
) (func(time.Duration), func(antibot.Report), error) {
	if logger == nil {
		logger = logging.NewNopLogger()
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
			return nil, nil, fmt.Errorf("failed to register collector: %w", err)
		}
	}

	onReaction := func(delay time.Duration) { reactions.Observe(delay.Seconds()) }

	// The address goes in the log and never on a label: per-IP labels are
	// unbounded cardinality, and they would put personal data in every scrape.
	onFlag := func(report antibot.Report) {
		fields := []lf.Field{
			lf.String("scope", report.Scope),
			lf.Int("flags", report.Flags),
			lf.Int("clicks", report.Clicks),
			lf.Any("activeFor", report.ActiveFor),
			lf.Any("longestGap", report.LongestGap),
			lf.String("topCountry", report.TopCountry),
			lf.Int("topCountryClicks", report.TopCountryClicks),
			lf.Any("tiles", report.Tiles),
		}

		// Every watchdog goes in the line, the quiet ones included: what did not
		// fire is half of reading a ban that did.
		for _, opinion := range report.Opinions {
			fields = append(fields, lf.String(opinion.Watchdog, formatOpinion(opinion)))

			if opinion.Verdict != antibot.Clear {
				flags.WithLabelValues(opinion.Watchdog).Inc()
			}
		}

		logger.Warning("antibot ban", fields...)
	}

	return onReaction, onFlag, nil
}

func formatOpinion(opinion antibot.Opinion) string {
	if opinion.Verdict == antibot.Clear {
		return antibot.Clear.String()
	}

	parts := make([]string, 0, len(opinion.Evidence.Fields)+1)
	parts = append(parts, fmt.Sprintf("%s %s", opinion.Verdict, opinion.Evidence.Rule))

	fields := opinion.Evidence.Fields
	sort.SliceStable(fields, func(i, j int) bool { return fields[i].Key < fields[j].Key })

	for _, field := range fields {
		parts = append(parts, fmt.Sprintf("%s=%v", field.Key, field.Value))
	}

	return strings.Join(parts, " ")
}
