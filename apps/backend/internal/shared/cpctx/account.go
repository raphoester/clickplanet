package cpctx

import (
	"context"
	"time"
)

type accountKey struct{}

func AddAccountToContext(ctx context.Context, account string) context.Context {
	return context.WithValue(ctx, accountKey{}, account)
}

func GetAccount(ctx context.Context) string {
	account, _ := ctx.Value(accountKey{}).(string)
	return account
}

type accountCreatedKey struct{}

func AddAccountCreatedToContext(ctx context.Context, at time.Time) context.Context {
	return context.WithValue(ctx, accountCreatedKey{}, at)
}

func GetAccountCreated(ctx context.Context) time.Time {
	at, _ := ctx.Value(accountCreatedKey{}).(time.Time)
	return at
}

type linkedKey struct{}

func AddLinkedToContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, linkedKey{}, true)
}

func GetLinked(ctx context.Context) bool {
	linked, _ := ctx.Value(linkedKey{}).(bool)
	return linked
}
