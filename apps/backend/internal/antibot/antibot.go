// Package antibot holds what is left after the address blocklist and the session
// mint: a caller who solved Turnstile in a real browser and then pointed a script
// at the API. Nothing about the address or the token separates them from a
// player, so everything here reads behaviour instead.
//
// This file is the whole of what a caller may use. The watchdogs, the jury that
// crosses what they say and the ban they all pass live under internal/, so the
// click edge cannot assemble them itself — how a bot is recognised is this
// package's business and changing it is one package's edit.
package antibot

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/catcher"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/cohort"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/defender"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/evidence"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/jury"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/metronome"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/postgres_ban_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/postgres_evidence_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/retaker"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/scraper"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/sequencer"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/shadowban"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

// The two types a caller writes down, because they appear in the signatures it
// implements: it builds a Click and it is handed a Report. They are defined
// under internal/detect because the watchdogs share them and cannot import this
// package without a cycle, so these aliases are what publish them and there is
// still one definition of each.
//
// Nothing else is aliased, because nothing else has to be named. A caller ranges
// a Report's opinions and asks each one whether it Fired and what it says for
// itself, and an Examination's Readings are already strings; the verdict ladder,
// the rule that tripped and the numbers behind it never leave this package as
// vocabulary the edge has to speak.
type (
	Click    = detect.Click       // one Click RPC, as the guard sees it
	Report   = detect.Report      // one ban, with every watchdog's opinion behind it
	Sentence = shadowban.Sentence // a scope's or an account's ban, as the operator tools read it

	Examination = detect.Examination // what the jury holds on one scope and the bans on it, as InspectPlayer reads it
	Reading     = detect.Reading     // one watchdog's opinion, already worded
)

// Config is the `antiBot:` block. A watchdog left out of the file is off, and the
// nested types are this package's or its interior's, so a new bound lands with
// the watchdog that reads it rather than here. None of them is named outside:
// koanf fills them by reflection and a caller still sets their fields, it just
// cannot write the type — the settings are published because they are in the
// file, the code that reads them is not.
type Config struct {
	Enabled bool

	// Database is where the bans and the evidence are kept between boots, in a schema of their own.
	Database cppg.Config

	ShadowBan shadowban.Config
	Jury      jury.Config
	Evidence  evidence.Config

	Retaker   retakerConfig
	Sequencer sequencerConfig
	Metronome metronomeConfig
	Defender  defenderConfig
	Catcher   catcherConfig
	Cohort    cohortConfig
	Scraper   scraperConfig
}

