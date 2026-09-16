package cpctx

import "context"

type sessionIDKey struct{}

func AddSessionIDToContext(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, sessionIDKey{}, id)
}

// GetSessionID reads back what AddSessionIDToContext put there. It is empty for
// a caller that presented no session, or one the server did not accept.
//
// It identifies a mint, not a person: two page loads by the same player are two
// ids, and a shared address is one scope with many. Its production reader is
// the antibot's re-challenge, which compares it against the id a caller was
// challenged on to tell one that went back through the mint from one that did
// not. Nothing keys a rule on it.
func GetSessionID(ctx context.Context) string {
	id, _ := ctx.Value(sessionIDKey{}).(string)
	return id
}
