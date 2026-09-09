package ctxutil

import "context"

type sessionIDKey struct{}

// GetSessionID is empty for a caller that presented no session, or one the
// server did not accept. It identifies a mint, not a person.
func GetSessionID(ctx context.Context) string {
	id, _ := ctx.Value(sessionIDKey{}).(string)
	return id
}

func AddSessionIDToContext(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, sessionIDKey{}, id)
}
