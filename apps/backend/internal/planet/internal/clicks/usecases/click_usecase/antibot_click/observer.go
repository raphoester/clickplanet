package antibot_click

import (
	"log/slog"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot"
)

// NewObserver is how the guard's findings reach the log and the registry: the
// metric names and the words of the ban line are the clicks side's, not antibot's.
func NewObserver(logger *slog.Logger, registerer prometheus.Registerer) antibot.Observer {
	factory := promauto.With(registerer)

	// Even resolution to 2s: a bot on a ~1s timer hides in a bucket any wider, which is where 0.5/1/2 left it invisible.
	reactions := factory.NewHistogram(prometheus.HistogramOpts{
		Name: "click_reaction_seconds",
		Help: "Delay between a tile being taken and another caller taking it back",
		Buckets: []float64{
			0.05, 0.1, 0.2, 0.3, 0.4,
			0.5, 0.6, 0.7, 0.8, 0.9,
			1.0, 1.1, 1.25, 1.5, 1.75,
			2.0, 2.5, 3.0, 4.0, 5.0,
		},
	})

	// Sampled once a sweep per caller, so a caller held there five minutes is five samples.
	retakeShares := factory.NewHistogram(prometheus.HistogramOpts{
		Name:    "click_retake_share",
		Help:    "Share of a caller's takes that win back a tile its country just lost, per caller per sweep",
		Buckets: []float64{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 0.95, 0.99, 1.0},
	})

	// Counts flags, not callers, and once per watchdog that argued for each one:
	// a caller flagged six times is six here and one on shadowban_flagged, and
	// the gap between the two is the thing to look at.
	flags := factory.NewCounterVec(prometheus.CounterOpts{
		Name: "shadowban_flags",
		Help: "Times a caller has been flagged, counted once per watchdog that argued for it",
	}, []string{"watchdog"})

	return antibot.Observer{
		OnReaction: func(delay time.Duration) { reactions.Observe(delay.Seconds()) },

		OnRetakeShare: retakeShares.Observe,

		// The address goes in the log and never on a label: per-IP labels are
		// unbounded cardinality, and they would put personal data in every scrape.
		OnFlag: func(report antibot.Report) {
			fields := make([]any, 0, 10+len(report.Opinions))
			fields = append(fields,
				slog.String("scope", report.Scope),
				slog.Int("flags", report.Flags),
				slog.Int("offence", report.Offence),
				slog.Time("bannedUntil", report.BannedUntil),
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
		},

		OnStateError: func(err error) {
			logger.Error("antibot state not persisted", slog.Any("error", err))
		},

		OnStart: func(described antibot.Description) {
			logger.Info("antibot enabled",
				slog.Any("watchdogs", described.Watchdogs),
				slog.Int("minSuspects", described.MinSuspects),
				slog.Bool("enforce", described.Enforcing),
			)
		},
	}
}
