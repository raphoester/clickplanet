// Package clickbudget caps what one minted session is worth.
//
// The throttle limits the rate a caller clicks at; this limits the total a
// single attestation buys, after which the caller has to go back to Turnstile.
// The two are different questions, and a bucket cannot answer this one: it
// refills, which is exactly what a budget must not do.
//
// What this is worth, honestly. It does not lower anyone's sustained click
// rate. That ceiling is the click throttle's, and a caller who exhausts a
// budget simply mints again — within the mint throttle, which allows far more
// mints per hour than a generous budget needs. It buys two things instead:
//
//   - A hard ceiling on what one attestation is worth, so a token that escapes
//     the browser it was minted in is worth a bounded number of clicks rather
//     than a whole TTL of them.
//   - A signal. A player does not exhaust a budget set where this one is set;
//     something clicking at the throttle for a quarter of an hour without
//     pause does, and does it again every quarter hour. click_budget_exhausted
//     is close to a bot counter.
//
// Sizing follows from that. Set it far above what a session of real play
// spends and it reports; set it near what the throttle allows over a TTL and
// it starts refusing players for playing. It is not a rate limiter, and tuning
// it as if it were one gets the second outcome.
package clickbudget

import (
	"context"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/xtime"
)

type Config struct {
	// Clicks one minted session may spend. Zero or less disables the budget
	// entirely, which is what a deployment that has not sized one yet wants:
	// the click path is otherwise unchanged.
	Clicks int

	SweepInterval time.Duration
}

const defaultSweepInterval = 5 * time.Minute

// New returns nil when no budget is configured, and a nil *Budget spends
// freely — so a disabled budget costs the click path one nil check rather than
// a branch at every call site.
//
// retention is how long a session's tally is kept, and must be the session
// TTL: a tally dropped while its token is still valid hands that token a
// second budget. The caller passes the signer's own TTL rather than a
// configured value so the two cannot drift apart.
func New(config Config, retention time.Duration, timeProvider xtime.Provider) *Budget {
	if config.Clicks <= 0 {
		return nil
	}

	if config.SweepInterval <= 0 {
		config.SweepInterval = defaultSweepInterval
	}

	if timeProvider == nil {
		timeProvider = xtime.ActualProvider{}
	}

	return &Budget{
		clicks:        config.Clicks,
		retention:     retention,
		sweepInterval: config.SweepInterval,
		timeProvider:  timeProvider,
		tallies:       make(map[string]*tally),
	}
}

type Budget struct {
	clicks        int
	retention     time.Duration
	sweepInterval time.Duration
	timeProvider  xtime.Provider

	mu      sync.Mutex
	tallies map[string]*tally
}

type tally struct {
	spent int
	first time.Time
}

// Spend charges one click to id and reports whether it was affordable. A
// session that has run out stays out: nothing here refills, and the caller is
// expected to mint another session rather than to wait.
func (b *Budget) Spend(id string) bool {
	if b == nil {
		return true
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	t, ok := b.tallies[id]
	if !ok {
		t = &tally{first: b.timeProvider.Now()}
		b.tallies[id] = t
	}

	if t.spent >= b.clicks {
		return false
	}

	t.spent++
	return true
}

func (b *Budget) Run(ctx context.Context) {
	if b == nil {
		return
	}

	ticker := time.NewTicker(b.sweepInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			b.sweep()
		case <-ctx.Done():
			return
		}
	}
}

// sweep forgets the tallies whose sessions can no longer be verified anyway.
//
// A tally is created on a session's first click, which is at or after the mint
// it belongs to, so first+retention is at or after the token's own expiry:
// dropping on that boundary can only ever forget a session that is already
// refused upstream, never one still holding a valid token.
func (b *Budget) sweep() {
	now := b.timeProvider.Now()

	b.mu.Lock()
	defer b.mu.Unlock()

	for id, t := range b.tallies {
		if now.Sub(t.first) >= b.retention {
			delete(b.tallies, id)
		}
	}
}
