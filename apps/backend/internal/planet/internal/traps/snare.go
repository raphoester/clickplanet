package traps

import (
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/notices"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
)

// Bank is the click limiter: a drain may leave a bucket below empty, down to minus its capacity.
type Bank interface {
	Drain(n float64, keys ...cpratelimit.Key) []cpratelimit.Drained
}

type Teller interface {
	PublishTrapSprung(sprung notices.TrapSprung)
}

func NewSnare(bank Bank, buckets clicks.Buckets, rules func() charges.Rules, teller Teller) Snare {
	return Snare{bank: bank, buckets: buckets, rules: rules, teller: teller}
}

// Snare takes what a trap a take sprang costs the take's player: charges.trap.clicks from its own bank, and from its
// network's guest bank for a guest, so a new private tab owes it too. Each bank goes down to minus one bank at most,
// and the clicks go to nobody: to the trap's player, a second account on another flag could feed a bank. Every stream
// hears the snap. It runs under the claim's lock of the tiles, so the trap leaves the map and the bank pays together.
type Snare struct {
	bank    Bank
	buckets clicks.Buckets
	rules   func() charges.Rules
	teller  Teller
}

var _ clicks.Snare = Snare{}

// Spring answers what left the payer's own bank: less than the rule's clicks near the floor.
func (s Snare) Spring(payer clicks.Payer, trap clicks.TrapSnapshot) float64 {
	var taken float64
	if drained := s.bank.Drain(float64(s.rules().Trap.Clicks), s.buckets.Bank(payer)...); len(drained) > 0 {
		taken = drained[0].Taken
	}
	s.teller.PublishTrapSprung(notices.TrapSprung{
		Tile: trap.Tile, Flag: trap.Flag, Clicks: taken, Taker: charges.HolderOf(payer),
	})

	return taken
}
