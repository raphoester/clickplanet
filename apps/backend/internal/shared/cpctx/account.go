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
