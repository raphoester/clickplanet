package antibot_click

import (
	"log/slog"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot"
)

func NewObserver(logger *slog.Logger, registerer prometheus.Registerer) antibot.Observer {
	factory := promauto.With(registerer)

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

	retakeShares := factory.NewHistogram(prometheus.HistogramOpts{
		Name:    "click_retake_share",
		Help:    "Share of a caller's takes that win back a tile its country just lost, per caller per sweep",
		Buckets: []float64{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 0.95, 0.99, 1.0},
	})

	gapSkews := factory.NewHistogram(prometheus.HistogramOpts{
		Name:    "click_gap_skew",
		Help:    "Skew (p90 + p10 - 2*p50) / (p90 - p10) of a caller's gaps between clicks tried, per caller per sweep",
		Buckets: []float64{-0.5, -0.3, -0.2, -0.1, 0, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0},
	})

	busyHours := factory.NewHistogram(prometheus.HistogramOpts{
		Name:    "click_busy_hours",
		Help:    "Hours of the metronome's stamina window a payer spent clicking at pace, per payer that clicked since the last sweep",
		Buckets: []float64{0.5, 1, 1.5, 2, 2.5, 3, 3.5, 4, 4.5, 5, 5.5, 6, 7, 8},
	})

	clockCoherences := factory.NewHistogram(prometheus.HistogramOpts{
		Name:    "click_clock_coherence",
		Help:    "How closely a caller's clicks tried keep one beat of the metronome's clock period, 0 to 1, per caller per sweep",
		Buckets: []float64{0.05, 0.1, 0.15, 0.2, 0.25, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0},
	})

	cohortScopes := factory.NewGauge(prometheus.GaugeOpts{
		Name: "click_cohort_scopes",
		Help: "Callers clicking in step with another caller: same flag, same start, same pace",
	})

	mapReads := factory.NewHistogram(prometheus.HistogramOpts{
		Name:    "click_map_reads",
		Help:    "Whole maps a clicking caller read beyond one per stream it opened, over the scraper's trackWindow, per caller per sweep",
		Buckets: []float64{0, 0.5, 1, 2, 3, 5, 8, 12, 15, 20, 30, 50},
	})

	scopeAccounts := factory.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "click_scope_accounts",
		Help:    "Guest accounts first seen on one scope inside the churner's window, per scope clicked on since the last sweep",
		Buckets: []float64{1, 2, 3, 4, 5, 6, 8, 10, 15, 20, 30, 50},
	}, []string{"family"})

	relayLinks := factory.NewHistogram(prometheus.HistogramOpts{
		Name:    "click_relay_links",
		Help:    "Fresh guest accounts that took over from a short-lived one on the same flag and prefix inside the churner's window, per relay per sweep",
		Buckets: []float64{0, 1, 2, 3, 4, 5, 6, 8, 10, 15, 20, 30},
	})

	flags := factory.NewCounterVec(prometheus.CounterOpts{
		Name: "shadowban_flags",
		Help: "Times a caller has been flagged, counted once per watchdog that argued for it",
	}, []string{"watchdog"})

	opinions := factory.NewCounterVec(prometheus.CounterOpts{
		Name: "antibot_opinions_total",
		Help: "Times a watchdog's reading of a caller rose to a level it had not held within jury.suspicionWindow; suspect includes certain",
	}, []string{"watchdog", "level"})

	standing := factory.NewGaugeVec(prometheus.GaugeOpts{
		Name: "antibot_opinions_standing",
		Help: "Callers a watchdog reads at a level or above at the last jury sweep; suspect includes certain",
	}, []string{"watchdog", "level"})

	return antibot.Observer{
		OnReaction: func(delay time.Duration) { reactions.Observe(delay.Seconds()) },

		OnRetakeShare: retakeShares.Observe,

		OnGapSkew: gapSkews.Observe,

		OnClockCoherence: clockCoherences.Observe,

		OnBusyTime: func(busy time.Duration) { busyHours.Observe(busy.Hours()) },

		OnCohortScopes: func(scopes int) { cohortScopes.Set(float64(scopes)) },

		OnMapReads: mapReads.Observe,

		OnScopeAccounts: func(accounts int, family string) {
			scopeAccounts.WithLabelValues(family).Observe(float64(accounts))
		},

		OnRelayLinks: func(links int) { relayLinks.Observe(float64(links)) },

		// The address goes in the log, never on a label: unbounded cardinality and personal data.
		OnFlag: func(report antibot.Report) {
			fields := make([]any, 0, 11+len(report.Opinions))
			fields = append(fields,
				slog.String("scope", report.Scope),
				slog.String("account", report.Account),
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

			for _, opinion := range report.Opinions {
				fields = append(fields, slog.String(opinion.Watchdog, opinion.String()))

				if opinion.Fired() {
					flags.WithLabelValues(opinion.Watchdog).Inc()
				}
			}

			logger.Warn("antibot ban", fields...)
		},

		OnRise: func(watchdog, level string) {
			opinions.WithLabelValues(watchdog, level).Inc()
		},

		OnStanding: func(watchdog, level string, callers int) {
			standing.WithLabelValues(watchdog, level).Set(float64(callers))
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
