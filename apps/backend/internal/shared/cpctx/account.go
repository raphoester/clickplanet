package cpctx

import (
	"context"
	"time"
)

type accountKey struct{}

// AddAccountToContext keeps the account a verified click token names.
func AddAccountToContext(ctx context.Context, account string) context.Context {
	return context.WithValue(ctx, accountKey{}, account)
}

// GetAccount is empty for a caller whose token names no account, or that sent no valid token.
func GetAccount(ctx context.Context) string {
	account, _ := ctx.Value(accountKey{}).(string)
	return account
}

type accountCreatedKey struct{}

// AddAccountCreatedToContext keeps when the account a verified click token names was made.
func AddAccountCreatedToContext(ctx context.Context, at time.Time) context.Context {
	return context.WithValue(ctx, accountCreatedKey{}, at)
}

// GetAccountCreated is the zero time when no account is known, or its id does not say.
func GetAccountCreated(ctx context.Context) time.Time {
	at, _ := ctx.Value(accountCreatedKey{}).(time.Time)
	return at
}

type linkedKey struct{}

// AddLinkedToContext keeps that the account a verified click token names signed in with a provider.
func AddLinkedToContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, linkedKey{}, true)
}

// GetLinked is false for a guest, a caller whose token names no account, or one that sent no valid token.
func GetLinked(ctx context.Context) bool {
	linked, _ := ctx.Value(linkedKey{}).(bool)
	return linked
}
