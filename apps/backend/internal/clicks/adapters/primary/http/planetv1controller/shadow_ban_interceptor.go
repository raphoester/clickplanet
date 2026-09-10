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

// Buckets are tight and low, because the whole question is where the bot band
// ends and human reaction time begins — the Prometheus defaults jump .25 to .5,
// straight across the answer.
var reactionBuckets = []float64{0.05, 0.075, 0.1, 0.15, 0.2, 0.3, 0.5, 1, 2}

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

	if err := registerer.Register(reactions); err != nil {
		return nil, nil, fmt.Errorf("failed to register histogram: %w", err)
	}

	onReaction := func(delay time.Duration) { reactions.Observe(delay.Seconds()) }

	// The address goes in the log and never on a label: per-IP labels are
	// unbounded cardinality, and they would put personal data in every scrape.
	onFlag := func(scope string, report shadowban.Report) {
		logger.Warning("shadowban candidate",
			lf.String("scope", scope),
			lf.Int("reactions", report.Reactions),
			lf.Any("median", report.Median),
			lf.Any("spread", report.Spread),
			lf.String("topCountry", report.TopCountry),
			lf.Int("topCountryClicks", report.TopCountryClicks),
			lf.Int("clicks", report.Clicks),
			lf.Any("tiles", report.Tiles),
		)
	}

	return onReaction, onFlag, nil
}
