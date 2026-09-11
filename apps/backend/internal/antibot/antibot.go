// Package antibot holds what is left after the address blocklist and the
// session mint: a caller who solved Turnstile in a real browser and then pointed
// a script at the API. Nothing about the address or the token separates them
// from a player, so everything here reads behaviour instead.
//
// A Watchdog measures one behaviour and says how sure it is. The Jury crosses
// what the watchdogs say. The shadowban subpackage carries out the sentence and
// knows nothing about bots.
package antibot

import (
	"sort"
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
