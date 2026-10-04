package detect

import (
	"fmt"
	"net/netip"
	"sort"
	"strings"
	"time"
)

type Click struct {
	Scope    string
	Account  string
	SignedIn bool

	Tile    uint32
	Country string
	At      time.Time

	Held string
	NoOp bool

	Cleared bool

	// How many times its plain pace the caller's bank refills at; 0 is the plain pace.
	Pace float64
}

type Verdict uint8

const (
	Clear Verdict = iota
	Suspect
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

type Field struct {
	Key   string
	Value any
}

type Evidence struct {
	Rule   string
	Fields []Field
}

type Watchdog interface {
	Name() string

	Attempted(click Click)

	Watch(click Click) (Verdict, Evidence)

	Committed(click Click)
}

type Opinion struct {
	Watchdog string
	Verdict  Verdict
	Evidence Evidence
	At       time.Time
}

func (o Opinion) Fired() bool { return o.Verdict != Clear }

func (o Opinion) String() string {
	if !o.Fired() {
		return o.Verdict.String()
	}

	return o.Verdict.String() + " " + o.Evidence.String()
}

func (e Evidence) String() string {
	if e.Rule == "" && len(e.Fields) == 0 {
		return ""
	}

	parts := make([]string, 0, len(e.Fields)+1)
	parts = append(parts, e.Rule)

	// Copy before sorting: the caller still holds e.Fields.
	fields := append([]Field(nil), e.Fields...)
	sort.SliceStable(fields, func(i, j int) bool { return fields[i].Key < fields[j].Key })

	for _, field := range fields {
		parts = append(parts, fmt.Sprintf("%s=%v", field.Key, field.Value))
	}

	return strings.Join(parts, " ")
}

type Reading struct {
	Watchdog string
	Level    string
	Evidence string
	At       time.Time
}

func (o Opinion) Reading() Reading {
	return Reading{Watchdog: o.Watchdog, Level: o.Verdict.String(), Evidence: o.Evidence.String(), At: o.At}
}

type Examination struct {
	Scope   string
	Account string
	Tracked bool

	Banned      bool
	Flags       int
	Offence     int
	BannedUntil time.Time

	Readings []Reading

	Suspects    int
	MinSuspects int
	Guilty      bool

	Clicks           int
	ActiveFor        time.Duration
	LongestGap       time.Duration
	LastClickAt      time.Time
	TopCountry       string
	TopCountryClicks int
}

type Report struct {
	Scope   string
	Account string
	Flags   int

	Offence     int
	BannedUntil time.Time

	Opinions []Opinion

	Clicks int

	ActiveFor  time.Duration
	LongestGap time.Duration

	TopCountry       string
	TopCountryClicks int

	Tiles []uint32
}

type Outage struct {
	From time.Time
	To   time.Time
}

func (o Outage) Across(last, next time.Time) bool {
	return o.To.After(o.From) && !last.After(o.From) && !next.Before(o.To)
}

func (o Outage) Length() time.Duration {
	if !o.To.After(o.From) {
		return 0
	}
	return o.To.Sub(o.From)
}

func (o Outage) Gap(last, next time.Time) time.Duration {
	gap := next.Sub(last)
	if o.Across(last, next) {
		gap -= o.Length()
	}
	return gap
}

func WiderPrefix(scope string, v4Bits, v6Bits int) string {
	if addr, err := netip.ParseAddr(scope); err == nil {
		addr = addr.Unmap()
		bits := v6Bits
		if addr.Is4() {
			bits = v4Bits
		}
		if prefix, err := addr.Prefix(bits); err == nil {
			return prefix.String()
		}
		return ""
	}

	prefix, err := netip.ParsePrefix(scope)
	if err != nil || prefix.Addr().Is4() || prefix.Bits() < v6Bits {
		return ""
	}

	wide, err := prefix.Addr().Prefix(v6Bits)
	if err != nil {
		return ""
	}
	return wide.String()
}

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

// Spread sorts delays in place.
func Spread(delays []time.Duration) (median, spread time.Duration) {
	sort.Slice(delays, func(i, j int) bool { return delays[i] < delays[j] })
	return Quantile(delays, 0.5), Quantile(delays, 0.9) - Quantile(delays, 0.1)
}
