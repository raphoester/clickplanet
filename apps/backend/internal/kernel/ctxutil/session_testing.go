//go:build testing

package ctxutil

import "context"

// GetSessionID reads back what AddSessionIDToContext put there. The key is
// unexported, so a test in another package has no other way to assert that the
// session interceptor did its job.
//
// It is behind the `testing` tag because today the tests are its only
// caller: nothing in production reads the session id yet. When the behavioural
// signals it exists for arrive — timing regularity and tile-id structure over a
// session — this moves back into session.go and the tag comes off.
//
// It is empty for a caller that presented no session, or one the server did not
// accept. It identifies a mint, not a person.
func GetSessionID(ctx context.Context) string {
	id, _ := ctx.Value(sessionIDKey{}).(string)
	return id
}
