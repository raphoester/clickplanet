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

	// Set once a sweep. A pool that rotates addresses shows here as a floor that
	// never drops to zero, long before its cohorts chain into a ban.
	cohortScopes := factory.NewGauge(prometheus.GaugeOpts{
		Name: "click_cohort_scopes",
		Help: "Callers clicking in step with another caller: same flag, same start, same pace",
	})

	mapReads := factory.NewHistogram(prometheus.HistogramOpts{
		Name:    "click_map_reads",
		Help:    "Whole maps a clicking caller read beyond one per stream it opened, over the scraper's trackWindow, per caller per sweep",
		Buckets: []float64{0, 0.5, 1, 2, 3, 5, 8, 12, 15, 20, 30, 50},
	})

	// Counts flags, not callers, and once per watchdog that argued for each one:
	// a caller flagged six times is six here and one on shadowban_flagged, and
	// the gap between the two is the thing to look at.
	flags := factory.NewCounterVec(prometheus.CounterOpts{
		Name: "shadowban_flags",
		Help: "Times a caller has been flagged, counted once per watchdog that argued for it",
	}, []string{"watchdog"})

	// Counts rises, not clicks: a watchdog's reading of a caller reaching a level
	// it has not held within jury.suspicionWindow. Levels are cumulative, so
	// suspect includes every certain, and suspect minus certain is the near misses.
	// It is the signal on a day with no ban: how close the watchdogs came.
	opinions := factory.NewCounterVec(prometheus.CounterOpts{
		Name: "antibot_opinions_total",
		Help: "Times a watchdog's reading of a caller rose to a level it had not held within jury.suspicionWindow; suspect includes certain",
	}, []string{"watchdog", "level"})

	// Set once a jury sweep, so it lags by up to a minute.
	standing := factory.NewGaugeVec(prometheus.GaugeOpts{
		Name: "antibot_opinions_standing",
		Help: "Callers a watchdog reads at a level or above at the last jury sweep; suspect includes certain",
	}, []string{"watchdog", "level"})

	// Issued against passed is the pair to read together: a challenge a real
	// player met is answered within seconds, and the gap between the two is
	// what a script costs itself by ignoring one. Labelled by nothing — the
	// address is what would be worth labelling and never can be.
	challenges := factory.NewCounter(prometheus.CounterOpts{
		Name: "antibot_challenges_issued",
		Help: "Callers asked to prove they are a person again, whether or not challenge.enforce is on",
	})

	passed := factory.NewCounter(prometheus.CounterOpts{
		Name: "antibot_challenges_passed",
		Help: "Challenges answered by a caller coming back with a session it was not challenged on",
	})

	return antibot.Observer{
		OnReaction: func(delay time.Duration) { reactions.Observe(delay.Seconds()) },

		OnRetakeShare: retakeShares.Observe,

		OnCohortScopes: func(scopes int) { cohortScopes.Set(float64(scopes)) },

		OnMapReads: mapReads.Observe,

		// The address goes in the log and never on a label: per-IP labels are
		// unbounded cardinality, and they would put personal data in every scrape.
		OnFlag: func(report antibot.Report) {
			fields := append(caller(report),
				slog.Int("flags", report.Flags),
				slog.Int("offence", report.Offence),
				slog.Time("bannedUntil", report.BannedUntil),
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

		// Info rather than Warn: a challenge is a question, not a sentence, and
		// on a busy day there are more of them than there are bans.
		OnChallenge: func(report antibot.Report) {
			challenges.Inc()

			fields := caller(report)
			for _, opinion := range report.Opinions {
				fields = append(fields, slog.String(opinion.Watchdog, opinion.String()))
			}

			logger.Info("antibot challenge", fields...)
		},

		OnChallengePassed: passed.Inc,

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
				slog.Int("challengeAt", described.ChallengeAt),
				slog.Bool("challenge", described.Challenging),
				slog.Duration("challengeFor", described.ChallengeFor),
			)
		},
	}
}

// caller is what a ban line and a challenge line say the same way: who, how
// much, and whether the finding changed anything or was only counted.
func caller(report antibot.Report) []any {
	return []any{
		slog.String("scope", report.Scope),
		slog.Bool("enforced", report.Enforced),
		slog.Int("clicks", report.Clicks),
		slog.Duration("activeFor", report.ActiveFor),
		slog.Duration("longestGap", report.LongestGap),
		slog.String("topCountry", report.TopCountry),
		slog.Int("topCountryClicks", report.TopCountryClicks),
		slog.Any("tiles", report.Tiles),
	}
}
