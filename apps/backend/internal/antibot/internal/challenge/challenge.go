// Package challenge is the second consequence, and like shadowban it is the
// boring half: it takes a scope, the session that scope is clicking with and a
// clock, and remembers who has been sent back to prove it is a person. It knows
// nothing about tiles, watchdogs or what earned it.
//
// It exists because a ban has to be nearly certain before it fires — a wrong
// one is invisible to the player and unrecoverable — while most of what the
// watchdogs collect is a suspicion that never reaches that bar and was
// therefore thrown away. A challenge can be wrong and cost a real player about
// two seconds, so it can fire where the evidence actually is.
package challenge

import (
	"context"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Config struct {
	// Enforce false raises and reports challenges without refusing anything:
	// the count of what enforcing would refuse, before it refuses it. The same
	// shape as shadowBan.enforce and session.enforce, and the mode to deploy in.
	Enforce bool

	// MinSuspects is how many watchdogs at Suspect are enough to challenge.
	// Unset takes one below the jury's ban bar. It is necessarily below that
	// bar: a caller reaching it is banned, and challenging one would only tell
	// a bot which of its clicks were noticed.
	MinSuspects int

	// Interval does two jobs, because both are the same question — how long one
	// challenge's worth of evidence is good for. It is how long a challenge
	// stands unanswered, so a caller whose client cannot answer one is refused
	// for this long and not forever; and it is the soonest a caller that did
	// answer one is challenged again, so answering does not put the player
	// straight back in front of the widget. Unset takes jury.suspicionWindow,
	// the window a reading stands in.
	Interval time.Duration
}

const defaultInterval = 10 * time.Minute

// WithDefaults fills the unset bounds. Exported inside this tree so the caller
// can log the numbers that will actually be enforced rather than the raw file.
func (c Config) WithDefaults(banAt int) Config {
	if c.MinSuspects == 0 {
		c.MinSuspects = banAt - 1
	}
	if c.Interval <= 0 {
		c.Interval = defaultInterval
	}
	return c
}

// New builds the register. It is always built, even with Enforce false, because
// counting what a challenge would refuse is the whole of the rollout mode.
func New(config Config, clock cptime.Clock, hooks Hooks) *Challenges {
	if clock == nil {
		clock = cptime.SystemClock{}
	}

	// MinSuspects is deliberately not defaulted here: it only means anything
	// against the jury's bar, which this package cannot see, so WithDefaults
	// takes it and a zero left here challenges nobody.
	if config.Interval <= 0 {
		config.Interval = defaultInterval
	}

	return &Challenges{
		config:     config,
		clock:      clock,
		hooks:      hooks,
		challenges: make(map[string]*challenge),
	}
}

// Hooks is how a challenge leaves this package. Both are optional.
type Hooks struct {
	// OnAnswered is a caller clicking with a session it was not challenged on:
	// it went back through the mint, which cannot be passed without Turnstile.
	OnAnswered func(scope string)
}

type Challenges struct {
	config Config
	clock  cptime.Clock
	hooks  Hooks

	mu         sync.Mutex
	challenges map[string]*challenge
}

type challenge struct {
	// session is the one the caller held when it was challenged. Clicking with
	// another is the proof: the only way to get one is the mint, and the only
	// way through the mint is Turnstile.
	session string
	at      time.Time

	// answered keeps the record alive after it is cleared, so the next click
	// does not raise a fresh challenge and put the player back in front of the
	// widget. It lapses with the rest of the record at at+Interval.
	answered bool
}

// Raise records a challenge and says whether this is a new one. It records
// whatever Enforce is: with it off, this is the count of what would have been
// challenged, which is the point of shipping that way.
//
// A challenge already standing, answered or not, is not raised again — a caller
// is reported once per Interval however many clicks it makes inside it.
func (c *Challenges) Raise(scope, session string) bool {
	// Nothing to prove and no way to prove it: the caller presented no session
	// this server accepted, so there is no fresh one it could come back with
	// and a challenge would refuse it until the record lapsed. That is the
	// session context's refusal to make, not this one's.
	if scope == "" || session == "" {
		return false
	}

	now := c.clock.Now()

	c.mu.Lock()
	defer c.mu.Unlock()

	if record, ok := c.challenges[scope]; ok && c.standingLocked(record, now) {
		return false
	}

	c.challenges[scope] = &challenge{session: session, at: now}

	return true
}

// Standing says whether this click must be refused, and clears a challenge the
// caller has just answered. It is false while Enforce is off, the way a ban is
// not served while shadowBan.enforce is off.
//
// Clearing happens here rather than in a method of its own because the two are
// one question asked of one click: a caller holding a session it was not
// challenged on has answered, and answering is what stops the refusal.
func (c *Challenges) Standing(scope, session string) bool {
	held, answered := c.read(scope, session, c.clock.Now())

	// Outside the lock, for the same reason the jury reports outside its own:
	// the hook writes a log line and a metric.
	if answered && c.hooks.OnAnswered != nil {
		c.hooks.OnAnswered(scope)
	}

	return held && c.config.Enforce
}

// read clears a challenge the caller has just answered and says whether one is
// still being asked of it. Both in one pass under one lock, because they are
// one question asked of one click.
func (c *Challenges) read(scope, session string, now time.Time) (held, answered bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	record, ok := c.challenges[scope]
	if !ok || !c.standingLocked(record, now) {
		return false, false
	}

	// A caller holding a session it was not challenged on went back through the
	// mint, and the only way through the mint is Turnstile. A caller that
	// dropped its token and came back without one has proved nothing, so an
	// empty session leaves the challenge standing.
	if session != "" && session != record.session && !record.answered {
		record.answered = true
		return false, true
	}

	return !record.answered, false
}

// Clear drops any challenge on the scope. The jury calls it when the caller
// crosses the ban bar: a ban is silent, and a caller left being told to
// re-authenticate while its clicks are quietly dropped has been told.
func (c *Challenges) Clear(scope string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.challenges, scope)
}

func (c *Challenges) standingLocked(record *challenge, now time.Time) bool {
	return now.Sub(record.at) < c.config.Interval
}

// Count is how many challenges are standing unanswered right now, for the gauge.
func (c *Challenges) Count() int {
	now := c.clock.Now()

	c.mu.Lock()
	defer c.mu.Unlock()

	var count int
	for _, record := range c.challenges {
		if !record.answered && c.standingLocked(record, now) {
			count++
		}
	}

	return count
}

func (c *Challenges) Enforcing() bool { return c.config.Enforce }

// MinSuspects is the bar the jury asks this register for rather than being told
// twice, so the number and the thing that acts on it never drift.
func (c *Challenges) MinSuspects() int { return c.config.MinSuspects }

// Run forgets lapsed records; without it the map keeps an entry for every
// caller ever challenged.
//
// Nothing here is saved between boots, unlike the bans and the evidence. A
// challenge is minutes long and clears itself, so a restart costs a real player
// nothing — they are simply not asked — and costs a bot one Interval of the
// suspicion that raised it, which the watchdogs re-earn from evidence that does
// survive. A section in antibot.evidence would be state with a shorter life
// than the flush that writes it.
func (c *Challenges) Run(ctx context.Context) {
	ticker := time.NewTicker(c.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.forget(c.clock.Now())
		case <-ctx.Done():
			return
		}
	}
}

func (c *Challenges) forget(now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for scope, record := range c.challenges {
		if !c.standingLocked(record, now) {
			delete(c.challenges, scope)
		}
	}
}
