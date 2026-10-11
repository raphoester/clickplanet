package rotator

import (
	"context"
	"math"
	"math/bits"
	"sort"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const Name = "rotator"

type Config struct {
	Slot   time.Duration
	Window time.Duration

	MinSeconds  int
	MinByChance float64
	MaxShare    float64

	MinSlots     int
	CertainSlots int

	SweepInterval time.Duration
}

const (
	defaultSlot          = 5 * time.Minute
	defaultWindow        = 24 * time.Hour
	defaultMinSeconds    = 15
	defaultMinByChance   = 90
	defaultMaxShare      = 0.05
	defaultSweepInterval = time.Minute

	rejudgeAfter        = 10 * time.Second
	maxTrackedCountries = 16
)

func (c Config) withDefaults() Config {
	if c.Slot < time.Second {
		c.Slot = defaultSlot
	}
	if c.Window < c.Slot {
		c.Window = defaultWindow
	}
	if c.MinSeconds <= 0 {
		c.MinSeconds = defaultMinSeconds
	}
	if c.MinByChance <= 0 {
		c.MinByChance = defaultMinByChance
	}
	if c.MaxShare <= 0 {
		c.MaxShare = defaultMaxShare
	}
	if c.SweepInterval <= 0 {
		c.SweepInterval = defaultSweepInterval
	}
	return c
}

func New(config Config, clock cptime.Clock, onShare func(share float64)) *Watchdog {
	config = config.withDefaults()

	return &Watchdog{
		config:  config,
		clock:   clock,
		onShare: onShare,
		span:    int64(config.Slot / time.Second),
		scopes:  make(map[scopeID]*network),
	}
}

type Watchdog struct {
	config  Config
	clock   cptime.Clock
	onShare func(float64)
	span    int64

	mu      sync.Mutex
	scopes  map[scopeID]*network
	sweptAt time.Time
}

var _ detect.Watchdog = (*Watchdog)(nil)

type (
	scopeID   string
	accountID string
	slot      int64
)

type network struct {
	accounts map[accountID]*account
	last     time.Time
}

type account struct {
	id        accountID
	signedIn  bool
	countries map[string]int
	slots     map[slot]marks

	judgedAt time.Time
	verdict  detect.Verdict
	evidence detect.Evidence
}

type marks []uint64

func (m marks) mark(second int64) { m[second/64] |= 1 << (second % 64) }

func (m marks) count() int {
	n := 0
	for _, word := range m {
		n += bits.OnesCount64(word)
	}
	return n
}

func (w *Watchdog) Name() string { return Name }

func (w *Watchdog) Attempted(click detect.Click) {
	if click.Scope == "" || click.Account == "" {
		return
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	n, ok := w.scopes[scopeID(click.Scope)]
	if !ok {
		n = &network{accounts: make(map[accountID]*account)}
		w.scopes[scopeID(click.Scope)] = n
	}
	if click.At.After(n.last) {
		n.last = click.At
	}

	a, ok := n.accounts[accountID(click.Account)]
	if !ok {
		a = &account{
			id:        accountID(click.Account),
			countries: make(map[string]int),
			slots:     make(map[slot]marks),
		}
		n.accounts[a.id] = a
	}

	a.signedIn = click.SignedIn
	if _, known := a.countries[click.Country]; known || len(a.countries) < maxTrackedCountries {
		a.countries[click.Country]++
	}

	at := w.slotOf(click.At)
	clicked, ok := a.slots[at]
	if !ok {
		clicked = make(marks, w.words())
		a.slots[at] = clicked
	}
	clicked.mark(click.At.Unix() - int64(at)*w.span)
}

func (w *Watchdog) Committed(detect.Click) {}

func (w *Watchdog) Watch(click detect.Click) (detect.Verdict, detect.Evidence) {
	if click.Scope == "" || click.Account == "" {
		return detect.Clear, detect.Evidence{}
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	n, ok := w.scopes[scopeID(click.Scope)]
	if !ok {
		return detect.Clear, detect.Evidence{}
	}
	a, ok := n.accounts[accountID(click.Account)]
	if !ok {
		return detect.Clear, detect.Evidence{}
	}

	if a.judgedAt.IsZero() || click.At.Sub(a.judgedAt) >= rejudgeAfter {
		a.judgedAt = click.At
		a.verdict, a.evidence = w.judgeLocked(n, a, click.At)
	}

	return a.verdict, a.evidence
}

func (w *Watchdog) slotOf(at time.Time) slot { return slot(at.Unix() / w.span) }

func (w *Watchdog) words() int { return int((w.span + 63) / 64) }

func (w *Watchdog) judgeLocked(n *network, a *account, now time.Time) (detect.Verdict, detect.Evidence) {
	from := w.slotOf(now.Add(-w.config.Window))

	verdict := detect.Clear
	ring := []*account{a}
	for _, b := range n.accounts {
		if b == a || !paired(a, b) {
			continue
		}
		if level := w.level(w.sharingOf(from, a, b)); level > detect.Clear {
			ring = append(ring, b)
			verdict = max(verdict, level)
		}
	}

	if verdict == detect.Clear {
		return detect.Clear, detect.Evidence{}
	}

	partners := ring[1:]
	sort.Slice(partners, func(i, j int) bool { return partners[i].id < partners[j].id })

	accomplices := make([]detect.Accomplice, 0, len(partners))
	for _, partner := range partners {
		accomplices = append(accomplices, detect.Accomplice{Account: string(partner.id), SignedIn: partner.signedIn})
	}

	together := w.sharingOf(from, ring...)

	return verdict, detect.Evidence{
		Rule: "rotation",
		Fields: []detect.Field{
			{Key: "accounts", Value: len(ring)},
			{Key: "flag", Value: a.flag()},
			{Key: "sharedSlots", Value: together.slots},
			{Key: "sharedSeconds", Value: together.seconds},
			{Key: "byChance", Value: math.Round(together.byChance)},
		},
		Accomplices: accomplices,
	}
}

// Two guests on one scope spend from one bank: taking turns buys them nothing.
func paired(a, b *account) bool {
	if !a.signedIn && !b.signedIn {
		return false
	}

	flag := a.flag()
	return flag != "" && flag == b.flag()
}

func (a *account) flag() string {
	var (
		top   string
		count int
	)
	for country, n := range a.countries {
		if n > count || (n == count && country < top) {
			top, count = country, n
		}
	}
	return top
}

type sharing struct {
	slots    int
	seconds  int
	byChance float64
}

func (s sharing) share() float64 { return float64(s.seconds) / s.byChance }

func (w *Watchdog) sharingOf(from slot, accounts ...*account) sharing {
	var (
		shared   sharing
		products int
		once     = make(marks, w.words())
		twice    = make(marks, w.words())
	)

	for i, first := range accounts {
		for at := range first.slots {
			if at < from || held(accounts[:i], at) {
				continue
			}

			clear(once)
			clear(twice)
			present, sum, squares := 0, 0, 0

			for _, member := range accounts[i:] {
				clicked := member.slots[at]
				n := clicked.count()
				if n < w.config.MinSeconds {
					continue
				}

				present++
				sum += n
				squares += n * n
				for j, word := range clicked {
					twice[j] |= once[j] & word
					once[j] |= word
				}
			}

			if present < 2 {
				continue
			}

			shared.slots++
			shared.seconds += twice.count()
			products += (sum*sum - squares) / 2
		}
	}

	shared.byChance = float64(products) / float64(w.span)

	return shared
}

func held(accounts []*account, at slot) bool {
	for _, a := range accounts {
		if _, ok := a.slots[at]; ok {
			return true
		}
	}
	return false
}

func (w *Watchdog) readable(s sharing) bool { return s.byChance >= w.config.MinByChance }

func (w *Watchdog) level(s sharing) detect.Verdict {
	if !w.readable(s) || s.share() > w.config.MaxShare {
		return detect.Clear
	}

	switch {
	case w.config.CertainSlots > 0 && s.slots >= w.config.CertainSlots:
		return detect.Certain
	case w.config.MinSlots > 0 && s.slots >= w.config.MinSlots:
		return detect.Suspect
	default:
		return detect.Clear
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
	from := w.slotOf(now.Add(-w.config.Window))

	var shares []float64

	w.mu.Lock()

	w.forgetLocked(from)
	for _, n := range w.scopes {
		if n.last.After(w.sweptAt) {
			shares = append(shares, w.sharesLocked(n, from)...)
		}
	}
	w.sweptAt = now

	w.mu.Unlock()

	for _, share := range shares {
		w.onShare(share)
	}
}

func (w *Watchdog) sharesLocked(n *network, from slot) []float64 {
	var shares []float64

	for _, a := range n.accounts {
		for _, b := range n.accounts {
			if a.id >= b.id || !paired(a, b) {
				continue
			}
			if together := w.sharingOf(from, a, b); w.readable(together) {
				shares = append(shares, together.share())
			}
		}
	}

	return shares
}

func (w *Watchdog) forgetLocked(from slot) {
	for scope, n := range w.scopes {
		for id, a := range n.accounts {
			for at := range a.slots {
				if at < from {
					delete(a.slots, at)
				}
			}
			if len(a.slots) == 0 {
				delete(n.accounts, id)
			}
		}
		if len(n.accounts) == 0 {
			delete(w.scopes, scope)
		}
	}
}
