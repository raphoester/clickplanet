package guard

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/bans"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/catcher"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/churner"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/cohort"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/defender"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/evidence"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/hopper"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/jury"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/metronome"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/retaker"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/rotator"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/scraper"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/sequencer"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type (
	Click    = detect.Click
	Point    = detect.Point
	Report   = detect.Report
	Sentence = bans.Sentence
	Pardon   = bans.Pardon

	Examination = detect.Examination
	Reading     = detect.Reading
)

type Config struct {
	Bans     bans.Config
	Jury     jury.Config
	Evidence evidence.Config

	Retaker   retakerConfig
	Sequencer sequencerConfig
	Metronome metronomeConfig
	Defender  defenderConfig
	Catcher   catcherConfig
	Cohort    cohortConfig
	Scraper   scraperConfig
	Churner   churnerConfig
	Hopper    hopperConfig
	Rotator   rotatorConfig
}

func (c Config) Validate() error {
	if !c.Cohort.Enabled {
		return nil
	}

	if err := c.Cohort.Detector.Validate(); err != nil {
		return fmt.Errorf("antiBot.cohort.detector: %w", err)
	}

	return nil
}

type retakerConfig struct {
	Enabled  bool
	Detector retaker.Config
}

type sequencerConfig struct {
	Enabled  bool
	Detector sequencer.Config
}

type metronomeConfig struct {
	Enabled  bool
	Detector metronome.Config
}

type defenderConfig struct {
	Enabled  bool
	Detector defender.Config
}

type catcherConfig struct {
	Enabled  bool
	Detector catcher.Config
}

type cohortConfig struct {
	Enabled  bool
	Detector cohort.Config
}

type scraperConfig struct {
	Enabled  bool
	Detector scraper.Config
}

type churnerConfig struct {
	Enabled  bool
	Detector churner.Config
}

type hopperConfig struct {
	Enabled  bool
	Detector hopper.Config
}

type rotatorConfig struct {
	Enabled  bool
	Detector rotator.Config
}

type Observer struct {
	OnReaction func(delay time.Duration)

	OnRetakeShare func(share float64)

	OnGapSkew func(skew float64)

	OnClockCoherence func(coherence float64)

	OnBusyTime func(busy time.Duration)

	OnCohortScopes func(scopes int)

	OnMapReads func(maps float64)

	OnScopeAccounts func(accounts int, family string)

	OnRelayLinks func(links int)

	OnHopShare func(share float64)

	OnRotationShare func(share float64)

	OnFlag func(report Report)

	OnRise func(watchdog, level string)

	OnStanding func(watchdog, level string, callers int)

	OnStateError func(err error)

	OnStart func(description Description)
}

