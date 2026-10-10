package access

import (
	"context"
	"errors"
)

var ErrAnonymous = errors.New("the assertion names no caller")

type Caller string

type callerKey struct{}

func WithCaller(ctx context.Context, caller Caller) context.Context {
	return context.WithValue(ctx, callerKey{}, caller)
}

func CallerOf(ctx context.Context) Caller {
	caller, _ := ctx.Value(callerKey{}).(Caller)
	return caller
}
