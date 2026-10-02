//go:build testing

package cpctx

import "context"

func GetSessionID(ctx context.Context) string {
	id, _ := ctx.Value(sessionIDKey{}).(string)
	return id
}
