package cohort

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const Name = "cohort"

type Config struct {
	StartWindow time.Duration

	MinClicks    int
	MinFlagShare float64

	RateRatio   float64
	LengthRatio float64
	QuietAfter  time.Duration

	MinMembers int

	V4Bits int
	V6Bits int

	CertainCohorts int
	CertainMembers int
	ChainWindow    time.Duration

	TrackWindow time.Duration

	SweepInterval time.Duration
}

const (
	defaultStartWindow    = 5 * time.Second
	defaultMinClicks      = 20
	defaultMinFlagShare   = 0.9
	defaultRateRatio      = 1.5
	defaultLengthRatio    = 1.2
	defaultQuietAfter     = time.Minute
	defaultMinMembers     = 2
	defaultV4Bits         = 24
	defaultV6Bits         = 44
	defaultCertainCohorts = 3
	defaultCertainMembers = 6
	defaultChainWindow    = 30 * time.Minute
	defaultSweepInterval  = time.Minute

	rejudgeAfter = 2 * time.Second

	maxTrackedCountries = 16
)

func (c Config) Validate() error {
	var errs []error

	for _, d := range []struct {
		name  string
		value time.Duration
	}{
		{"startWindow", c.StartWindow},
		{"quietAfter", c.QuietAfter},
		{"chainWindow", c.ChainWindow},
		{"trackWindow", c.TrackWindow},
		{"sweepInterval", c.SweepInterval},
	} {
		if d.value < 0 {
			errs = append(errs, fmt.Errorf("%s is %v: a duration must not be negative", d.name, d.value))
		}
	}

	if c.MinClicks < 0 || c.MinClicks == 1 {
		errs = append(errs, fmt.Errorf("minClicks is %d: a rate needs at least 2 clicks", c.MinClicks))
	}
	if c.MinFlagShare < 0 || c.MinFlagShare > 1 || math.IsNaN(c.MinFlagShare) {
		errs = append(errs, fmt.Errorf("minFlagShare is %v: a share is between 0 and 1", c.MinFlagShare))
	}
	if c.RateRatio != 0 && (c.RateRatio < 1 || math.IsNaN(c.RateRatio)) {
		errs = append(errs, fmt.Errorf("rateRatio is %v: faster over slower is never under 1", c.RateRatio))
	}
	if c.LengthRatio != 0 && (c.LengthRatio < 1 || math.IsNaN(c.LengthRatio)) {
		errs = append(errs, fmt.Errorf("lengthRatio is %v: longer over shorter is never under 1", c.LengthRatio))
	}
	if c.MinMembers < 0 || c.MinMembers == 1 {
		errs = append(errs, fmt.Errorf("minMembers is %d: a cohort is at least 2 scopes", c.MinMembers))
	}
	if c.CertainCohorts < 0 || c.CertainCohorts == 1 {
		errs = append(errs, fmt.Errorf("certainCohorts is %d: one cohort is a suspicion, a chain is at least 2", c.CertainCohorts))
	}
	if c.CertainMembers < 0 {
		errs = append(errs, fmt.Errorf("certainMembers is %d: it must not be negative", c.CertainMembers))
	}
	if c.V4Bits < 0 || c.V4Bits > 32 {
		errs = append(errs, fmt.Errorf("v4Bits is %d: an IPv4 prefix is 1 to 32 bits", c.V4Bits))
	}
	if c.V6Bits < 0 || c.V6Bits > 64 {
		errs = append(errs, fmt.Errorf("v6Bits is %d: a scope is a /64, so the prefix is 1 to 64 bits", c.V6Bits))
	}

	return errors.Join(errs...)
}

func (c Config) withDefaults() Config {
	if c.StartWindow <= 0 {
		c.StartWindow = defaultStartWindow
	}
	if c.MinClicks <= 0 {
		c.MinClicks = defaultMinClicks
	}
	if c.MinFlagShare <= 0 {
		c.MinFlagShare = defaultMinFlagShare
	}
	if c.RateRatio < 1 {
		c.RateRatio = defaultRateRatio
	}
	if c.LengthRatio < 1 {
		c.LengthRatio = defaultLengthRatio
	}
	if c.QuietAfter <= 0 {
		c.QuietAfter = defaultQuietAfter
	}
	if c.MinMembers < 2 {
		c.MinMembers = defaultMinMembers
	}
	if c.V4Bits <= 0 {
		c.V4Bits = defaultV4Bits
	}
	if c.V6Bits <= 0 {
		c.V6Bits = defaultV6Bits
	}
	if c.CertainCohorts < 2 {
		c.CertainCohorts = defaultCertainCohorts
	}
	if c.CertainMembers <= 0 {
		c.CertainMembers = defaultCertainMembers
	}
	if c.CertainMembers < c.MinMembers {
		c.CertainMembers = c.MinMembers
	}
	if c.ChainWindow <= 0 {
		c.ChainWindow = defaultChainWindow
	}
	if c.TrackWindow < c.ChainWindow {
		c.TrackWindow = c.ChainWindow
	}
	if c.SweepInterval <= 0 {
		c.SweepInterval = defaultSweepInterval
	}
	return c
}

