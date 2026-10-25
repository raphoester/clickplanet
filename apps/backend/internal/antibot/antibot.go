package antibot

import (
	"context"
	"fmt"
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
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/postgres_ban_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/postgres_conviction_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/postgres_evidence_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/retaker"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/rotator"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/scraper"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/sequencer"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
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

type Config struct {
	Enabled bool

	Database cppg.Config

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

func New(config Config, clock cptime.Clock, observer Observer) (*Guard, error) {
	if !config.Enabled {
		return &Guard{}, nil
	}

	db := cppg.New(config.Database)

	return build(config, clock, observer, postgres{db: db},
		postgres_ban_store.NewScopes(db), postgres_ban_store.NewAccounts(db), postgres_conviction_store.New(db),
		postgres_evidence_store.New(db))
}

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
	scopeBans bans.Store,
	accountBans bans.Store,
	convicted convictions.Store,
	evidences evidence.Persistence,
) (*Guard, error) {
	if clock == nil {
		clock = cptime.SystemClock{}
	}

	onStateError := observer.OnStateError
	if onStateError == nil {
		onStateError = func(error) {}
	}

	g := &Guard{onStart: observer.OnStart, onStateError: onStateError, database: db, convictions: convicted}

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

	banner := bans.NewBook(config.Bans, config.Evidence.WithDefaults().Retention, clock, scopeBans, accountBans)

	g.banner = banner
	g.jury = jury.New(juryConfig, banner, convicted, clock, juryHooks(observer), watchdogs...)
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
	convictions convictions.Store
	evidence    *evidence.Store
	database    database
	catcher     *catcher.Watchdog
	scraper     *scraper.Watchdog
	runners     []func(context.Context)
	description Description
	onStart     func(Description)

	onStateError func(error)
}

func (g *Guard) Attempted(click Click) {
	if g.Enabled() {
		g.jury.Attempted(click)
	}
}

const banTimeout = time.Second

// Call before the handler runs: afterwards the map no longer knows who held the tile.
func (g *Guard) Inspect(ctx context.Context, click Click) {
	if !g.Enabled() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, banTimeout)
	defer cancel()

	if err := g.jury.Inspect(ctx, click); err != nil {
		g.onStateError(err)
	}
}

// Only for a click the handler accepted; a refused one would frame the next clicker.
func (g *Guard) Committed(click Click) {
	if g.Enabled() {
		g.jury.Committed(click)
	}
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

func (g *Guard) LoadState(ctx context.Context) error {
	if !g.Enabled() {
		return nil
	}

	if err := g.database.open(ctx); err != nil {
		return err
	}

	if err := g.evidence.Load(ctx); err != nil {
		_ = g.database.close()
		return fmt.Errorf("failed to load the antibot evidence: %w", err)
	}

	return nil
}

func (g *Guard) Flagged(ctx context.Context) (int, error) {
	if !g.Enabled() {
		return 0, nil
	}

	return g.jury.Flagged(ctx) //nolint:wrapcheck // the banner already named what failed.
}

func (g *Guard) Ban(ctx context.Context, scope, account string, duration time.Duration) (Sentence, error) {
	if !g.Enabled() {
		return Sentence{}, nil
	}

	return g.banner.Ban(ctx, bans.Caller{Scope: scope, Account: account}, duration) //nolint:wrapcheck // the banner already named what failed.
}

func (g *Guard) Unban(ctx context.Context, scope, account string) error {
	if !g.Enabled() {
		return nil
	}

	return g.banner.Unban(ctx, bans.Caller{Scope: scope, Account: account}) //nolint:wrapcheck // the banner already named what failed.
}

func (g *Guard) Sentence(ctx context.Context, scope, account string) (Sentence, bool, error) {
	if !g.Enabled() {
		return Sentence{}, false, nil
	}

	return g.banner.Sentence(ctx, bans.Caller{Scope: scope, Account: account}) //nolint:wrapcheck // the banner already named what failed.
}

func (g *Guard) LastSentence(ctx context.Context, scope, account string) (Sentence, bool, error) {
	if !g.Enabled() {
		return Sentence{}, false, nil
	}

	return g.banner.LastSentence(ctx, bans.Caller{Scope: scope, Account: account}) //nolint:wrapcheck // the banner already named what failed.
}

func (g *Guard) Pardon(ctx context.Context, scope, account string) (Pardon, bool, error) {
	if !g.Enabled() {
		return Pardon{}, false, nil
	}

	return g.banner.Pardon(ctx, bans.Caller{Scope: scope, Account: account}) //nolint:wrapcheck // the banner already named what failed.
}

func (g *Guard) Examine(ctx context.Context, scope, account string) (Examination, error) {
	if !g.Enabled() {
		return Examination{Scope: scope, Account: account}, nil
	}

	examination := g.jury.Examine(scope)
	examination.Account = account

	caller := bans.Caller{Scope: scope, Account: account}

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
	if !g.Enabled() {
		return nil, nil
	}

	return g.convictions.Between(ctx, since, until) //nolint:wrapcheck // the store already named what failed.
}

func (g *Guard) Enforcing() bool { return g.Enabled() && g.banner.Enforcing() }

// A ban that cannot be read is no ban: the caller plays on, and the error says why.
func (g *Guard) EnforcedSentence(ctx context.Context, scope, account string) (Sentence, bool) {
	if !g.Enabled() {
		return Sentence{}, false
	}

	ctx, cancel := context.WithTimeout(ctx, banTimeout)
	defer cancel()

	sentence, banned, err := g.banner.EnforcedSentence(ctx, bans.Caller{Scope: scope, Account: account})
	if err != nil {
		g.onStateError(err)
		return Sentence{}, false
	}

	return sentence, banned
}

func (g *Guard) Enabled() bool { return g.jury != nil }

func (g *Guard) Name() string { return "antibot" }

func (g *Guard) Run(ctx context.Context) {
	if !g.Enabled() {
		return
	}

	// Before any runner starts: the outage ends when this process starts watching.
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
