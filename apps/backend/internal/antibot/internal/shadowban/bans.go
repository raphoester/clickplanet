package shadowban

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

// Caller is who a ban falls on: a scope, the account its token names, or both.
type Caller struct {
	Scope   string
	Account string

	// SignedIn is an account a provider vouches for. A guest can shed its account with a new cookie, so a
	// ban on a guest falls on its scope too.
	SignedIn bool
}

// NewBans keeps scopes and accounts apart, each with its own ladder and its own table.
func NewBans(config Config, clock cptime.Clock, scopes, accounts Persistence, onStateError func(error)) *Bans {
	return &Bans{
		scopes:   New(config, clock, scopes, onStateError),
		accounts: New(config, clock, accounts, onStateError),
	}
}

type Bans struct {
	scopes   *Banner
	accounts *Banner
}

// Flag passes the ban on the account, and on the scope when there is no account or it is a guest's.
func (b *Bans) Flag(caller Caller) (Sentence, bool) {
	var (
		sentence Sentence
		accepted bool
	)

	for _, target := range b.targets(caller) {
		if s, ok := target.banner.Flag(target.key); ok {
			sentence, accepted = latest(sentence, s), true
		}
	}

	return sentence, accepted
}

// Ban is an operator's ban, passed on the same targets as a flag.
func (b *Bans) Ban(caller Caller, duration time.Duration) Sentence {
	var sentence Sentence
	for _, target := range b.targets(caller) {
		sentence = latest(sentence, target.banner.Ban(target.key, duration))
	}

	return sentence
}

// Banned is true when the scope or the account is banned, and enforce is on.
func (b *Bans) Banned(caller Caller) bool {
	return b.scopes.Banned(caller.Scope) || b.accounts.Banned(caller.Account)
}

// Sentence is the running ban on the scope or the account that ends last.
func (b *Bans) Sentence(caller Caller) (Sentence, bool) {
	scope, onScope := b.scopes.Sentence(caller.Scope)
	account, onAccount := b.accounts.Sentence(caller.Account)

	return latest(scope, account), onScope || onAccount
}

// Flagged counts running bans on scopes and on accounts, so a guest banned on both counts twice.
func (b *Bans) Flagged() int {
	return b.scopes.Flagged() + b.accounts.Flagged()
}

func (b *Bans) Enforcing() bool { return b.scopes.Enforcing() }

func (b *Bans) Load(ctx context.Context) error {
	return errors.Join(b.scopes.Load(ctx), b.accounts.Load(ctx))
}

func (b *Bans) Run(ctx context.Context) {
	var wg sync.WaitGroup
	for _, banner := range []*Banner{b.scopes, b.accounts} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			banner.Run(ctx)
		}()
	}
	wg.Wait()
}

type target struct {
	banner *Banner
	key    string
}

func (b *Bans) targets(caller Caller) []target {
	targets := make([]target, 0, 2)
	if caller.Account != "" {
		targets = append(targets, target{banner: b.accounts, key: caller.Account})
	}
	if caller.Account == "" || !caller.SignedIn {
		targets = append(targets, target{banner: b.scopes, key: caller.Scope})
	}

	return targets
}

func latest(a, b Sentence) Sentence {
	if b.Until.After(a.Until) {
		return b
	}
	return a
}
