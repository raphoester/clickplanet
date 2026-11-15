package sharing

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const Name = "sharing"

type Rules struct {
	Enabled bool

	Window      time.Duration
	MaxAccounts int
}

func (r Rules) Validate() error {
	return errors.Join(
		detect.Positive("window", r.Window),
		detect.AtLeast("maxAccounts", r.MaxAccounts, "the two accounts a pair is", 2),
	)
}

const sweepInterval = time.Minute

// Not a watchdog: it reads no level, the jury never hears it, and nothing it finds bans. It finds two accounts that
// play in turns from one scope, so each is reminded that one player plays one account.
func New(rules func() Rules, clock cptime.Clock) *Tracker {
	return &Tracker{rules: rules, clock: clock, scopes: make(map[string]map[string]*span)}
}

type Tracker struct {
	rules func() Rules
	clock cptime.Clock

	mu     sync.Mutex
	scopes map[string]map[string]*span
}

// An account's tries on one scope inside the window.
type span struct {
	first, last time.Time
	// The account was answered once: the next answer waits for its span to lapse.
	answered bool
}

// Tried keeps the try, and answers the account the caller plays in turns with on its scope: one that tried after the
// caller's first try inside the window, then the caller again. A caller that only follows another, as a guest that
// signs in to its account does, plays in turns with nobody. Each span of an account is answered once.
func (t *Tracker) Tried(click detect.Click) (string, bool) {
	if click.Scope == "" || click.Account == "" {
		return "", false
	}

	rules := t.rules()

	t.mu.Lock()
	defer t.mu.Unlock()

	accounts := t.scopes[click.Scope]
	if accounts == nil {
		accounts = make(map[string]*span)
		t.scopes[click.Scope] = accounts
	}
	lapse(accounts, click.At.Add(-rules.Window))

	caller, ok := accounts[click.Account]
	if !ok {
		caller = &span{first: click.At, last: click.At}
		accounts[click.Account] = caller
	}
	if click.At.Before(caller.first) {
		caller.first = click.At
	}
	if click.At.After(caller.last) {
		caller.last = click.At
	}

	if !rules.Enabled || caller.answered || len(accounts) > rules.MaxAccounts {
		return "", false
	}

	alongside, latest := "", time.Time{}
	for account, other := range accounts {
		if account == click.Account || !other.last.After(caller.first) || other.last.After(click.At) {
			continue
		}
		if other.last.After(latest) {
			alongside, latest = account, other.last
		}
	}
	if alongside == "" {
		return "", false
	}

	caller.answered = true
	return alongside, true
}

func lapse(accounts map[string]*span, before time.Time) {
	for account, s := range accounts {
		if s.last.Before(before) {
			delete(accounts, account)
		}
	}
}

func (t *Tracker) Run(ctx context.Context) {
	ticker := time.NewTicker(sweepInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			t.sweep()
		case <-ctx.Done():
			return
		}
	}
}

func (t *Tracker) sweep() {
	before := t.clock.Now().Add(-t.rules().Window)

	t.mu.Lock()
	defer t.mu.Unlock()

	for scope, accounts := range t.scopes {
		lapse(accounts, before)
		if len(accounts) == 0 {
			delete(t.scopes, scope)
		}
	}
}
