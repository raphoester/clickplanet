package cpctx

import "context"

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