func New(
	config Config,
	clock cptime.Clock,
	observer Observer,
	banner *bans.Book,
	evidences evidence.Persistence,
	heard ...evidence.Section,
) (*Guard, error) {
	onStateError := observer.OnStateError
	if onStateError == nil {
		onStateError = func(error) {}
	}

	g := &Guard{onStart: observer.OnStart, onStateError: onStateError}

	var (
		watchdogs []detect.Watchdog
		sections  []evidence.Section
		names     []string
	)

	if config.Retaker.Enabled {
		watchdog := retaker.New(config.Retaker.Detector, clock, observer.OnReaction)
		g.runners = append(g.runners, watchdog.Run)
		watchdogs = append(watchdogs, watchdog)
		sections = append(sections, watchdog)
		names = append(names, retaker.Name)
	}

	if config.Sequencer.Enabled {
		watchdog := sequencer.New(config.Sequencer.Detector, clock)
		g.runners = append(g.runners, watchdog.Run)
		watchdogs = append(watchdogs, watchdog)
		sections = append(sections, watchdog)
		names = append(names, sequencer.Name)
	}

	if config.Metronome.Enabled {
		onGapSkew := observer.OnGapSkew
		if onGapSkew == nil {
			onGapSkew = func(float64) {}
		}

		onClockCoherence := observer.OnClockCoherence
		if onClockCoherence == nil {
			onClockCoherence = func(float64) {}
		}

		onBusyTime := observer.OnBusyTime
		if onBusyTime == nil {
			onBusyTime = func(time.Duration) {}
		}

		watchdog := metronome.New(config.Metronome.Detector, clock, onGapSkew, onClockCoherence, onBusyTime)
		g.runners = append(g.runners, watchdog.Run)
		watchdogs = append(watchdogs, watchdog)
		sections = append(sections, watchdog)
		names = append(names, metronome.Name)
	}

	if config.Defender.Enabled {
		watchdog := defender.New(config.Defender.Detector, clock, observer.OnRetakeShare)
		g.runners = append(g.runners, watchdog.Run)
		watchdogs = append(watchdogs, watchdog)
		sections = append(sections, watchdog)
		names = append(names, defender.Name)
	}

	if config.Catcher.Enabled {
		watchdog := catcher.New(config.Catcher.Detector, clock)
		g.runners = append(g.runners, watchdog.Run)
		watchdogs = append(watchdogs, watchdog)
		sections = append(sections, watchdog)
		names = append(names, catcher.Name)
		g.catcher = watchdog
	}

	if config.Cohort.Enabled {
		watchdog := cohort.New(config.Cohort.Detector, clock, observer.OnCohortScopes)
		g.runners = append(g.runners, watchdog.Run)
		watchdogs = append(watchdogs, watchdog)
		sections = append(sections, watchdog)
		names = append(names, cohort.Name)
	}

	if config.Scraper.Enabled {
		onMapReads := observer.OnMapReads
		if onMapReads == nil {
			onMapReads = func(float64) {}
		}

		watchdog := scraper.New(config.Scraper.Detector, clock, onMapReads)
		g.runners = append(g.runners, watchdog.Run)
		watchdogs = append(watchdogs, watchdog)
		sections = append(sections, watchdog)
		names = append(names, scraper.Name)
		g.scraper = watchdog
	}

	if config.Churner.Enabled {
		onScopeAccounts := observer.OnScopeAccounts
		if onScopeAccounts == nil {
			onScopeAccounts = func(int, string) {}
		}

		onRelayLinks := observer.OnRelayLinks
		if onRelayLinks == nil {
			onRelayLinks = func(int) {}
		}

		watchdog := churner.New(config.Churner.Detector, clock, onScopeAccounts, onRelayLinks)
		g.runners = append(g.runners, watchdog.Run)
		watchdogs = append(watchdogs, watchdog)
		sections = append(sections, watchdog)
		names = append(names, churner.Name)
	}

	if config.Hopper.Enabled {
		onHopShare := observer.OnHopShare
		if onHopShare == nil {
			onHopShare = func(float64) {}
		}

		watchdog := hopper.New(config.Hopper.Detector, clock, onHopShare)
		g.runners = append(g.runners, watchdog.Run)
		watchdogs = append(watchdogs, watchdog)
		sections = append(sections, watchdog)
		names = append(names, hopper.Name)
	}

	if config.Rotator.Enabled {
		onRotationShare := observer.OnRotationShare
		if onRotationShare == nil {
			onRotationShare = func(float64) {}
		}

		watchdog := rotator.New(config.Rotator.Detector, clock, onRotationShare)
		g.runners = append(g.runners, watchdog.Run)
		watchdogs = append(watchdogs, watchdog)
		sections = append(sections, watchdog)
		names = append(names, rotator.Name)
	}

	if len(watchdogs) == 0 {
		return nil, fmt.Errorf("antiBot is enabled with no watchdog turned on")
	}

	juryConfig := config.Jury.WithDefaults()

	g.banner = banner
	g.jury = jury.New(juryConfig, banner, clock, juryHooks(observer), watchdogs...)
	g.runners = append(g.runners, g.jury.Run)

	g.evidence = evidence.New(config.Evidence, clock, evidences, onStateError, append(append(sections, g.jury, g.jury.Accounts()), heard...)...)
	g.runners = append(g.runners, g.evidence.Run)

	g.description = Description{
		Watchdogs:   names,
		MinSuspects: juryConfig.MinSuspects,
		Enforcing:   banner.Enforcing(),
	}

	return g, nil
}

func juryHooks(observer Observer) jury.Hooks {
	hooks := jury.Hooks{OnFlag: observer.OnFlag}

	if observer.OnRise != nil {
		hooks.OnRise = func(watchdog string, level detect.Verdict) {
			observer.OnRise(watchdog, level.String())
		}
	}

	if observer.OnStanding != nil {
		hooks.OnStanding = func(watchdog string, level detect.Verdict, callers int) {
			observer.OnStanding(watchdog, level.String(), callers)
		}
	}

	return hooks
}

type Description struct {
	Watchdogs   []string
	MinSuspects int
	Enforcing   bool
}

type Guard struct {
	jury        *jury.Jury
	banner      *bans.Book
	evidence    *evidence.Store
	catcher     *catcher.Watchdog
	scraper     *scraper.Watchdog
	runners     []func(context.Context)
	description Description
	onStart     func(Description)

	onStateError func(error)
}

func (g *Guard) Attempted(click Click) {
	g.jury.Attempted(click)
}

const banTimeout = time.Second