func New(config Config, clock cptime.Clock, onInStep func(scopes int)) *Watchdog {
	if clock == nil {
		clock = cptime.SystemClock{}
	}

	return &Watchdog{
		config:   config.withDefaults(),
		clock:    clock,
		onInStep: onInStep,
		members:  make(map[string]*member),
		starts:   make(map[int64]*cpcolls.Set[string]),
		prefixes: make(map[string]*cpcolls.Set[string]),
	}
}

type Watchdog struct {
	config   Config
	clock    cptime.Clock
	onInStep func(int)

	mu      sync.Mutex
	members map[string]*member

	starts   map[int64]*cpcolls.Set[string]
	prefixes map[string]*cpcolls.Set[string]
}

var _ detect.Watchdog = (*Watchdog)(nil)

type member struct {
	scope  string
	prefix string

	first  time.Time
	last   time.Time
	clicks int

	countries map[string]int

	judgedAt time.Time
	verdict  detect.Verdict
	evidence detect.Evidence
	inStep   bool
}

func (w *Watchdog) Name() string { return Name }

func (w *Watchdog) Attempted(click detect.Click) {
	if click.Scope == "" {
		return
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	w.memberLocked(click)
}

func (w *Watchdog) Committed(detect.Click) {}

func (w *Watchdog) Watch(click detect.Click) (detect.Verdict, detect.Evidence) {
	if click.Scope == "" {
		return detect.Clear, detect.Evidence{}
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	m := w.memberLocked(click)
	if click.At.After(m.last) {
		m.last = click.At
	}
	m.clicks++
	if _, known := m.countries[click.Country]; known || len(m.countries) < maxTrackedCountries {
		m.countries[click.Country]++
	}

	if !m.judgedAt.IsZero() && click.At.Sub(m.judgedAt) < rejudgeAfter {
		return m.verdict, m.evidence
	}

	m.judgedAt = click.At
	m.verdict, m.evidence, m.inStep = w.judgeLocked(m, click.At)

	return m.verdict, m.evidence
}

func (w *Watchdog) memberLocked(click detect.Click) *member {
	m, ok := w.members[click.Scope]
	if ok {
		return m
	}

	m = &member{
		scope:     click.Scope,
		prefix:    detect.WiderPrefix(click.Scope, w.config.V4Bits, w.config.V6Bits),
		first:     click.At,
		last:      click.At,
		countries: make(map[string]int),
	}
	w.members[click.Scope] = m

	index(w.starts, m.first.Unix(), m.scope)
	if m.prefix != "" {
		index(w.prefixes, m.prefix, m.scope)
	}

	return m
}

func (w *Watchdog) judgeLocked(m *member, now time.Time) (detect.Verdict, detect.Evidence, bool) {
	flag, ok := w.painting(m)
	if !ok {
		return detect.Clear, detect.Evidence{}, false
	}

	cohort := w.partnersLocked(m, flag, now)
	if len(cohort) < w.config.MinMembers {
		return detect.Clear, detect.Evidence{}, len(cohort) > 1
	}

	fields := []detect.Field{
		{Key: "members", Value: len(cohort)},
		{Key: "flag", Value: flag},
		{Key: "startSpread", Value: startSpread(cohort)},
		{Key: "perMinute", Value: math.Round(rate(m)*600) / 10},
	}

	sharing := 0
	for _, other := range cohort {
		if m.prefix != "" && other.prefix == m.prefix {
			sharing++
		}
	}

	if sharing >= w.config.CertainMembers {
		return detect.Certain, detect.Evidence{
			Rule:   "crowd",
			Fields: append(fields, detect.Field{Key: "prefix", Value: m.prefix}),
		}, true
	}

	if sharing >= w.config.MinMembers {
		if cohorts := w.chainLocked(m, flag, now); cohorts >= w.config.CertainCohorts {
			return detect.Certain, detect.Evidence{
				Rule: "chain",
				Fields: append(fields,
					detect.Field{Key: "cohorts", Value: cohorts},
					detect.Field{Key: "prefix", Value: m.prefix},
				),
			}, true
		}
	}

	return detect.Suspect, detect.Evidence{Rule: "lockstep", Fields: fields}, true
}

func (w *Watchdog) partnersLocked(m *member, flag string, now time.Time) []*member {
	cohort := []*member{m}

	from := m.first.Add(-w.config.StartWindow).Unix()
	to := m.first.Add(w.config.StartWindow).Unix()

	for second := from; second <= to; second++ {
		w.starts[second].ForEach(func(scope string) {
			other := w.members[scope]
			if other != m && w.inStep(m, other, flag, now) {
				cohort = append(cohort, other)
			}
		})
	}

	return cohort
}

func (w *Watchdog) chainLocked(m *member, flag string, now time.Time) int {
	var group []*member

	w.prefixes[m.prefix].ForEach(func(scope string) {
		other := w.members[scope]
		if f, ok := w.painting(other); !ok || f != flag {
			return
		}
		if absDuration(other.first.Sub(m.first)) > w.config.ChainWindow {
			return
		}
		group = append(group, other)
	})

	sort.Slice(group, func(i, j int) bool {
		if group[i].first.Equal(group[j].first) {
			return group[i].scope < group[j].scope
		}
		return group[i].first.Before(group[j].first)
	})

	cohorts := 0

	for start := 0; start < len(group); {
		end := start + 1
		for end < len(group) && group[end].first.Sub(group[start].first) <= w.config.StartWindow {
			end++
		}

		cluster := group[start:end]
		inStep := 0
		for _, a := range cluster {
			for _, b := range cluster {
				if a != b && w.inStep(a, b, flag, now) {
					inStep++
					break
				}
			}
		}
		if inStep >= w.config.MinMembers {
			cohorts++
		}

		start = end
	}

	return cohorts
}

func (w *Watchdog) inStep(a, b *member, flag string, now time.Time) bool {
	if f, ok := w.painting(b); !ok || f != flag {
		return false
	}
	if absDuration(a.first.Sub(b.first)) > w.config.StartWindow {
		return false
	}

	if ratio(rate(a), rate(b)) > w.config.RateRatio {
		return false
	}

	shorter, longer := a, b
	if longer.last.Sub(longer.first) < shorter.last.Sub(shorter.first) {
		shorter, longer = longer, shorter
	}

	if now.Sub(shorter.last) >= w.config.QuietAfter {
		return ratio(float64(shorter.last.Sub(shorter.first)), float64(longer.last.Sub(longer.first))) <= w.config.LengthRatio
	}

	return true
}

func (w *Watchdog) painting(m *member) (string, bool) {
	if m.clicks < w.config.MinClicks || !m.last.After(m.first) {
		return "", false
	}

	var (
		top   string
		count int
	)
	for country, n := range m.countries {
		if n > count || (n == count && country < top) {
			top, count = country, n
		}
	}

	if top == "" || float64(count)/float64(m.clicks) < w.config.MinFlagShare {
		return "", false
	}
	return top, true
}

func rate(m *member) float64 {
	span := m.last.Sub(m.first).Seconds()
	if span <= 0 {
		return 0
	}
	return float64(m.clicks-1) / span
}

func startSpread(cohort []*member) time.Duration {
	earliest, latest := cohort[0].first, cohort[0].first
	for _, m := range cohort[1:] {
		if m.first.Before(earliest) {
			earliest = m.first
		}
		if m.first.After(latest) {
			latest = m.first
		}
	}
	return latest.Sub(earliest)
}

func ratio(a, b float64) float64 {
	if a < b {
		a, b = b, a
	}
	if b <= 0 {
		return math.Inf(1)
	}
	return a / b
}

func absDuration(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}

func index[K comparable](buckets map[K]*cpcolls.Set[string], key K, scope string) {
	bucket, ok := buckets[key]
	if !ok {
		bucket = cpcolls.NewSet[string]()
		buckets[key] = bucket
	}
	bucket.Add(scope)
}

func unindex[K comparable](buckets map[K]*cpcolls.Set[string], key K, scope string) {
	bucket, ok := buckets[key]
	if !ok {
		return
	}
	bucket.Delete(scope)
	if bucket.Empty() {
		delete(buckets, key)
	}
}

func (w *Watchdog) Run(ctx context.Context) {
	ticker := time.NewTicker(w.config.SweepInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.sweep()
		case <-ctx.Done():
			return
		}
	}
}

func (w *Watchdog) sweep() {
	now := w.clock.Now()

	w.mu.Lock()

	inStep := 0
	cutoff := now.Add(-w.config.TrackWindow)

	for scope, m := range w.members {
		if m.last.Before(cutoff) {
			delete(w.members, scope)
			unindex(w.starts, m.first.Unix(), scope)
			if m.prefix != "" {
				unindex(w.prefixes, m.prefix, scope)
			}
			continue
		}

		if m.inStep && now.Sub(m.last) < w.config.QuietAfter {
			inStep++
		}
	}

	w.mu.Unlock()

	if w.onInStep != nil {
		w.onInStep(inStep)
	}
}
