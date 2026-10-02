package shadowban

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Caller struct {
	Scope   string
	Account string

	SignedIn bool
}

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

func (b *Bans) Ban(caller Caller, duration time.Duration) Sentence {
	var sentence Sentence
	for _, target := range b.targets(caller) {
		sentence = latest(sentence, target.banner.Ban(target.key, duration))
	}

	return sentence
}

func (b *Bans) Banned(caller Caller) bool {
	return b.scopes.Banned(caller.Scope) || b.accounts.Banned(caller.Account)
}

func (b *Bans) Sentence(caller Caller) (Sentence, bool) {
	scope, onScope := b.scopes.Sentence(caller.Scope)
	account, onAccount := b.accounts.Sentence(caller.Account)

	return latest(scope, account), onScope || onAccount
}

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
	// A guest sheds its account with a new cookie, so its ban falls on the scope too.
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
