package cohort

import (
	"context"
	"errors"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const Name = "cohort"

type Rules struct {
	Enabled bool

	StartWindow time.Duration

	MinClicks    int
	MinFlagShare float64

	RateRatio   float64
	LengthRatio float64
	QuietAfter  time.Duration

	MinMembers int

	CertainCohorts int
	CertainMembers int
	ChainWindow    time.Duration
}

func (r Rules) Validate() error {
	return errors.Join(
		detect.Positive("startWindow", r.StartWindow),
		detect.AtLeast("minClicks", r.MinClicks, "the two clicks a rate needs", 2),
		detect.Positive("minFlagShare", r.MinFlagShare),
		detect.Share("minFlagShare", r.MinFlagShare),
		detect.AtLeast("rateRatio", r.RateRatio, "faster over slower, which is never under", 1),
		detect.AtLeast("lengthRatio", r.LengthRatio, "longer over shorter, which is never under", 1),
		detect.Positive("quietAfter", r.QuietAfter),
		detect.AtLeast("minMembers", r.MinMembers, "the scopes a cohort is at least", 2),
		detect.AtLeast("certainCohorts", r.CertainCohorts, "the cohorts a chain is at least", 2),
		detect.AtLeast("certainMembers", r.CertainMembers, "minMembers", r.MinMembers),
		detect.Positive("chainWindow", r.ChainWindow),
	)
}

type Config struct {
	// Read when a scope is first seen, to index it by its wider prefix.
	V4Bits int
	V6Bits int

	// Raised to the rules' chainWindow if shorter.
	TrackWindow time.Duration

	SweepInterval time.Duration
}

const (
	defaultV4Bits        = 24
	defaultV6Bits        = 44
	defaultSweepInterval = time.Minute

	rejudgeAfter = 2 * time.Second

	maxTrackedCountries = 16
)

func (c Config) Validate() error {
	return errors.Join(
		detect.NotNegative("trackWindow", c.TrackWindow),
		detect.NotNegative("sweepInterval", c.SweepInterval),
		detect.NotNegative("v4Bits", c.V4Bits),
		detect.AtMost("v4Bits", c.V4Bits, "the bits of an IPv4 address", 32),
		detect.NotNegative("v6Bits", c.V6Bits),
		detect.AtMost("v6Bits", c.V6Bits, "the bits of a scope, a /64,", 64),
	)
}

func (c Config) withDefaults() Config {
	if c.V4Bits <= 0 {
		c.V4Bits = defaultV4Bits
	}
	if c.V6Bits <= 0 {
		c.V6Bits = defaultV6Bits
	}
	if c.SweepInterval <= 0 {
		c.SweepInterval = defaultSweepInterval
	}
	return c
}

func New(config Config, rules func() Rules, clock cptime.Clock, onInStep func(scopes int)) *Watchdog {
	if clock == nil {
		clock = cptime.SystemClock{}
	}

	return &Watchdog{
		config:   config.withDefaults(),
		rules:    rules,
		clock:    clock,
		onInStep: onInStep,
		members:  make(map[string]*member),
		starts:   make(map[int64]*cpcolls.Set[string]),
		prefixes: make(map[string]*cpcolls.Set[string]),
	}
}

type Watchdog struct {
	config   Config
	rules    func() Rules
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

func (w *Watchdog) Enabled() bool { return w.rules().Enabled }

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
	m.verdict, m.evidence, m.inStep = w.judgeLocked(m, click.At, w.rules())

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

func (w *Watchdog) judgeLocked(m *member, now time.Time, rules Rules) (detect.Verdict, detect.Evidence, bool) {
	flag, ok := rules.painting(m)
	if !ok {
		return detect.Clear, detect.Evidence{}, false
	}

	cohort := w.partnersLocked(m, flag, now, rules)
	if len(cohort) < rules.MinMembers {
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

	if sharing >= rules.CertainMembers {
		return detect.Certain, detect.Evidence{
			Rule:   "crowd",
			Fields: append(fields, detect.Field{Key: "prefix", Value: m.prefix}),
		}, true
	}

	if sharing >= rules.MinMembers {
		if cohorts := w.chainLocked(m, flag, now, rules); cohorts >= rules.CertainCohorts {
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

func (w *Watchdog) partnersLocked(m *member, flag string, now time.Time, rules Rules) []*member {
	cohort := []*member{m}

	from := m.first.Add(-rules.StartWindow).Unix()
	to := m.first.Add(rules.StartWindow).Unix()

	for second := from; second <= to; second++ {
		w.starts[second].ForEach(func(scope string) {
			other := w.members[scope]
			if other != m && rules.inStep(m, other, flag, now) {
				cohort = append(cohort, other)
			}
		})
	}

	return cohort
}

func (w *Watchdog) chainLocked(m *member, flag string, now time.Time, rules Rules) int {
	var group []*member

	w.prefixes[m.prefix].ForEach(func(scope string) {
		other := w.members[scope]
		if f, ok := rules.painting(other); !ok || f != flag {
			return
		}
		if absDuration(other.first.Sub(m.first)) > rules.ChainWindow {
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
		for end < len(group) && group[end].first.Sub(group[start].first) <= rules.StartWindow {
			end++
		}

		cluster := group[start:end]
		inStep := 0
		for _, a := range cluster {
			for _, b := range cluster {
				if a != b && rules.inStep(a, b, flag, now) {
					inStep++
					break
				}
			}
		}
		if inStep >= rules.MinMembers {
			cohorts++
		}

		start = end
	}

	return cohorts
}

func (r Rules) inStep(a, b *member, flag string, now time.Time) bool {
	if f, ok := r.painting(b); !ok || f != flag {
		return false
	}
	if absDuration(a.first.Sub(b.first)) > r.StartWindow {
		return false
	}

	if ratio(rate(a), rate(b)) > r.RateRatio {
		return false
	}

	shorter, longer := a, b
	if longer.last.Sub(longer.first) < shorter.last.Sub(shorter.first) {
		shorter, longer = longer, shorter
	}

	if now.Sub(shorter.last) >= r.QuietAfter {
		return ratio(float64(shorter.last.Sub(shorter.first)), float64(longer.last.Sub(longer.first))) <= r.LengthRatio
	}

	return true
}

func (r Rules) painting(m *member) (string, bool) {
	if m.clicks < r.MinClicks || !m.last.After(m.first) {
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

	if top == "" || float64(count)/float64(m.clicks) < r.MinFlagShare {
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

	rules := w.rules()
	inStep := 0
	cutoff := now.Add(-max(w.config.TrackWindow, rules.ChainWindow))

	for scope, m := range w.members {
		if m.last.Before(cutoff) {
			delete(w.members, scope)
			unindex(w.starts, m.first.Unix(), scope)
			if m.prefix != "" {
				unindex(w.prefixes, m.prefix, scope)
			}
			continue
		}

		if m.inStep && now.Sub(m.last) < rules.QuietAfter {
			inStep++
		}
	}

	w.mu.Unlock()

	if w.onInStep != nil {
		w.onInStep(inStep)
	}
}
