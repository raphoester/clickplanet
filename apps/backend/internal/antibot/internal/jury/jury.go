package jury

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/bans"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/convictions"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Rules struct {
	MinSuspects int

	SuspicionWindow time.Duration
}

func (r Rules) Validate() error {
	return errors.Join(
		detect.Positive("minSuspects", r.MinSuspects),
		detect.Positive("suspicionWindow", r.SuspicionWindow),
	)
}

type Config struct {
	TrackWindow time.Duration

	SweepInterval time.Duration
}

const (
	defaultTrackWindow   = 15 * time.Minute
	defaultSweepInterval = time.Minute

	// Bounds memory: a client could cycle through every country code.
	maxTrackedCountries = 16
	keptTiles           = 8
)

func (c Config) withDefaults() Config {
	if c.TrackWindow <= 0 {
		c.TrackWindow = defaultTrackWindow
	}
	if c.SweepInterval <= 0 {
		c.SweepInterval = defaultSweepInterval
	}
	return c
}

type Banner interface {
	Flag(ctx context.Context, caller bans.Caller) (bans.Sentence, bool, error)
	Pardoned(ctx context.Context, caller bans.Caller) (bool, error)
	Flagged(ctx context.Context) (int, error)
}

type Convictions interface {
	Keep(ctx context.Context, conviction convictions.Conviction) error
}

type Hooks struct {
	OnFlag func(detect.Report)

	OnRise func(watchdog string, level detect.Verdict)

	OnStanding func(watchdog string, level detect.Verdict, callers int)
}

func New(
	config Config,
	rules func() Rules,
	banner Banner,
	convictions Convictions,
	clock cptime.Clock,
	hooks Hooks,
	watchdogs ...detect.Watchdog,
) *Jury {
	if clock == nil {
		clock = cptime.SystemClock{}
	}

	return &Jury{
		config:      config.withDefaults(),
		rules:       rules,
		banner:      banner,
		convictions: convictions,
		clock:       clock,
		hooks:       hooks,
		watchdogs:   watchdogs,
		callers:     make(map[string]*caller),
		accounts:    make(map[string]sighting),
	}
}

type Jury struct {
	config      Config
	rules       func() Rules
	banner      Banner
	convictions Convictions
	clock       cptime.Clock
	hooks       Hooks
	watchdogs   []detect.Watchdog

	mu       sync.Mutex
	callers  map[string]*caller
	accounts map[string]sighting
	outage   detect.Outage
}

// Where an account clicked last: the watchdogs judge scopes, so an account is examined on one.
type sighting struct {
	scope string
	at    time.Time
}

type caller struct {
	firstSeen  time.Time
	lastSeen   time.Time
	longestGap time.Duration

	clicks    int
	countries map[string]int
	tiles     []uint32

	opinions map[string]detect.Opinion

	reached map[string]*[detect.Certain + 1]time.Time
}

func (j *Jury) Inspect(ctx context.Context, click detect.Click) error {
	if click.Scope == "" {
		return nil
	}

	j.record(click)

	for _, watchdog := range j.watchdogs {
		// An off watchdog still watches, so its windows are full when it is turned on again.
		verdict, evidence := watchdog.Watch(click)
		if !watchdog.Enabled() {
			verdict, evidence = detect.Clear, detect.Evidence{}
		}
		for _, level := range j.opine(click, watchdog.Name(), verdict, evidence) {
			if j.hooks.OnRise != nil {
				j.hooks.OnRise(watchdog.Name(), level)
			}
		}
	}

	if report, guilty := j.deliberate(click); guilty {
		return j.flag(ctx, click, report)
	}

	return nil
}

func (j *Jury) flag(ctx context.Context, click detect.Click, report detect.Report) error {
	pardoned, err := j.banner.Pardoned(ctx, callerOf(click))
	if err != nil || pardoned {
		return err //nolint:wrapcheck // the banner already named what failed.
	}

	var errs []error

	for _, convict := range convictsOf(click, report) {
		sentence, accepted, err := j.banner.Flag(ctx, convict)
		errs = append(errs, err)
		if !accepted {
			continue
		}

		passed := report
		passed.Account = convict.Account
		if convict.Account != click.Account {
			passed.Alongside = click.Account
		}
		passed.Flags = sentence.Flags
		passed.Offence = sentence.Offence
		passed.BannedUntil = sentence.Until

		errs = append(errs, j.convictions.Keep(ctx, convictions.Of(passed, j.clock.Now())))
		if j.hooks.OnFlag != nil {
			j.hooks.OnFlag(passed)
		}
	}

	return errors.Join(errs...)
}

