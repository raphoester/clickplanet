package ctxutil

import "context"

type sessionIDKey struct{}

func AddSessionIDToContext(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, sessionIDKey{}, id)
}
