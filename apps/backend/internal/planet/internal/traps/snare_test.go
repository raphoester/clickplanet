package traps_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/notices"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/traps"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const (
	guest  = "0192f5a0-7c00-7000-8000-0000000000c1"
	member = "0192f5a0-7c00-7000-8000-0000000000c2"
	scope  = "198.51.100.0/24"
)

var trap = clicks.TrapSnapshot{Tile: 5, Flag: "fr"}

type teller struct{ told []notices.TrapSprung }

func (t *teller) PublishTrapSprung(sprung notices.TrapSprung) { t.told = append(t.told, sprung) }

type snared struct {
	snare   traps.Snare
	limiter *cpratelimit.Limiter
	buckets clicks.Buckets
	teller  *teller
}

// snaredAt plays the test throttle: a bank of 10, a network's guests' bank of 10, a network's of 100.
func snaredAt(t *testing.T, clicksPerTrap int) snared {
	t.Helper()
	clock := cptime.NewFixedClock(time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC))
	limiter := cpratelimit.NewRuled("clicks", func() cpratelimit.Policy { return clicks.TestThrottle().Policy(time.Time{}) },
		time.Minute, clock)
	buckets := clicks.NewBuckets(clicks.TestThrottle, clock)
	rules := charges.TestRules()
	rules.Trap.Clicks = clicksPerTrap
	told := &teller{}
	return snared{
		snare:   traps.NewSnare(limiter, buckets, func() charges.Rules { return rules }, told),
		limiter: limiter, buckets: buckets, teller: told,
	}
}

func (s snared) bank(payer clicks.Payer) []float64 {
	keys := s.buckets.Keys(payer, clicks.Price{Slowdown: 1})
	tokens := make([]float64, 0, len(keys))
	for _, key := range keys {
		tokens = append(tokens, s.limiter.Peek(key).Tokens)
	}
	return tokens
}

func TestATrapTakesItsClicksFromAGuestsBankAndItsNetworksGuestBankAndNotTheNetworks(t *testing.T) {
	s := snaredAt(t, 10)
	payer := clicks.Payer{Account: guest, Scope: scope}
	full := s.bank(payer)

	taken := s.snare.Spring(payer, trap)

	assert.InDelta(t, 10.0, taken, 1e-9)
	after := s.bank(payer)
	require.Len(t, after, 3)
	assert.InDelta(t, full[0]-10, after[0], 1e-9, "its own bank")
	assert.InDelta(t, full[1]-10, after[1], 1e-9, "its network's guests', so a new private tab owes it too")
	assert.InDelta(t, full[2], after[2], 1e-9, "not the network's, which every account behind it shares")
}

func TestATrapTakesASignedInPlayersOwnBankAlone(t *testing.T) {
	s := snaredAt(t, 10)
	payer := clicks.Payer{Account: member, Scope: scope, Linked: true}
	before := s.bank(payer)
	guestsBefore := s.bank(clicks.Payer{Account: guest, Scope: scope})

	s.snare.Spring(payer, trap)

	assert.InDelta(t, before[0]-10, s.bank(payer)[0], 1e-9)
	assert.InDelta(t, before[1], s.bank(payer)[1], 1e-9, "not the network's")
	assert.InDelta(t, guestsBefore[1], s.bank(clicks.Payer{Account: guest, Scope: scope})[1], 1e-9,
		"the guests behind its network owe nothing")
}

func TestATrapTakesTheBankBelowZeroDownToMinusOneBank(t *testing.T) {
	s := snaredAt(t, 8)
	payer := clicks.Payer{Account: member, Scope: scope, Linked: true}

	first := s.snare.Spring(payer, trap)
	second := s.snare.Spring(payer, trap)
	third := s.snare.Spring(payer, trap)

	assert.InDelta(t, 8.0, first, 1e-9)
	assert.InDelta(t, 8.0, second, 1e-9, "a bank of 2 goes to -6")
	assert.InDelta(t, 4.0, third, 1e-9, "what was left above minus one bank")
	assert.InDelta(t, -float64(clicks.TestThrottle().Burst), s.bank(payer)[0], 1e-9)
}

func TestEveryStreamHearsWhatLeftTheTakersBank(t *testing.T) {
	s := snaredAt(t, 10)

	s.snare.Spring(clicks.Payer{Account: guest, Scope: scope}, clicks.TrapSnapshot{Tile: 7, Flag: "de"})

	assert.Equal(t, []notices.TrapSprung{{Tile: 7, Flag: "de", Clicks: 10, Taker: charges.Holder(guest)}}, s.teller.told)
}
