package planet

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click/antibot_click"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

// Even resolution to 2s: a bot on a ~1s timer hides in a bucket any wider, which is where 0.5/1/2 left it invisible.
var reactionBuckets = []float64{
	0.05, 0.1, 0.2, 0.3, 0.4,
	0.5, 0.6, 0.7, 0.8, 0.9,
	1.0, 1.1, 1.25, 1.5, 1.75,
	2.0, 2.5, 3.0, 4.0, 5.0,
}

// wrapWithAntiBot puts the shadow ban around the click use case, and returns the
// use case unchanged when the block is off. Nothing about it reaches the edge
// any more: what is left of the clicks side is the metric names and the words of
// the ban line, both of them below.
func wrapWithAntiBot(
	useCase click.IUseCase,
	config antibot.Config,
	owner antibot_click.TileOwner,
	props cpbootstrap.Props,
) (click.IUseCase, error) {
	clock := cptime.SystemClock{}

	observer, err := newAntiBotObserver(props.Logger, props.Metrics)
	if err != nil {
		return nil, fmt.Errorf("failed to create the antibot observer: %w", err)
	}

	guard, err := antibot.New(config, clock, observer)
	if err != nil {
		return nil, fmt.Errorf("failed to build the antibot guard: %w", err)
	}
	if guard == nil {
		return useCase, nil
	}

	props.Runners.Add("antibot", guard.Run)

	described := guard.Describe()
	props.Logger.Info("antibot enabled",
		slog.Any("watchdogs", described.Watchdogs),
		slog.Int("minSuspects", described.MinSuspects),
		slog.Bool("enforce", described.Enforcing),
	)

	wrapped, err := antibot_click.New(useCase, guard, owner, clock, props.Metrics)
	if err != nil {
		return nil, fmt.Errorf("failed to create the antibot click use case: %w", err)
	}

	return wrapped, nil
}

// newAntiBotObserver builds the hooks the guard reports through: a histogram for
// the shape of the reactions, and a log line for who.
func newAntiBotObserver(
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
