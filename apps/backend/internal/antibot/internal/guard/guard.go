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
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/convictions"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/defender"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/evidence"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/hopper"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/jury"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/metronome"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/retaker"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/rotator"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/rules"
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

	Conviction = convictions.Conviction

	Examination = detect.Examination
	Reading     = detect.Reading
)

// The machine the antibot runs on; what it judges with is the rule set in force (rules.InForce).
type Config struct {
	Jury     jury.Config
	Evidence evidence.Config

	Retaker   retaker.Config
	Sequencer sequencer.Config
	Metronome metronome.Config
	Defender  defender.Config
	Catcher   catcher.Config
	Cohort    cohort.Config
	Scraper   scraper.Config
	Churner   churner.Config
	Hopper    hopper.Config
	Rotator   rotator.Config
}

func (c Config) Validate() error {
	if err := c.Cohort.Validate(); err != nil {
		return fmt.Errorf("antiBot.cohort: %w", err)
	}

	return nil
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

// Parts are what a rule set holds: one per component the guard builds.
func Parts() rules.Parts {
	return rules.Parts{
		bans.Part, jury.Part, retaker.Part, sequencer.Part, metronome.Part, defender.Part,
		catcher.Part, cohort.Part, scraper.Part, churner.Part, hopper.Part, rotator.Part,
	}
}

// Every watchdog is built: one the rules turn off keeps watching, and the jury reads it as clear.
func New(
	config Config,
	inForce *rules.InForce,
	clock cptime.Clock,
	observer Observer,
	banner *bans.Book,
	convicted convictions.Store,
	evidences evidence.Persistence,
	heard ...evidence.Section,
) *Guard {
	onStateError := orNothing(observer.OnStateError)

	g := &Guard{onStart: observer.OnStart, onStateError: onStateError, rules: inForce, banner: banner, convictions: convicted}

	onScopeAccounts := observer.OnScopeAccounts
	if onScopeAccounts == nil {
		onScopeAccounts = func(int, string) {}
	}

	g.catcher = catcher.New(config.Catcher, rules.Reader(inForce, catcher.Part), clock)
	g.scraper = scraper.New(config.Scraper, rules.Reader(inForce, scraper.Part), clock, orNothing(observer.OnMapReads))

	watchdogs := []interface {
		detect.Watchdog
		evidence.Section
		Run(ctx context.Context)
	}{
		retaker.New(config.Retaker, rules.Reader(inForce, retaker.Part), clock, observer.OnReaction),
		sequencer.New(config.Sequencer, rules.Reader(inForce, sequencer.Part), clock),
		metronome.New(config.Metronome, rules.Reader(inForce, metronome.Part), clock, orNothing(observer.OnGapSkew),
			orNothing(observer.OnClockCoherence), orNothing(observer.OnBusyTime)),
		defender.New(config.Defender, rules.Reader(inForce, defender.Part), clock, observer.OnRetakeShare),
		g.catcher,
		cohort.New(config.Cohort, rules.Reader(inForce, cohort.Part), clock, observer.OnCohortScopes),
		g.scraper,
		churner.New(config.Churner, rules.Reader(inForce, churner.Part), clock, onScopeAccounts, orNothing(observer.OnRelayLinks)),
		hopper.New(config.Hopper, rules.Reader(inForce, hopper.Part), clock, orNothing(observer.OnHopShare)),
		rotator.New(config.Rotator, rules.Reader(inForce, rotator.Part), clock, orNothing(observer.OnRotationShare)),
	}

	sections := make([]evidence.Section, 0, len(watchdogs)+2+len(heard))
	for _, watchdog := range watchdogs {
		g.runners = append(g.runners, watchdog.Run)
		g.watchdogs = append(g.watchdogs, watchdog)
		sections = append(sections, watchdog)
	}

	g.jury = jury.New(config.Jury, rules.Reader(inForce, jury.Part), banner, convicted, clock, juryHooks(observer), g.watchdogs...)
	g.runners = append(g.runners, g.jury.Run)

	g.evidence = evidence.New(config.Evidence, clock, evidences, onStateError, append(append(sections, g.jury, g.jury.Accounts()), heard...)...)
	g.runners = append(g.runners, g.evidence.Run)

	return g
}

func orNothing[T any](observe func(T)) func(T) {
	if observe == nil {
		return func(T) {}
	}
	return observe
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
	Rules       string
	MinSuspects int
	Enforcing   bool
}

type Guard struct {
	jury        *jury.Jury
	banner      *bans.Book
	convictions convictions.Store
	evidence    *evidence.Store
	catcher     *catcher.Watchdog
	scraper     *scraper.Watchdog
	runners     []func(context.Context)
	watchdogs   []detect.Watchdog
	rules       *rules.InForce
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
	g.catcher.Caught(scope, after)
}

func (g *Guard) Missed(scope string) {
	g.catcher.Missed(scope)
}

func (g *Guard) Foreign(scope string) {
	g.catcher.Foreign(scope)
}

func (g *Guard) Fetched(scope string, maps float64, offMap bool) {
	g.scraper.Fetched(scope, maps, offMap)
}

func (g *Guard) Listened(scope string) {
	g.scraper.Listened(scope)
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

// From since, until before until, oldest first.
func (g *Guard) Convictions(ctx context.Context, since, until time.Time) ([]Conviction, error) {
	return g.convictions.Between(ctx, since, until) //nolint:wrapcheck // the store already named what failed.
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
		var on []string
		for _, watchdog := range g.watchdogs {
			if watchdog.Enabled() {
				on = append(on, watchdog.Name())
			}
		}
		g.onStart(Description{
			Watchdogs:   on,
			Rules:       string(g.rules.Name()),
			MinSuspects: rules.Reader(g.rules, jury.Part)().MinSuspects,
			Enforcing:   g.banner.Enforcing(),
		})
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
