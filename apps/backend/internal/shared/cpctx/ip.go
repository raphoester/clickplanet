package cpctx

import "context"

type ipKey struct{}

func GetSourceIP(ctx context.Context) string {
	ip, _ := ctx.Value(ipKey{}).(string)
	return ip
}

func AddIPToContext(ctx context.Context, ip string) context.Context {
	newContext := context.WithValue(ctx, ipKey{}, ip)
	return newContext
}