func callerOf(click detect.Click) bans.Caller {
	return bans.Caller{Scope: click.Scope, Account: click.Account, SignedIn: click.SignedIn}
}

func convictsOf(click detect.Click, report detect.Report) []bans.Caller {
	convicts := []bans.Caller{callerOf(click)}
	named := cpcolls.NewSet(click.Account)

	for _, opinion := range report.Opinions {
		if !opinion.Fired() {
			continue
		}
		for _, accomplice := range opinion.Evidence.Accomplices {
			if named.Contains(accomplice.Account) {
				continue
			}
			named.Add(accomplice.Account)
			convicts = append(convicts, bans.Caller{Scope: click.Scope, Account: accomplice.Account, SignedIn: accomplice.SignedIn})
		}
	}

	return convicts
}

func (j *Jury) Attempted(click detect.Click) {
	if click.Scope == "" {
		return
	}

	for _, watchdog := range j.watchdogs {
		watchdog.Attempted(click)
	}
}

func (j *Jury) Committed(click detect.Click) {
	for _, watchdog := range j.watchdogs {
		watchdog.Committed(click)
	}
}

func (j *Jury) Flagged(ctx context.Context) (int, error) {
	return j.banner.Flagged(ctx) //nolint:wrapcheck // the banner already named what failed.
}

func (j *Jury) record(click detect.Click) {
	j.mu.Lock()
	defer j.mu.Unlock()

	c := j.callerLocked(click)

	if click.Account != "" {
		j.accounts[click.Account] = sighting{scope: click.Scope, at: click.At}
	}

	if j.outage.Across(c.lastSeen, click.At) {
		c.firstSeen = c.firstSeen.Add(j.outage.Length())
	}
	if gap := j.outage.Gap(c.lastSeen, click.At); gap > c.longestGap {
		c.longestGap = gap
	}
	c.lastSeen = click.At
	c.clicks++

	c.countCountry(click.Country)

	c.tiles = append(c.tiles, click.Tile)
	if len(c.tiles) > keptTiles {
		c.tiles = append(c.tiles[:0], c.tiles[len(c.tiles)-keptTiles:]...)
	}
}

func (j *Jury) opine(click detect.Click, watchdog string, verdict detect.Verdict, evidence detect.Evidence) []detect.Verdict {
	j.mu.Lock()
	defer j.mu.Unlock()

	c := j.callerLocked(click)
	c.opinions[watchdog] = detect.Opinion{
		Watchdog: watchdog,
		Verdict:  verdict,
		Evidence: evidence,
		At:       click.At,
	}

	reached, ok := c.reached[watchdog]
	if !ok {
		reached = new([detect.Certain + 1]time.Time)
		c.reached[watchdog] = reached
	}

	cutoff := click.At.Add(-j.rules().SuspicionWindow)

	var rises []detect.Verdict
	for level := detect.Suspect; level <= verdict; level++ {
		if reached[level].IsZero() || reached[level].Before(cutoff) {
			rises = append(rises, level)
		}
		reached[level] = click.At
	}

	return rises
}

func (j *Jury) deliberate(click detect.Click) (detect.Report, bool) {
	j.mu.Lock()
	defer j.mu.Unlock()

	c := j.callerLocked(click)

	opinions, _, guilty := j.weighLocked(c, click.At, j.rules())
	if !guilty {
		return detect.Report{}, false
	}

	country, countryClicks := c.topCountry()

	return detect.Report{
		Scope:            click.Scope,
		Account:          click.Account,
		Opinions:         opinions,
		Clicks:           c.clicks,
		ActiveFor:        click.At.Sub(c.firstSeen),
		LongestGap:       c.longestGap,
		TopCountry:       country,
		TopCountryClicks: countryClicks,
		Tiles:            append([]uint32(nil), c.tiles...),
	}, true
}

