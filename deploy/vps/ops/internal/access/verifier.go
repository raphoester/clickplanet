package access

import "context"

type Caller string

type Verifier interface {
	Verify(ctx context.Context, assertion string) (Caller, error)
}

type callerKey struct{}

func WithCaller(ctx context.Context, caller Caller) context.Context {
	return context.WithValue(ctx, callerKey{}, caller)
}

func CallerFrom(ctx context.Context) Caller {
	caller, _ := ctx.Value(callerKey{}).(Caller)
	return caller
}
