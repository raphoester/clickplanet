// Package detect is the vocabulary every watchdog and the jury share: what a
// click looks like, how sure a watchdog is, and what a ban has to say for
// itself.
//
// It sits under antibot/internal because a caller of the antibot package needs
// these types but must not be able to build a watchdog or a jury out of them.
// The antibot root re-exports the handful a caller actually reads.
package detect

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// Click is one Click RPC as the jury sees it, before the handler runs.
type Click struct {
	Scope   string
	Tile    uint32
	Country string
	At      time.Time

	// Held is the country owning the tile as the click arrives, empty when
	// nobody does, and NoOp says the caller's own country already holds it. Both
	// are read before the handler runs, because by the time it has run the map
	// no longer remembers what was there.
	Held string
	NoOp bool
}

// Verdict is how sure one watchdog is. The split exists because the bounds that
// catch a bot on their own also catch the most obsessed players: a watchdog that
// only had one level would have to be set at the strict end and would then miss
// every bot that jitters, or at the loose end and ban humans.
type Verdict uint8

const (
	// Clear is the caller looking like anybody else.
	Clear Verdict = iota
	// Suspect is a reading no single watchdog should ban on. It counts only
	// alongside another watchdog measuring something else.
	Suspect
	// Certain is a reading no hand produces. It bans on its own.
	Certain
)

func (v Verdict) String() string {
	switch v {
	case Suspect:
		return "suspect"
	case Certain:
		return "certain"
	default:
		return "clear"
	}
}

// Field is one number a watchdog wants in the log line. Watchdogs measure
// different things, so the shape of the evidence is theirs and not the jury's.
type Field struct {
	Key   string
	Value any
}

// Evidence is why a watchdog returned the verdict it did. Rule names which of
// its rules spoke, for the watchdogs that have more than one.
type Evidence struct {
	Rule   string
	Fields []Field
}

// Watchdog measures one behaviour over one caller.
type Watchdog interface {
	Name() string

	// Watch records the click and says how the caller reads now. It is called
	// for every click, including the ones an existing ban is already dropping:
	// a watchdog that stops being fed while its caller is banned cannot say
	// whether the ban is still earned, and the ban would lapse on silence the
	// caller never actually produced.
	Watch(click Click) (Verdict, Evidence)

	// Committed is called once the click has reached the map. A watchdog that
	// does not care what the map does ignores it.
	Committed(click Click)
}

// Opinion is one watchdog's standing verdict on a caller.
type Opinion struct {
	Watchdog string
	Verdict  Verdict
	Evidence Evidence
	At       time.Time
}

// Fired says whether this watchdog argued for the ban. It is the only thing a
// caller asks of a verdict — how sure the ones that did fire are is the jury's
// business — so it is a method here rather than a verdict a caller compares.
func (o Opinion) Fired() bool { return o.Verdict != Clear }

// String renders one watchdog's reading for the log line: the verdict, the rule
// that tripped, and every number that rule wanted, ordered by key so two lines
// about the same watchdog read the same way. The caller owns the message and the
// attribute names; what one reading says is this package's to word.
func (o Opinion) String() string {
	if !o.Fired() {
		return o.Verdict.String()
	}

	parts := make([]string, 0, len(o.Evidence.Fields)+1)
	parts = append(parts, fmt.Sprintf("%s %s", o.Verdict, o.Evidence.Rule))

	// Copied before sorting: the report holds this slice and rendering it must
	// not reorder what the caller is still holding.
	fields := append([]Field(nil), o.Evidence.Fields...)
	sort.SliceStable(fields, func(i, j int) bool { return fields[i].Key < fields[j].Key })

	for _, field := range fields {
		parts = append(parts, fmt.Sprintf("%s=%v", field.Key, field.Value))
	}

	return strings.Join(parts, " ")
}

// Report is one ban, with everything that argued for it. Every watchdog is in
// Opinions, including the ones that said Clear, because what did not fire is
// half of reading a line that did.
type Report struct {
	Scope string
	Flags int

	Opinions []Opinion

	Clicks int

	// What a randomised delay cannot fake: a person stops. Neither feeds any
	// rule — deciding on them would ban the genuinely obsessed — but a ban with
	// hours of ActiveFor and a LongestGap in seconds reads very differently from
	// one without.
	ActiveFor  time.Duration
	LongestGap time.Duration

	// Self-declared by the client and trivially changed: context for whoever
	// reads the line, never an input to a decision.
	TopCountry       string
	TopCountryClicks int

	// A few of the tiles involved, most recent last.
	Tiles []uint32
}

// Quantile reads a sorted slice. It rounds to the nearest sample rather than
// interpolating, so every number reaching a log line is one that was actually
// measured. Watchdogs judge callers on the spread of what they measured rather
// than its average, so they all need this.
func Quantile(sorted []time.Duration, q float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}

	i := int(q * float64(len(sorted)-1))
	if i < 0 {
		i = 0
	}
	if i >= len(sorted) {
		i = len(sorted) - 1
	}

	return sorted[i]
}

// Spread is the p90-p10 of what a watchdog measured. It sorts in place.
func Spread(delays []time.Duration) (median, spread time.Duration) {
	sort.Slice(delays, func(i, j int) bool { return delays[i] < delays[j] })
	return Quantile(delays, 0.5), Quantile(delays, 0.9) - Quantile(delays, 0.1)
}