func (j *Jury) weighLocked(c *caller, at time.Time, rules Rules) ([]detect.Opinion, int, bool) {
	cutoff := at.Add(-rules.SuspicionWindow)

	var (
		certain  bool
		zeal     bool
		suspects int
		opinions = make([]detect.Opinion, 0, len(j.watchdogs))
	)

	for _, watchdog := range j.watchdogs {
		opinion, ok := c.opinions[watchdog.Name()]
		if !ok {
			opinion = detect.Opinion{Watchdog: watchdog.Name()}
		}

		if opinion.At.Before(cutoff) || !watchdog.Enabled() {
			opinion.Verdict = detect.Clear
		}

		switch {
		case opinion.Evidence.Turns:
		case opinion.Verdict == detect.Certain:
			certain = true
			suspects++
		case opinion.Verdict == detect.Suspect && opinion.Evidence.Zeal:
			zeal = true
		case opinion.Verdict == detect.Suspect:
			suspects++
		}

		opinions = append(opinions, opinion)
	}

	if zeal {
		suspects++
	}

	return opinions, suspects, certain || suspects >= rules.MinSuspects
}

func (j *Jury) Examine(scope, account string) detect.Examination {
	now := j.clock.Now()
	rules := j.rules()

	j.mu.Lock()
	defer j.mu.Unlock()

	if scope == "" {
		scope = j.accounts[account].scope
	}

	examination := detect.Examination{Scope: scope, Account: account, MinSuspects: rules.MinSuspects}

	c, tracked := j.callers[scope]
	if !tracked {
		c = &caller{}
	}

	opinions, suspects, guilty := j.weighLocked(c, now, rules)

	examination.Readings = make([]detect.Reading, 0, len(opinions))
	for _, opinion := range opinions {
		examination.Readings = append(examination.Readings, opinion.Reading())
	}

	if !tracked {
		return examination
	}

	country, countryClicks := c.topCountry()

	examination.Tracked = true
	examination.Suspects = suspects
	examination.Guilty = guilty
	examination.Clicks = c.clicks
	examination.ActiveFor = c.lastSeen.Sub(c.firstSeen)
	examination.LongestGap = c.longestGap
	examination.LastClickAt = c.lastSeen
	examination.TopCountry = country
	examination.TopCountryClicks = countryClicks

	return examination
}

func (j *Jury) callerLocked(click detect.Click) *caller {
	c, ok := j.callers[click.Scope]
	if !ok {
		c = &caller{
			firstSeen: click.At,
			lastSeen:  click.At,
			countries: make(map[string]int),
			opinions:  make(map[string]detect.Opinion),
			reached:   make(map[string]*[detect.Certain + 1]time.Time),
		}
		j.callers[click.Scope] = c
	}
	return c
}

func (c *caller) countCountry(country string) {
	if country == "" {
		return
	}
	if _, known := c.countries[country]; !known && len(c.countries) >= maxTrackedCountries {
		return
	}
	c.countries[country]++
}

func (c *caller) topCountry() (string, int) {
	var (
		top   string
		count int
	)

	for country, n := range c.countries {
		if n > count || (n == count && country < top) {
			top, count = country, n
		}
	}

	return top, count
}

func (j *Jury) Run(ctx context.Context) {
	ticker := time.NewTicker(j.config.SweepInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			j.sweep()
		case <-ctx.Done():
			return
		}
	}
}

func (j *Jury) sweep() {
	now := j.clock.Now()

	standing := j.forget(now)

	if j.hooks.OnStanding == nil {
		return
	}
	for i, watchdog := range j.watchdogs {
		for level := detect.Suspect; level <= detect.Certain; level++ {
			j.hooks.OnStanding(watchdog.Name(), level, standing[i][level])
		}
	}
}

func (j *Jury) forget(now time.Time) [][detect.Certain + 1]int {
	j.mu.Lock()
	defer j.mu.Unlock()

	trackCutoff := now.Add(-j.config.TrackWindow)
	suspicionCutoff := now.Add(-j.rules().SuspicionWindow)

	standing := make([][detect.Certain + 1]int, len(j.watchdogs))

	for scope, c := range j.callers {
		if c.lastSeen.Before(trackCutoff) {
			delete(j.callers, scope)
			continue
		}

		for i, watchdog := range j.watchdogs {
			opinion, ok := c.opinions[watchdog.Name()]
			if !ok || opinion.At.Before(suspicionCutoff) || !watchdog.Enabled() {
				continue
			}
			for level := detect.Suspect; level <= opinion.Verdict; level++ {
				standing[i][level]++
			}
		}
	}

	return standing
}
