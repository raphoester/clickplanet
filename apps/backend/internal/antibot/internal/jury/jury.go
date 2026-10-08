package jury

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/shadowban"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Config struct {
	MinSuspects int

	SuspicionWindow time.Duration

	TrackWindow time.Duration

	SweepInterval time.Duration
}

const (
	defaultMinSuspects     = 2
	defaultSuspicionWindow = 10 * time.Minute
	defaultTrackWindow     = 15 * time.Minute
	defaultSweepInterval   = time.Minute

	// Bounds memory: a client could cycle through every country code.
	maxTrackedCountries = 16
	keptTiles           = 8
)

func (c Config) WithDefaults() Config {
	if c.MinSuspects <= 0 {
		c.MinSuspects = defaultMinSuspects
	}
	if c.SuspicionWindow <= 0 {
		c.SuspicionWindow = defaultSuspicionWindow
	}
	if c.TrackWindow <= 0 {
		c.TrackWindow = defaultTrackWindow
	}
	if c.SweepInterval <= 0 {
		c.SweepInterval = defaultSweepInterval
	}
	return c
}

type Banner interface {
	Flag(ctx context.Context, caller shadowban.Caller) (shadowban.Sentence, bool, error)
	Banned(ctx context.Context, caller shadowban.Caller) (bool, error)
	Flagged(ctx context.Context) (int, error)
}

type Hooks struct {
	OnFlag func(detect.Report)

	OnRise func(watchdog string, level detect.Verdict)

	OnStanding func(watchdog string, level detect.Verdict, callers int)
}

func New(
	config Config,
	banner Banner,
	clock cptime.Clock,
	hooks Hooks,
	watchdogs ...detect.Watchdog,
) *Jury {
	if clock == nil {
		clock = cptime.SystemClock{}
	}

	return &Jury{
		config:    config.WithDefaults(),
		banner:    banner,
		clock:     clock,
		hooks:     hooks,
		watchdogs: watchdogs,
		callers:   make(map[string]*caller),
	}
}

type Jury struct {
	config    Config
	banner    Banner
	clock     cptime.Clock
	hooks     Hooks
	watchdogs []detect.Watchdog

	mu      sync.Mutex
	callers map[string]*caller
	outage  detect.Outage
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

// A ban that cannot be read is no ban: the click goes through, and the error says why.
func (j *Jury) Inspect(ctx context.Context, click detect.Click) (bool, error) {
	if click.Scope == "" {
		return false, nil
	}

	j.record(click)

	// No short-circuit on a ban: a starved watchdog lets the ban lapse on fake silence.
	for _, watchdog := range j.watchdogs {
		verdict, evidence := watchdog.Watch(click)
		for _, level := range j.opine(click, watchdog.Name(), verdict, evidence) {
			if j.hooks.OnRise != nil {
				j.hooks.OnRise(watchdog.Name(), level)
			}
		}
	}

	var flagErr error
	if report, guilty := j.deliberate(click); guilty {
		sentence, accepted, err := j.banner.Flag(ctx, callerOf(click))
		flagErr = err
		if accepted {
			report.Flags = sentence.Flags
			report.Offence = sentence.Offence
			report.BannedUntil = sentence.Until
			if j.hooks.OnFlag != nil {
				j.hooks.OnFlag(report)
			}
		}
	}

	banned, err := j.banner.Banned(ctx, callerOf(click))
	return banned, errors.Join(flagErr, err)
}

func callerOf(click detect.Click) shadowban.Caller {
	return shadowban.Caller{Scope: click.Scope, Account: click.Account, SignedIn: click.SignedIn}
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

	cutoff := click.At.Add(-j.config.SuspicionWindow)

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

	opinions, _, guilty := j.weighLocked(c, click.At)
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

func (j *Jury) weighLocked(c *caller, at time.Time) ([]detect.Opinion, int, bool) {
	cutoff := at.Add(-j.config.SuspicionWindow)

	var (
		certain  bool
		suspects int
		opinions = make([]detect.Opinion, 0, len(j.watchdogs))
	)

	for _, watchdog := range j.watchdogs {
		opinion, ok := c.opinions[watchdog.Name()]
		if !ok {
			opinion = detect.Opinion{Watchdog: watchdog.Name()}
		}

		if opinion.At.Before(cutoff) {
			opinion.Verdict = detect.Clear
		}

		switch opinion.Verdict {
		case detect.Certain:
			certain = true
			suspects++
		case detect.Suspect:
			suspects++
		}

		opinions = append(opinions, opinion)
	}

	return opinions, suspects, certain || suspects >= j.config.MinSuspects
}

func (j *Jury) Examine(scope string) detect.Examination {
	now := j.clock.Now()

	j.mu.Lock()
	defer j.mu.Unlock()

	examination := detect.Examination{Scope: scope, MinSuspects: j.config.MinSuspects}

	c, tracked := j.callers[scope]
	if !tracked {
		c = &caller{}
	}

	opinions, suspects, guilty := j.weighLocked(c, now)

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
	suspicionCutoff := now.Add(-j.config.SuspicionWindow)

	standing := make([][detect.Certain + 1]int, len(j.watchdogs))

	for scope, c := range j.callers {
		if c.lastSeen.Before(trackCutoff) {
			delete(j.callers, scope)
			continue
		}

		for i, watchdog := range j.watchdogs {
			opinion, ok := c.opinions[watchdog.Name()]
			if !ok || opinion.At.Before(suspicionCutoff) {
				continue
			}
			for level := detect.Suspect; level <= opinion.Verdict; level++ {
				standing[i][level]++
			}
		}
	}

	return standing
}