// Validate refuses a bound that cannot mean what it says. Only the cohort has
// any yet: the other watchdogs clamp what they are given.
func (c Config) Validate() error {
	if !c.Enabled {
		return nil
	}

	if err := c.Database.Validate(); err != nil {
		return fmt.Errorf("antiBot.database: %w", err)
	}

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

// Observer is how a finding leaves this package, which measures and judges but
// logs and counts nothing itself. Every hook is optional.
type Observer struct {
	// Every reaction, not only the ones arguing for a ban: the shape of the whole
	// distribution is what shows the bot band.
	OnReaction func(delay time.Duration)

	// Each caller's retake share, once a sweep, whether or not a bound is set to judge it.
	OnRetakeShare func(share float64)

	// Each caller's click gap skew, once a sweep, whether or not a bound is set to judge it.
	OnGapSkew func(skew float64)

	// Each caller's coherence on the metronome's clock period, once a sweep, while a clock bound is set.
	OnClockCoherence func(coherence float64)

	// How many callers are clicking in step with another, once a sweep, whether or not it reads as more than clear.
	OnCohortScopes func(scopes int)

	// Each clicking caller's whole maps read beyond one per stream opened, once a sweep, whether or not it reads as more than clear.
	OnMapReads func(maps float64)

	OnFlag func(report Report)

	// OnRise is a watchdog's reading of a caller reaching level, "suspect" or
	// "certain", when it has not held that level inside jury.suspicionWindow: a
	// rise, not a click. Levels are cumulative, so going straight to certain rises
	// to suspect too. It is the near miss a ban never reports.
	OnRise func(watchdog, level string)

	// OnStanding is, once a jury sweep, how many callers a watchdog reads at level
	// or above right now, zero included — what the jury would count if it
	// deliberated at that moment.
	OnStanding func(watchdog, level string, callers int)

	// Bans or evidence that could not be saved, or a section that did not decode.
	OnStateError func(err error)

	// What was turned on, once, when the guard starts running. Never called when the block is off.
	OnStart func(description Description)
}

// New assembles the watchdogs the config asks for, the jury that crosses them and
// the one ban they all pass. With the block off it hands back a guard that drops
// and bans nothing, so a caller wires it the same way either way; enabling it with
// every watchdog off is an error, because that measures nothing while looking like a defence.
// Nothing connects here: LoadState does.
func New(config Config, clock cptime.Clock, observer Observer) (*Guard, error) {
	if !config.Enabled {
		return &Guard{}, nil
	}

	db := cppg.New(config.Database)

	return build(config, clock, observer, postgres{db: db},
		postgres_ban_store.NewScopes(db), postgres_ban_store.NewAccounts(db), postgres_evidence_store.New(db))
}

// database is the pool behind the persistences: opened by LoadState, closed when Run returns.
type database interface {
	open(ctx context.Context) error
	close() error
}

type postgres struct {
	db *cppg.Postgres
}

func (p postgres) open(ctx context.Context) error {
	if err := p.db.ConnectCtx(ctx); err != nil {
		return fmt.Errorf("failed to connect the antibot to postgres: %w", err)
	}
	if err := p.db.Migrate(ctx, migrations.FS); err != nil {
		_ = p.db.Close()
		return fmt.Errorf("failed to migrate the antibot schema: %w", err)
	}
	return nil
}

func (p postgres) close() error { return p.db.Close() }

func build(
	config Config,
	clock cptime.Clock,
	observer Observer,
	db database,
	scopeBans shadowban.Persistence,
	accountBans shadowban.Persistence,
	evidences evidence.Persistence,
) (*Guard, error) {
	if clock == nil {
		clock = cptime.SystemClock{}
	}

	onStateError := observer.OnStateError
	if onStateError == nil {
		onStateError = func(error) {}
	}

	g := &Guard{onStart: observer.OnStart, onStateError: onStateError, database: db}

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

		watchdog := metronome.New(config.Metronome.Detector, clock, onGapSkew, onClockCoherence)
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

	if len(watchdogs) == 0 {
		return nil, fmt.Errorf("antiBot is enabled with no watchdog turned on")
	}

	// Defaulted here so the description carries the bounds actually enforced.
	juryConfig := config.Jury.WithDefaults()

	banner := shadowban.NewBans(config.ShadowBan, clock, scopeBans, accountBans, onStateError)
	g.runners = append(g.runners, banner.Run)

	g.banner = banner
	g.jury = jury.New(juryConfig, banner, clock, juryHooks(observer), watchdogs...)
	g.runners = append(g.runners, g.jury.Run)

	g.evidence = evidence.New(config.Evidence, clock, evidences, onStateError, append(sections, g.jury)...)
	g.runners = append(g.runners, g.evidence.Run)

	g.description = Description{
		Watchdogs:   names,
		MinSuspects: juryConfig.MinSuspects,
		Enforcing:   banner.Enforcing(),
	}

	return g, nil
}

// juryHooks words the jury's levels for the observer, so the ladder never leaves this package as a type.
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

// Description is what a guard says about itself, through OnStart, for the caller's boot line. Not
// the config back: this is what was turned on, after the defaults were applied.
type Description struct {
	Watchdogs   []string
	MinSuspects int
	Enforcing   bool
}

// Guard is what the click edge gates on. It is a struct, not an interface: each
// caller declares the one or two methods it uses, so the zero Guard New hands
// back when the block is off is enough. That one drops nothing, bans nothing and
// its Run returns at once.
type Guard struct {
	jury        *jury.Jury        // nil when the block is off
	banner      *shadowban.Bans   // nil when the block is off
	evidence    *evidence.Store   // nil when the block is off
	database    database          // nil when the block is off
	catcher     *catcher.Watchdog // nil when the catcher is off
	scraper     *scraper.Watchdog // nil when the scraper is off
	runners     []func(context.Context)
	description Description
	onStart     func(Description)

	onStateError func(error)
}

// Attempted is every click tried, before the throttle: a loop's timing survives only here.
func (g *Guard) Attempted(click Click) {
	if g.Enabled() {
		g.jury.Attempted(click)
	}
}

// Inspect is called before the handler runs, because the map stops remembering
// who held the tile the moment it does.
func (g *Guard) Inspect(click Click) (drop bool) {
	return g.Enabled() && g.jury.Inspect(click)
}

// Committed is only for a click the handler accepted: a refused one recorded as a
// take is how the next honest clicker of that tile comes to look like it is reacting.
func (g *Guard) Committed(click Click) {
	if g.Enabled() {
		g.jury.Committed(click)
	}
}

// Caught and Missed tell the guard what a caller did with a bonus box offered to
// them: claimed after a delay, or let lapse. They say nothing about the click
// that follows until the jury next asks.
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

// Foreign tells the guard a caller claimed a box that was offered to somebody else, or to nobody.
func (g *Guard) Foreign(scope string) {
	if g.catcher != nil {
		g.catcher.Foreign(scope)
	}
}

// Fetched and Listened tell the guard what a caller read: a share of the map, or the live stream opened.
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

// LoadState connects to postgres, migrates the antibot schema and reads the bans and the evidence. An error refuses the boot.
func (g *Guard) LoadState(ctx context.Context) error {
	if !g.Enabled() {
		return nil
	}

	if err := g.database.open(ctx); err != nil {
		return err
	}

	if err := g.banner.Load(ctx); err != nil {
		_ = g.database.close()
		return fmt.Errorf("failed to load the bans: %w", err)
	}

	if err := g.evidence.Load(ctx); err != nil {
		_ = g.database.close()
		return fmt.Errorf("failed to load the antibot evidence: %w", err)
	}

	return nil
}

// Flagged is how many callers are currently banned, for the gauge.
func (g *Guard) Flagged() int {
	if !g.Enabled() {
		return 0
	}

	return g.jury.Flagged()
}

// Ban is an operator's ban on a scope or on an account, whichever is named; a zero duration takes the
// ladder's. An account named alone is banned alone: which scope it plays from is not known here.
func (g *Guard) Ban(scope, account string, duration time.Duration) Sentence {
	if !g.Enabled() {
		return Sentence{}
	}

	return g.banner.Ban(shadowban.Caller{Scope: scope, Account: account}, duration)
}

// Sentence is the running ban on the scope or the account that ends last.
func (g *Guard) Sentence(scope, account string) (Sentence, bool) {
	if !g.Enabled() {
		return Sentence{}, false
	}

	return g.banner.Sentence(shadowban.Caller{Scope: scope, Account: account})
}

// Examine reads every watchdog's opinion and the jury's decision on a scope, and any ban on the scope or
// the account, and changes nothing.
func (g *Guard) Examine(scope, account string) Examination {
	if !g.Enabled() {
		return Examination{Scope: scope, Account: account}
	}

	examination := g.jury.Examine(scope)
	examination.Account = account

	if sentence, banned := g.banner.Sentence(shadowban.Caller{Scope: scope, Account: account}); banned {
		examination.Banned = true
		examination.Flags = sentence.Flags
		examination.Offence = sentence.Offence
		examination.BannedUntil = sentence.Until
	}

	return examination
}

// Enforcing says whether a ban drops anything.
func (g *Guard) Enforcing() bool { return g.Enabled() && g.banner.Enforcing() }

// Banned says whether a caller's actions should be dropped, for its scope or its account: false while enforce is off.
func (g *Guard) Banned(scope, account string) bool {
	return g.Enabled() && g.banner.Banned(shadowban.Caller{Scope: scope, Account: account})
}

// Enabled is false for the guard New hands back when the block is off.
func (g *Guard) Enabled() bool { return g.jury != nil }

func (g *Guard) Name() string { return "antibot" }

// Run fans out to every sweeper enabled and blocks until they all return. One
// runner whatever the config turned on: how many sweepers there are is this
// package's business. The pool closes after the last flush: closers run before runners stop.
func (g *Guard) Run(ctx context.Context) {
	if !g.Enabled() {
		return
	}

	// Before any runner, and before the server listens: the outage ends when this process starts watching.
	g.evidence.Resume()

	if g.onStart != nil {
		g.onStart(g.description)
	}

	var wg sync.WaitGroup

	for _, run := range g.runners {
		wg.Add(1)
		go func() {
			defer wg.Done()
			run(ctx)
		}()
	}

	wg.Wait()

	if err := g.database.close(); err != nil {
		g.onStateError(fmt.Errorf("failed to close the antibot postgres pool: %w", err))
	}
}
