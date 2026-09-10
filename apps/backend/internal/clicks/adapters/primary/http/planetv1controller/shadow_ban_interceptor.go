package planetv1controller

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/prometheus/client_golang/prometheus"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/ctxutil"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/ipscope"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/logging"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/logging/lf"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/shadowban"
)

// This one cannot live in connectutil like the other three: it reads tile_id
// and country_id out of the message, so it is tied to this contract.
type ClickShadowBanner interface {
	Observe(scope string, tile uint32, country string) (drop bool, takes bool)
	Took(scope string, tile uint32)
	Flagged() int
}

// Even resolution to 2s: a bot on a ~1s timer hides in a bucket any wider, which is where 0.5/1/2 left it invisible.
var reactionBuckets = []float64{
	0.05, 0.1, 0.2, 0.3, 0.4,
	0.5, 0.6, 0.7, 0.8, 0.9,
	1.0, 1.1, 1.25, 1.5, 1.75,
	2.0, 2.5, 3.0, 4.0, 5.0,
}

func NewShadowBanInterceptor(
	detector ClickShadowBanner,
	registerer prometheus.Registerer,
) (connect.Interceptor, error) {
	dropped := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "shadowbanned_clicks",
		Help: "Clicks answered OK and dropped without touching the map",
	})

	flagged := prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "shadowban_flagged",
		Help: "Callers currently flagged, whether or not shadowBan.enforce is on",
	}, func() float64 { return float64(detector.Flagged()) })

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

			// The same unit the throttle is charged to, so a caller cannot serve
			// a ban on one address and click from the next one in its own /64.
			scope := ipscope.Of(ctxutil.GetSourceIP(ctx))

			drop, takes := detector.Observe(scope, msg.GetTileId(), msg.GetCountryId())
			if drop {
				dropped.Inc()
				return connect.NewResponse(&planetv1.ClickResponse{}), nil
			}

			res, err := next(ctx, req)

			// Only a click the handler accepted actually took the tile. A
			// refused one recorded as a take is a way to have the next honest
			// clicker of that tile look like it is reacting to something.
			if err == nil && takes {
				detector.Took(scope, msg.GetTileId())
			}

			return res, err
		}
	}), nil
}

// NewShadowBanReporter builds the hooks the detector reports through: a
// histogram for the shape of the reactions, and a log line for who.
func NewShadowBanReporter(
	logger logging.Logger,
	registerer prometheus.Registerer,
) (func(time.Duration), func(string, shadowban.Report), error) {
	if logger == nil {
		logger = logging.NewNopLogger()
	}

	reactions := prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "click_reaction_seconds",
		Help:    "Delay between a tile being taken and another caller taking it back",
		Buckets: reactionBuckets,
	})

	// Counts flags, not callers: one caller flagged six times is six here and one
	// on shadowban_flagged, and the gap between the two is the thing to look at.
	flags := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "shadowban_flags",
		Help: "Times a caller has been flagged, counting repeat flags on the same caller",
	})

	for _, collector := range []prometheus.Collector{reactions, flags} {
		if err := registerer.Register(collector); err != nil {
			return nil, nil, fmt.Errorf("failed to register collector: %w", err)
		}
	}

	onReaction := func(delay time.Duration) { reactions.Observe(delay.Seconds()) }

	// The address goes in the log and never on a label: per-IP labels are
	// unbounded cardinality, and they would put personal data in every scrape.
	onFlag := func(scope string, report shadowban.Report) {
		flags.Inc()

		logger.Warning("shadowban candidate",
			lf.String("scope", scope),
			lf.Int("flags", report.Flags),
			lf.Int("reactions", report.Reactions),
			lf.Any("median", report.Median),
			lf.Any("spread", report.Spread),
			lf.Any("activeFor", report.ActiveFor),
			lf.Any("longestGap", report.LongestGap),
			lf.String("topCountry", report.TopCountry),
			lf.Int("topCountryClicks", report.TopCountryClicks),
			lf.Int("clicks", report.Clicks),
			lf.Any("tiles", report.Tiles),
		)
	}

	return onReaction, onFlag, nil
}
