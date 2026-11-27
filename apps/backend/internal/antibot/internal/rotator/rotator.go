package rotator

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/bits"
	"sort"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const Name = "rotator"

type Rules struct {
	Enabled bool

	Window time.Duration
	Span   time.Duration

	MinSeconds  int
	MinByChance float64
	MaxShare    float64

	MinSpans        int
	CertainSpans    int
	CertainByChance float64
}

func (r Rules) Validate() error {
	return errors.Join(
		detect.Positive("window", r.Window),
		detect.Positive("span", r.Span),
		wholeSeconds("span", r.Span),
		detect.Positive("minSeconds", r.MinSeconds),
		detect.Positive("minByChance", r.MinByChance),
		detect.Positive("maxShare", r.MaxShare),
		detect.Share("maxShare", r.MaxShare),
		detect.NotNegative("minSpans", r.MinSpans),
		detect.NotNegative("certainSpans", r.CertainSpans),
		detect.AtLeast("certainByChance", r.CertainByChance, "minByChance", r.MinByChance),
	)
}

func wholeSeconds(name string, value time.Duration) error {
	if value%time.Second == 0 {
		return nil
	}
	return fmt.Errorf("%s is %v, and must be whole seconds", name, value)
}

type Config struct {
	// The evidence is kept per slot and saved with its length.
	Slot time.Duration

	SweepInterval time.Duration
}

const (
	defaultSlot          = 5 * time.Minute
	defaultSweepInterval = time.Minute

	rejudgeAfter        = 10 * time.Second
	maxTrackedCountries = 16
)

func (c Config) withDefaults() Config {
	if c.Slot < time.Second {
		c.Slot = defaultSlot
	}
	if c.SweepInterval <= 0 {
		c.SweepInterval = defaultSweepInterval
	}
	return c
}

func New(config Config, rules func() Rules, clock cptime.Clock, onShare func(share float64)) *Watchdog {
	config = config.withDefaults()

	return &Watchdog{
		config:   config,
		rules:    rules,
		clock:    clock,
		onShare:  onShare,
		seconds:  int64(config.Slot / time.Second),
		accounts: make(map[accountID]*account),
	}
}

type Watchdog struct {
	config  Config
	rules   func() Rules
	clock   cptime.Clock
	onShare func(float64)
	seconds int64

	mu       sync.Mutex
	accounts map[accountID]*account
	sweptAt  time.Time
}

var _ detect.Watchdog = (*Watchdog)(nil)

type (
	accountID string
	slot      int64
)

