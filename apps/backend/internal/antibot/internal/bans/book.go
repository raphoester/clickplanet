package bans

import (
	"context"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Caller struct {
	Scope   string
	Account string

	SignedIn bool
}

func NewBook(
	config Config,
	pardon time.Duration,
	clock cptime.Clock,
	scopes, accounts Store,
	transactioner cpatomic.Transactioner,
	enforcer Enforcer,
) *Book {
	return &Book{
		scopes:   New(config, pardon, clock, Scopes, scopes, transactioner, enforcer),
		accounts: New(config, pardon, clock, Accounts, accounts, transactioner, enforcer),
	}
}

type Book struct {
	scopes   *Banner
	accounts *Banner
}

func (b *Book) Flag(ctx context.Context, caller Caller) (Sentence, bool, error) {
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

func (b *Book) Ban(ctx context.Context, caller Caller, duration time.Duration) (Sentence, error) {
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

func (b *Book) Unban(ctx context.Context, caller Caller) error {
	if err := b.scopes.Unban(ctx, caller.Scope); err != nil {
		return err
	}
	return b.accounts.Unban(ctx, caller.Account)
}

func (b *Book) Sentence(ctx context.Context, caller Caller) (Sentence, bool, error) {
	return b.sentenceOf(ctx, caller, (*Banner).Sentence)
}

func (b *Book) LastSentence(ctx context.Context, caller Caller) (Sentence, bool, error) {
	return b.sentenceOf(ctx, caller, (*Banner).LastSentence)
}

func (b *Book) sentenceOf(
	ctx context.Context,
	caller Caller,
	read func(*Banner, context.Context, string) (Sentence, bool, error),
) (Sentence, bool, error) {
	scope, onScope, err := read(b.scopes, ctx, caller.Scope)
	if err != nil {
		return Sentence{}, false, err
	}
	account, onAccount, err := read(b.accounts, ctx, caller.Account)
	if err != nil {
		return Sentence{}, false, err
	}

	return latest(scope, account), onScope || onAccount, nil
}

func (b *Book) Pardoned(ctx context.Context, caller Caller) (bool, error) {
	for _, target := range b.targets(caller) {
		_, pardoned, err := target.banner.Pardon(ctx, target.key)
		if err != nil || !pardoned {
			return false, err
		}
	}
	return true, nil
}

func (b *Book) Pardon(ctx context.Context, caller Caller) (Pardon, bool, error) {
	scope, onScope, err := b.scopes.Pardon(ctx, caller.Scope)
	if err != nil {
		return Pardon{}, false, err
	}
	account, onAccount, err := b.accounts.Pardon(ctx, caller.Account)
	if err != nil {
		return Pardon{}, false, err
	}

	if onAccount && account.Until.After(scope.Until) {
		return account, true, nil
	}
	return scope, onScope, nil
}

func (b *Book) Flagged(ctx context.Context) (int, error) {
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

func (b *Book) Enforcing() bool { return b.scopes.Enforcing() }

func (b *Book) Restate(ctx context.Context) error {
	if err := b.scopes.Restate(ctx); err != nil {
		return err
	}
	return b.accounts.Restate(ctx)
}

type target struct {
	banner *Banner
	key    string
}

func (b *Book) targets(caller Caller) []target {
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