func (g *Guard) Inspect(ctx context.Context, click Click) {
	ctx, cancel := context.WithTimeout(ctx, banTimeout)
	defer cancel()

	if err := g.jury.Inspect(ctx, click); err != nil {
		g.onStateError(err)
	}
}

// Only for a click the game accepted; a refused one would frame the next clicker.
func (g *Guard) Committed(click Click) {
	g.jury.Committed(click)
}

func (g *Guard) Caught(scope string, after time.Duration) {
	if g.catcher != nil {
		g.catcher.Caught(scope, after)
	}
}

func (g *Guard) Missed(scope string) {
	if g.catcher != nil {
		g.catcher.Missed(scope)
	}
}

func (g *Guard) Foreign(scope string) {
	if g.catcher != nil {
		g.catcher.Foreign(scope)
	}
}

func (g *Guard) Fetched(scope string, maps float64, offMap bool) {
	if g.scraper != nil {
		g.scraper.Fetched(scope, maps, offMap)
	}
}

func (g *Guard) Listened(scope string) {
	if g.scraper != nil {
		g.scraper.Listened(scope)
	}
}

func (g *Guard) Load(ctx context.Context) error {
	if err := g.evidence.Load(ctx); err != nil {
		return fmt.Errorf("failed to load the antibot evidence: %w", err)
	}
	return nil
}

func (g *Guard) Flagged(ctx context.Context) (int, error) {
	return g.jury.Flagged(ctx) //nolint:wrapcheck // the banner already named what failed.
}

func (g *Guard) Ban(ctx context.Context, scope, account string, duration time.Duration) (Sentence, error) {
	return g.banner.Ban(ctx, bans.Caller{Scope: scope, Account: account}, duration) //nolint:wrapcheck // the banner already named what failed.
}

func (g *Guard) Unban(ctx context.Context, scope, account string) error {
	return g.banner.Unban(ctx, bans.Caller{Scope: scope, Account: account}) //nolint:wrapcheck // the banner already named what failed.
}

func (g *Guard) LastSentence(ctx context.Context, scope, account string) (Sentence, bool, error) {
	return g.banner.LastSentence(ctx, bans.Caller{Scope: scope, Account: account}) //nolint:wrapcheck // the banner already named what failed.
}

func (g *Guard) Pardon(ctx context.Context, scope, account string) (Pardon, bool, error) {
	return g.banner.Pardon(ctx, bans.Caller{Scope: scope, Account: account}) //nolint:wrapcheck // the banner already named what failed.
}

// An account is read on the scope of its latest click the jury heard.
func (g *Guard) Examine(ctx context.Context, scope, account string) (Examination, error) {
	examination := g.jury.Examine(scope, account)

	caller := bans.Caller{Scope: examination.Scope, Account: account}

	sentence, banned, err := g.banner.Sentence(ctx, caller)
	if err != nil {
		return Examination{}, err //nolint:wrapcheck // the banner already named what failed.
	}
	if banned {
		examination.Banned = true
		examination.Flags = sentence.Flags
		examination.Offence = sentence.Offence
		examination.BannedUntil = sentence.Until
	}

	pardon, pardoned, err := g.banner.Pardon(ctx, caller)
	if err != nil {
		return Examination{}, err //nolint:wrapcheck // the banner already named what failed.
	}
	if pardoned {
		examination.PardonedUntil = pardon.Until
	}

	return examination, nil
}

func (g *Guard) Enforcing() bool { return g.banner.Enforcing() }

func (g *Guard) Name() string { return "antibot" }

// The watchdogs stop, and their evidence is saved, once the last fact they heard is handled.
func (g *Guard) Hearing(feeds ...cpbootstrap.Runner) cpbootstrap.Runner {
	return hearing{guard: g, feeds: feeds}
}

type hearing struct {
	guard *Guard
	feeds []cpbootstrap.Runner
}

func (h hearing) Name() string {
	names := make([]string, 0, 1+len(h.feeds))
	names = append(names, h.guard.Name())
	for _, feed := range h.feeds {
		names = append(names, feed.Name())
	}
	return strings.Join(names, "+")
}

func (h hearing) Run(ctx context.Context) {
	g := h.guard

	// Before any runner starts: the outage ends when this process starts watching.
	g.evidence.Resume()

	if g.onStart != nil {
		g.onStart(g.description)
	}

	watching, stop := context.WithCancel(context.WithoutCancel(ctx))
	defer stop()

	var runners sync.WaitGroup
	for _, run := range g.runners {
		runners.Go(func() { run(watching) })
	}

	var feeds sync.WaitGroup
	for _, feed := range h.feeds {
		feeds.Go(func() { feed.Run(ctx) })
	}

	<-ctx.Done()
	feeds.Wait()
	stop()
	runners.Wait()
}