type account struct {
	id        accountID
	signedIn  bool
	scope     string
	last      time.Time
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

func (m marks) countIn(from, to int64) int {
	if m == nil {
		return 0
	}
	n := 0
	for i := from / 64; i <= (to-1)/64; i++ {
		n += bits.OnesCount64(m[i] & within(i, from, to))
	}
	return n
}

// The bits of word i that are seconds of [from, to).
func within(i, from, to int64) uint64 {
	lo, hi := max(from-i*64, 0), min(to-i*64, 64)
	if lo >= hi {
		return 0
	}
	ones := ^uint64(0)
	if hi-lo < 64 {
		ones = 1<<(hi-lo) - 1
	}
	return ones << lo
}

func (w *Watchdog) Name() string { return Name }

func (w *Watchdog) Enabled() bool { return w.rules().Enabled }

func (w *Watchdog) Attempted(click detect.Click) {
	if click.Scope == "" || click.Account == "" {
		return
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	a, ok := w.accounts[accountID(click.Account)]
	if !ok {
		a = &account{
			id:        accountID(click.Account),
			countries: make(map[string]int),
			slots:     make(map[slot]marks),
		}
		w.accounts[a.id] = a
	}

	a.signedIn = click.SignedIn
	if !click.At.Before(a.last) {
		a.scope, a.last = click.Scope, click.At
	}
	if _, known := a.countries[click.Country]; known || len(a.countries) < maxTrackedCountries {
		a.countries[click.Country]++
	}

	at := w.slotOf(click.At)
	clicked, ok := a.slots[at]
	if !ok {
		clicked = make(marks, w.words())
		a.slots[at] = clicked
	}
	clicked.mark(click.At.Unix() - int64(at)*w.seconds)
}

func (w *Watchdog) Committed(detect.Click) {}

func (w *Watchdog) Watch(click detect.Click) (detect.Verdict, detect.Evidence) {
	if click.Scope == "" || click.Account == "" {
		return detect.Clear, detect.Evidence{}
	}

	rules := w.rules()

	w.mu.Lock()
	defer w.mu.Unlock()

	a, ok := w.accounts[accountID(click.Account)]
	if !ok {
		return detect.Clear, detect.Evidence{}
	}

	if a.judgedAt.IsZero() || click.At.Sub(a.judgedAt) >= rejudgeAfter {
		a.judgedAt = click.At
		a.verdict, a.evidence = w.judgeLocked(a, click.At, rules)
	}

	return a.verdict, a.evidence
}

// Every reading is judged again: a partner's would still name the turns forgotten.
func (w *Watchdog) ForgetTurns(account string, until time.Time) {
	w.mu.Lock()
	defer w.mu.Unlock()

	a, ok := w.accounts[accountID(account)]
	if !ok {
		return
	}

	last := w.slotOf(until)
	for at := range a.slots {
		if at <= last {
			delete(a.slots, at)
		}
	}

	for _, b := range w.accounts {
		b.judgedAt = time.Time{}
	}
}

func (w *Watchdog) slotOf(at time.Time) slot { return slot(at.Unix() / w.seconds) }

func (w *Watchdog) words() int { return int((w.seconds + 63) / 64) }

// The ring is the partners at the reading's level: a certain reading names no account it only suspects.
func (w *Watchdog) judgeLocked(a *account, now time.Time, rules Rules) (detect.Verdict, detect.Evidence) {
	from := w.slotOf(now.Add(-rules.Window))

	verdict := detect.Clear
	var partners [detect.Certain + 1][]*account
	for _, b := range w.accounts {
		if b == a || !paired(a, b) {
			continue
		}
		if level := rules.level(w.sharingOf(from, rules, a, b)); level > detect.Clear {
			partners[level] = append(partners[level], b)
			verdict = max(verdict, level)
		}
	}

	if verdict == detect.Clear {
		return detect.Clear, detect.Evidence{}
	}

	ring := partners[verdict]
	sort.Slice(ring, func(i, j int) bool { return ring[i].id < ring[j].id })

	accomplices := make([]detect.Accomplice, 0, len(ring))
	for _, partner := range ring {
		accomplices = append(accomplices, detect.Accomplice{Account: string(partner.id), SignedIn: partner.signedIn, Scope: partner.scope})
	}

	ring = append([]*account{a}, ring...)
	together := w.sharingOf(from, rules, ring...)

	return verdict, detect.Evidence{
		Rule: "rotation",
		Fields: []detect.Field{
			{Key: "accounts", Value: len(ring)},
			{Key: "flag", Value: a.flag()},
			{Key: "sharedSpans", Value: together.spans},
			{Key: "sharedSeconds", Value: together.seconds},
			{Key: "byChance", Value: math.Round(together.byChance)},
		},
		Turns:       true,
		Accomplices: accomplices,
	}
}

// Two guests on one scope spend from one bank, and anywhere a guest is shed with its cookie.
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
	spans    int
	seconds  int
	byChance float64
}

func (s sharing) share() float64 { return float64(s.seconds) / s.byChance }

// Chance is counted per span: a minute where one account alone clicked adds none.
func (w *Watchdog) sharingOf(from slot, rules Rules, accounts ...*account) sharing {
	var (
		shared sharing
		span   = int64(rules.Span / time.Second)
		once   = make(marks, w.words())
		twice  = make(marks, w.words())
	)

	for i, first := range accounts {
		for at := range first.slots {
			if at < from || held(accounts[:i], at) {
				continue
			}

			for start := int64(0); start < w.seconds; start += span {
				end := min(start+span, w.seconds)
				if counted, seconds, pairs := spanOf(accounts[i:], at, start, end, rules.MinSeconds, once, twice); counted {
					shared.spans++
					shared.seconds += seconds
					shared.byChance += float64(pairs) / float64(end-start)
				}
			}
		}
	}

	return shared
}

// pairs is the sum of a·b over every two accounts that clicked in minSeconds seconds of the span.
func spanOf(accounts []*account, at slot, start, end int64, minSeconds int, once, twice marks) (counted bool, seconds, pairs int) {
	clear(once)
	clear(twice)
	present, sum, squares := 0, 0, 0

	for _, member := range accounts {
		clicked := member.slots[at]
		n := clicked.countIn(start, end)
		if n < minSeconds {
			continue
		}

		present++
		sum += n
		squares += n * n
		for j := start / 64; j <= (end-1)/64; j++ {
			word := clicked[j] & within(j, start, end)
			twice[j] |= once[j] & word
			once[j] |= word
		}
	}

	if present < 2 {
		return false, 0, 0
	}
	return true, twice.count(), (sum*sum - squares) / 2
}

func held(accounts []*account, at slot) bool {
	for _, a := range accounts {
		if _, ok := a.slots[at]; ok {
			return true
		}
	}
	return false
}

func (r Rules) readable(s sharing) bool { return s.byChance >= r.MinByChance }

func (r Rules) level(s sharing) detect.Verdict {
	if !r.readable(s) || s.share() > r.MaxShare {
		return detect.Clear
	}

	switch {
	case r.CertainSpans > 0 && s.spans >= r.CertainSpans && s.byChance >= r.CertainByChance:
		return detect.Certain
	case r.MinSpans > 0 && s.spans >= r.MinSpans:
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
	rules := w.rules()
	from := w.slotOf(now.Add(-rules.Window))

	w.mu.Lock()

	w.forgetLocked(from)
	shares := w.sharesLocked(from, rules)
	w.sweptAt = now

	w.mu.Unlock()

	for _, share := range shares {
		w.onShare(share)
	}
}

func (w *Watchdog) sharesLocked(from slot, rules Rules) []float64 {
	var shares []float64

	for _, a := range w.accounts {
		for _, b := range w.accounts {
			if a.id >= b.id || (!a.last.After(w.sweptAt) && !b.last.After(w.sweptAt)) || !paired(a, b) {
				continue
			}
			if together := w.sharingOf(from, rules, a, b); rules.readable(together) {
				shares = append(shares, together.share())
			}
		}
	}

	return shares
}

func (w *Watchdog) forgetLocked(from slot) {
	for id, a := range w.accounts {
		for at := range a.slots {
			if at < from {
				delete(a.slots, at)
			}
		}
		if len(a.slots) == 0 {
			delete(w.accounts, id)
		}
	}
}
