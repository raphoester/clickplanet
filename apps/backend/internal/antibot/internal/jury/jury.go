// Package jury crosses what the watchdogs say and passes the sentence.
package jury

import (
	"context"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/shadowban"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Config struct {
	// MinSuspects is how many watchdogs must say Suspect at once before a
	// caller is banned on suspicion alone. One Certain always bans by itself.
	//
	// Crossing only buys confidence when the watchdogs measure different
	// things, and these barely do: a machine sweep is sequential *and* regular
	// *and* never rests, so two Suspects can be one behaviour counted twice.
	// That is also true of the most obsessed player. Two is the loosest this
	// should be.
	MinSuspects int

	// SuspicionWindow is how long a Suspect stands while another watchdog
	// catches up. A caller's rules do not all trip on the same click: the
	// sequencer needs a run of steps, the metronome needs a quarter of an hour.
	SuspicionWindow time.Duration

	// TrackWindow is how long a silent caller is remembered.
	TrackWindow time.Duration

	SweepInterval time.Duration
}

const (
	defaultMinSuspects     = 2
	defaultSuspicionWindow = 10 * time.Minute
	defaultTrackWindow     = 15 * time.Minute
	defaultSweepInterval   = time.Minute

	// Caps the country tally: real players use a handful, and without a bound a
	// client could spend memory by cycling through every valid code.
	maxTrackedCountries = 16
	keptTiles           = 8
)

// WithDefaults fills the unset bounds. Exported inside this tree so the caller
// can log the numbers that will actually be enforced rather than the raw file.
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

// Banner is the sentence the jury passes. shadowban.Banner implements it.
type Banner interface {
	Flag(scope string) (shadowban.Sentence, bool)
	Banned(scope string) bool
	Flagged() int
}

// Hooks is how the jury reports what it sees short of a ban. Every hook is optional.
type Hooks struct {
	OnFlag func(detect.Report)

	// OnRise is a watchdog's reading of a caller reaching a level it has not held
	// inside SuspicionWindow. Levels are cumulative, so a caller going straight to
	// Certain rises to Suspect too, and a reading flapping across a bound counts
	// once a window rather than once a click.
	OnRise func(watchdog string, level detect.Verdict)

	// OnStanding is, once a sweep, how many callers each watchdog reads at the
	// level or above — the readings the jury would count if it deliberated now.
	// Reported for every watchdog and level, zero included, so a gauge falls back.
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
}

type caller struct {
	firstSeen  time.Time
	lastSeen   time.Time
	longestGap time.Duration

	clicks    int
	countries map[string]int
	tiles     []uint32

	opinions map[string]detect.Opinion

	// reached is, per watchdog and indexed by level, when this caller was last
	// read at that level or above: what tells a rise from a reading still standing.
	reached map[string]*[detect.Certain + 1]time.Time
}

// Inspect runs every watchdog over the click and says whether it should be
// dropped.
func (j *Jury) Inspect(click detect.Click) bool {
	if click.Scope == "" {
		return false
	}

	j.record(click)

	// Every watchdog sees every click, and the ban is read afterwards rather
	// than short-circuiting here. A watchdog cut off the moment another one
	// banned the caller would be judging a caller that appears to have stopped.
	for _, watchdog := range j.watchdogs {
		verdict, evidence := watchdog.Watch(click)
		for _, level := range j.opine(click, watchdog.Name(), verdict, evidence) {
			if j.hooks.OnRise != nil {
				j.hooks.OnRise(watchdog.Name(), level)
			}
		}
	}

	// Outside the lock from here: onFlag writes a log line, and holding the
	// caller map through that would queue every other clicker behind the I/O.
	if report, guilty := j.deliberate(click); guilty {
		if sentence, accepted := j.banner.Flag(click.Scope); accepted {
			report.Flags = sentence.Flags
			report.Offence = sentence.Offence
			report.BannedUntil = sentence.Until
			if j.hooks.OnFlag != nil {
				j.hooks.OnFlag(report)
			}
		}
	}

	return j.banner.Banned(click.Scope)
}

// Attempted hands the watchdogs a click before the throttle judges it.
func (j *Jury) Attempted(click detect.Click) {
	if click.Scope == "" {
		return
	}

	for _, watchdog := range j.watchdogs {
		watchdog.Attempted(click)
	}
}

// Committed tells the watchdogs the click reached the map. Watchdogs that read
// the map decide for themselves what that is worth.
func (j *Jury) Committed(click detect.Click) {
	for _, watchdog := range j.watchdogs {
		watchdog.Committed(click)
	}
}

func (j *Jury) Flagged() int { return j.banner.Flagged() }

func (j *Jury) record(click detect.Click) {
	j.mu.Lock()
	defer j.mu.Unlock()

	c := j.callerLocked(click)

	if gap := click.At.Sub(c.lastSeen); gap > c.longestGap {
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

// opine stores the watchdog's reading and returns the levels it rose to.
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

	// The same window deliberate expires a reading on: a level last held longer
	// ago than that is no longer standing, so reaching it again is a rise.
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
		Opinions:         opinions,
		Clicks:           c.clicks,
		ActiveFor:        click.At.Sub(c.firstSeen),
		LongestGap:       c.longestGap,
		TopCountry:       country,
		TopCountryClicks: countryClicks,
		Tiles:            append([]uint32(nil), c.tiles...),
	}, true
}

// weighLocked is the decision, shared by a click that may ban and an operator who only asks.
func (j *Jury) weighLocked(c *caller, at time.Time) ([]detect.Opinion, int, bool) {
	cutoff := at.Add(-j.config.SuspicionWindow)

	var (
		certain  bool
		suspects int
		opinions = make([]detect.Opinion, 0, len(j.watchdogs))
	)

	for _, watchdog := range j.watchdogs {
		// Missing only for a caller never seen, which an examination still lists as clear.
		opinion, ok := c.opinions[watchdog.Name()]
		if !ok {
			opinion = detect.Opinion{Watchdog: watchdog.Name()}
		}

		// A verdict older than the window is not evidence any more, but it is
		// still worth printing next to the one that banned the caller.
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

// Examine reads what the jury holds on a scope as of now; it creates no caller and passes no ban.
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
		// Ties break on the code, so the same tally always names the same country.
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

// sweep forgets callers that can no longer affect a decision; without it the
// map keeps an entry for every caller that ever clicked. A ban outlives its
// caller record on purpose: the ban lives in the banner, which has its own
// clock, so forgetting the evidence here never shortens a sentence.
func (j *Jury) sweep() {
	now := j.clock.Now()

	standing := j.forget(now)

	// Outside the lock, for the same reason as onFlag.
	if j.hooks.OnStanding == nil {
		return
	}
	for i, watchdog := range j.watchdogs {
		for level := detect.Suspect; level <= detect.Certain; level++ {
			j.hooks.OnStanding(watchdog.Name(), level, standing[i][level])
		}
	}
}

// forget drops idle callers and counts, per watchdog, the callers still read at
// each level or above.
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
