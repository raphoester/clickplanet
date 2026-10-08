package shadowban

import (
	"context"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Caller struct {
	Scope   string
	Account string

	SignedIn bool
}

func NewBans(config Config, clock cptime.Clock, scopes, accounts Store) *Bans {
	return &Bans{
		scopes:   New(config, clock, scopes),
		accounts: New(config, clock, accounts),
	}
}

type Bans struct {
	scopes   *Banner
	accounts *Banner
}

func (b *Bans) Flag(ctx context.Context, caller Caller) (Sentence, bool, error) {
	var (
		sentence Sentence
		accepted bool
	)

	for _, target := range b.targets(caller) {
		s, ok, err := target.banner.Flag(ctx, target.key)
		if err != nil {
			return sentence, accepted, err
		}
		if ok {
			sentence, accepted = latest(sentence, s), true
		}
	}

	return sentence, accepted, nil
}

func (b *Bans) Ban(ctx context.Context, caller Caller, duration time.Duration) (Sentence, error) {
	var sentence Sentence
	for _, target := range b.targets(caller) {
		s, err := target.banner.Ban(ctx, target.key, duration)
		if err != nil {
			return sentence, err
		}
		sentence = latest(sentence, s)
	}

	return sentence, nil
}

func (b *Bans) Unban(ctx context.Context, caller Caller) error {
	if err := b.scopes.Unban(ctx, caller.Scope); err != nil {
		return err
	}
	return b.accounts.Unban(ctx, caller.Account)
}

func (b *Bans) Banned(ctx context.Context, caller Caller) (bool, error) {
	if banned, err := b.scopes.Banned(ctx, caller.Scope); err != nil || banned {
		return banned, err
	}
	return b.accounts.Banned(ctx, caller.Account)
}

func (b *Bans) Sentence(ctx context.Context, caller Caller) (Sentence, bool, error) {
	scope, onScope, err := b.scopes.Sentence(ctx, caller.Scope)
	if err != nil {
		return Sentence{}, false, err
	}
	account, onAccount, err := b.accounts.Sentence(ctx, caller.Account)
	if err != nil {
		return Sentence{}, false, err
	}

	return latest(scope, account), onScope || onAccount, nil
}

func (b *Bans) Flagged(ctx context.Context) (int, error) {
	scopes, err := b.scopes.Flagged(ctx)
	if err != nil {
		return 0, err
	}
	accounts, err := b.accounts.Flagged(ctx)
	if err != nil {
		return 0, err
	}
	return scopes + accounts, nil
}

func (b *Bans) Enforcing() bool { return b.scopes.Enforcing() }

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
