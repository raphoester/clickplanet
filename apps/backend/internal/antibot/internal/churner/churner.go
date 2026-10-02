package churner

import (
	"context"
	"net/netip"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const Name = "churner"

type Config struct {
	Window time.Duration

	V6 Bounds
	V4 Bounds

	Relay Relay

	TrackWindow   time.Duration
	SweepInterval time.Duration
}

type Bounds struct {
	MinAccounts     int
	CertainAccounts int
}

type Relay struct {
	Handoff time.Duration
	MaxLife time.Duration

	MinClicks    int
	MinFlagShare float64

	V4Bits int
	V6Bits int

	MinLinks     int
	CertainLinks int
}

const (
	defaultWindow        = time.Hour
	defaultHandoff       = 90 * time.Second
	defaultMaxLife       = 5 * time.Minute
	defaultMinClicks     = 20
	defaultMinFlagShare  = 0.9
	defaultV4Bits        = 24
	defaultV6Bits        = 32
	defaultSweepInterval = time.Minute

	rejudgeAfter        = 2 * time.Second
	maxTrackedCountries = 16
)

func (c Config) withDefaults() Config {
	if c.Window <= 0 {
		c.Window = defaultWindow
	}
	if c.Relay.Handoff <= 0 {
		c.Relay.Handoff = defaultHandoff
	}
	if c.Relay.MaxLife <= 0 {
		c.Relay.MaxLife = defaultMaxLife
	}
	if c.Relay.MinClicks < 2 {
		c.Relay.MinClicks = defaultMinClicks
	}
	if c.Relay.MinFlagShare <= 0 || c.Relay.MinFlagShare > 1 {
		c.Relay.MinFlagShare = defaultMinFlagShare
	}
	if c.Relay.V4Bits <= 0 || c.Relay.V4Bits > 32 {
		c.Relay.V4Bits = defaultV4Bits
	}
	if c.Relay.V6Bits <= 0 || c.Relay.V6Bits > 64 {
		c.Relay.V6Bits = defaultV6Bits
	}
	if minimum := c.Window + c.Relay.Handoff; c.TrackWindow < minimum {
		c.TrackWindow = minimum
	}
	if c.SweepInterval <= 0 {
		c.SweepInterval = defaultSweepInterval
	}
	return c
}

func New(config Config, clock cptime.Clock, onAccounts func(accounts int, family string), onLinks func(links int)) *Watchdog {
	return &Watchdog{
		config:     config.withDefaults(),
		clock:      clock,
		onAccounts: onAccounts,
		onLinks:    onLinks,
		accounts:   make(map[string]*account),
		born:       make(map[string]*cpcolls.Set[string]),
		prefixes:   make(map[string]*cpcolls.Set[string]),
	}
}

type Watchdog struct {
	config     Config
	clock      cptime.Clock
	onAccounts func(int, string)
	onLinks    func(int)

	mu       sync.Mutex
	accounts map[string]*account
	born     map[string]*cpcolls.Set[string]
	prefixes map[string]*cpcolls.Set[string]
	sweptAt  time.Time
}

var _ detect.Watchdog = (*Watchdog)(nil)

type account struct {
	id     string
	scope  string
	prefix string

	first     time.Time
	last      time.Time
	clicks    int
	countries map[string]int

	judgedAt time.Time
	verdict  detect.Verdict
	evidence detect.Evidence
}

func (w *Watchdog) Name() string { return Name }

func (w *Watchdog) Attempted(detect.Click) {}

func (w *Watchdog) Committed(detect.Click) {}

func (w *Watchdog) Watch(click detect.Click) (detect.Verdict, detect.Evidence) {
	if click.Scope == "" || click.Account == "" || click.SignedIn {
		return detect.Clear, detect.Evidence{}
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	a := w.recordLocked(click)

	churn, churnEvidence := w.churnLocked(click.Scope, click.At)

	if a.clicks >= w.config.Relay.MinClicks && (a.judgedAt.IsZero() || click.At.Sub(a.judgedAt) >= rejudgeAfter) {
		a.judgedAt = click.At
		a.verdict, a.evidence = w.relayLocked(a, click.At)
	}

	if a.verdict > churn {
		return a.verdict, a.evidence
	}
	return churn, churnEvidence
}

func (w *Watchdog) recordLocked(click detect.Click) *account {
	a, ok := w.accounts[click.Account]
	if !ok {
		a = &account{
			id:        click.Account,
			scope:     click.Scope,
			prefix:    detect.WiderPrefix(click.Scope, w.config.Relay.V4Bits, w.config.Relay.V6Bits),
			first:     click.At,
			last:      click.At,
			countries: make(map[string]int),
		}
		w.indexLocked(a)
	}

	if click.At.After(a.last) {
		a.last = click.At
	}
	a.clicks++
	if _, known := a.countries[click.Country]; known || len(a.countries) < maxTrackedCountries {
		a.countries[click.Country]++
	}

	return a
}

func (w *Watchdog) indexLocked(a *account) {
	w.accounts[a.id] = a
	index(w.born, a.scope, a.id)
	if a.prefix != "" {
		index(w.prefixes, a.prefix, a.id)
	}
}

func (w *Watchdog) dropLocked(a *account) {
	delete(w.accounts, a.id)
	unindex(w.born, a.scope, a.id)
	if a.prefix != "" {
		unindex(w.prefixes, a.prefix, a.id)
	}
}

func (w *Watchdog) churnLocked(scope string, now time.Time) (detect.Verdict, detect.Evidence) {
	accounts := w.bornLocked(scope, now)
	bounds, family := w.boundsOf(scope)

	verdict := level(accounts, bounds.MinAccounts, bounds.CertainAccounts)
	if verdict == detect.Clear {
		return detect.Clear, detect.Evidence{}
	}

	return verdict, detect.Evidence{Rule: "churn", Fields: []detect.Field{
		{Key: "accounts", Value: accounts},
		{Key: "family", Value: family},
	}}
}

func (w *Watchdog) bornLocked(scope string, now time.Time) int {
	from := now.Add(-w.config.Window)

	accounts := 0
	w.born[scope].ForEach(func(id string) {
		if !w.accounts[id].first.Before(from) {
			accounts++
		}
	})
	return accounts
}

func (w *Watchdog) boundsOf(scope string) (Bounds, string) {
	if v6(scope) {
		return w.config.V6, "v6"
	}
	return w.config.V4, "v4"
}

func v6(scope string) bool {
	if prefix, err := netip.ParsePrefix(scope); err == nil {
		return prefix.Addr().Is6() && !prefix.Addr().Is4In6()
	}
	if addr, err := netip.ParseAddr(scope); err == nil {
		return addr.Is6() && !addr.Is4In6()
	}
	return false
}

func (w *Watchdog) relayLocked(a *account, now time.Time) (detect.Verdict, detect.Evidence) {
	if a.prefix == "" {
		return detect.Clear, detect.Evidence{}
	}

	flag, ok := w.painting(a)
	if !ok {
		return detect.Clear, detect.Evidence{}
	}

	from, ok := w.predecessorLocked(a, flag)
	if !ok {
		return detect.Clear, detect.Evidence{}
	}

	links := w.linksLocked(a.prefix, flag, now)

	verdict := level(links, w.config.Relay.MinLinks, w.config.Relay.CertainLinks)
	if verdict == detect.Clear {
		return detect.Clear, detect.Evidence{}
	}

	return verdict, detect.Evidence{Rule: "relay", Fields: []detect.Field{
		{Key: "links", Value: links},
		{Key: "flag", Value: flag},
		{Key: "prefix", Value: a.prefix},
		{Key: "handoff", Value: a.first.Sub(from.last)},
	}}
}

func (w *Watchdog) predecessorLocked(a *account, flag string) (*account, bool) {
	var closest *account

	w.prefixes[a.prefix].ForEach(func(id string) {
		b := w.accounts[id]
		if b == a || !b.last.Before(a.first) || a.first.Sub(b.last) > w.config.Relay.Handoff {
			return
		}
		if b.last.Sub(b.first) > w.config.Relay.MaxLife {
			return
		}
		if f, ok := w.painting(b); !ok || f != flag {
			return
		}
		if closest == nil || b.last.After(closest.last) {
			closest = b
		}
	})

	return closest, closest != nil
}

func (w *Watchdog) linksLocked(prefix, flag string, now time.Time) int {
	from := now.Add(-w.config.Window)

	links := 0
	w.prefixes[prefix].ForEach(func(id string) {
		c := w.accounts[id]
		if c.first.Before(from) {
			return
		}
		if f, ok := w.painting(c); !ok || f != flag {
			return
		}
		if _, ok := w.predecessorLocked(c, flag); ok {
			links++
		}
	})
	return links
}

func (w *Watchdog) painting(a *account) (string, bool) {
	if a.clicks < w.config.Relay.MinClicks {
		return "", false
	}

	var (
		top   string
		count int
	)
	for country, n := range a.countries {
		if n > count || (n == count && country < top) {
			top, count = country, n
		}
	}

	if top == "" || float64(count)/float64(a.clicks) < w.config.Relay.MinFlagShare {
		return "", false
	}
	return top, true
}

func level(n, suspect, certain int) detect.Verdict {
	switch {
	case certain > 0 && n >= certain:
		return detect.Certain
	case suspect > 0 && n >= suspect:
		return detect.Suspect
	default:
		return detect.Clear
	}
}

func index(buckets map[string]*cpcolls.Set[string], key, id string) {
	bucket, ok := buckets[key]
	if !ok {
		bucket = cpcolls.NewSet[string]()
		buckets[key] = bucket
	}
	bucket.Add(id)
}

func unindex(buckets map[string]*cpcolls.Set[string], key, id string) {
	bucket, ok := buckets[key]
	if !ok {
		return
	}
	bucket.Delete(id)
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

type churnSample struct {
	accounts int
	family   string
}

type chain struct {
	prefix string
	flag   string
}

func (w *Watchdog) sweep() {
	now := w.clock.Now()

	w.mu.Lock()

	cutoff := now.Add(-w.config.TrackWindow)
	for _, a := range w.accounts {
		if a.last.Before(cutoff) {
			w.dropLocked(a)
		}
	}

	scopes := cpcolls.NewSet[string]()
	relays := cpcolls.NewSet[chain]()
	for _, a := range w.accounts {
		if !a.last.After(w.sweptAt) {
			continue
		}
		scopes.Add(a.scope)
		if flag, ok := w.painting(a); ok && a.prefix != "" {
			relays.Add(chain{prefix: a.prefix, flag: flag})
		}
	}

	churns := make([]churnSample, 0, scopes.Len())
	scopes.ForEach(func(scope string) {
		_, family := w.boundsOf(scope)
		churns = append(churns, churnSample{accounts: w.bornLocked(scope, now), family: family})
	})

	links := make([]int, 0, relays.Len())
	relays.ForEach(func(c chain) {
		links = append(links, w.linksLocked(c.prefix, c.flag, now))
	})

	w.sweptAt = now
	w.mu.Unlock()

	for _, sample := range churns {
		w.onAccounts(sample.accounts, sample.family)
	}
	for _, n := range links {
		w.onLinks(n)
	}
}
