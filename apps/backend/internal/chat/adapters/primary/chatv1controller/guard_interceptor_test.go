package chatv1controller

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1/chatv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/ctxutil"
	"github.com/stretchr/testify/require"
)

type fakeLimiter struct {
	allow bool
	keys  []string
}

func (l *fakeLimiter) Allow(key string) bool {
	l.keys = append(l.keys, key)
	return l.allow
}

// fakeRequest names a procedure, which is all the interceptor reads. The server
// fills the spec in for real; connect.NewRequest builds a client-side request
// whose spec is empty and read-only.
type fakeRequest struct {
	connect.AnyRequest
	spec connect.Spec
}

func (r fakeRequest) Spec() connect.Spec {
	return r.spec
}

func guard(
	ctx context.Context,
	limiter MessageLimiter,
	blockedIPs []string,
	procedure string,
) (bool, error) {
	handlerRan := false
	next := connect.UnaryFunc(func(context.Context, connect.AnyRequest) (connect.AnyResponse, error) {
		handlerRan = true
		return connect.NewResponse(&chatv1.SendMessageResponse{}), nil
	})

	req := fakeRequest{spec: connect.Spec{Procedure: procedure}}

	_, err := NewGuardInterceptor(limiter, blockedIPs).WrapUnary(next)(ctx, req)

	return handlerRan, err
}

func TestGuardInterceptor(t *testing.T) {
	t.Run("lets an allowed message through", func(t *testing.T) {
		ran, err := guard(context.Background(), &fakeLimiter{allow: true}, nil,
			chatv1connect.ChatServiceSendMessageProcedure)

		require.NoError(t, err)
		require.True(t, ran)
	})

	t.Run("refuses a message over the limit without reaching the domain", func(t *testing.T) {
		ran, err := guard(context.Background(), &fakeLimiter{allow: false}, nil,
			chatv1connect.ChatServiceSendMessageProcedure)

		require.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err))
		require.False(t, ran, "a refused message must not reach the domain")
	})

	t.Run("keys on the source IP from the context", func(t *testing.T) {
		limiter := &fakeLimiter{allow: true}
		ctx := ctxutil.AddIPToContext(context.Background(), "1.2.3.4")

		_, err := guard(ctx, limiter, nil, chatv1connect.ChatServiceSendMessageProcedure)

		require.NoError(t, err)
		require.Equal(t, []string{"1.2.3.4"}, limiter.keys)
	})

	t.Run("does not throttle the history read", func(t *testing.T) {
		limiter := &fakeLimiter{allow: false}

		ran, err := guard(context.Background(), limiter, nil,
			chatv1connect.ChatServiceGetHistoryProcedure)

		require.NoError(t, err)
		require.True(t, ran)
		require.Empty(t, limiter.keys, "a join is never even offered to the limiter")
	})

	t.Run("cuts a blocked sender off from every procedure", func(t *testing.T) {
		limiter := &fakeLimiter{allow: true}
		ctx := ctxutil.AddIPToContext(context.Background(), "9.9.9.9")

		for _, procedure := range []string{
			chatv1connect.ChatServiceSendMessageProcedure,
			chatv1connect.ChatServiceGetHistoryProcedure,
		} {
			ran, err := guard(ctx, limiter, []string{" 9.9.9.9 "}, procedure)

			require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
			require.False(t, ran)
		}

		require.Empty(t, limiter.keys, "a blocked sender should not even consume a token")
	})

	t.Run("an empty block list entry blocks nobody", func(t *testing.T) {
		// A caller whose IP the middleware could not resolve carries "", which
		// must not collide with a stray blank line in the config.
		ran, err := guard(context.Background(), &fakeLimiter{allow: true}, []string{"", "  "},
			chatv1connect.ChatServiceSendMessageProcedure)

		require.NoError(t, err)
		require.True(t, ran)
	})
}
