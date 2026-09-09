package chatv1controller

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1/chatv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/ctxutil"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/ipblock"
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

// fakeRequest names a procedure, which is all the interceptors read. The server
// fills the spec in for real; connect.NewRequest builds a client-side request
// whose spec is empty and read-only.
type fakeRequest struct {
	connect.AnyRequest
	spec connect.Spec
}

func (r fakeRequest) Spec() connect.Spec {
	return r.spec
}

func run(ctx context.Context, interceptor connect.Interceptor, procedure string) (bool, error) {
	handlerRan := false
	next := connect.UnaryFunc(func(context.Context, connect.AnyRequest) (connect.AnyResponse, error) {
		handlerRan = true
		return connect.NewResponse(&chatv1.SendMessageResponse{}), nil
	})

	req := fakeRequest{spec: connect.Spec{Procedure: procedure}}
	_, err := interceptor.WrapUnary(next)(ctx, req)

	return handlerRan, err
}

func TestRateLimitInterceptor(t *testing.T) {
	t.Run("lets an allowed message through", func(t *testing.T) {
		ran, err := run(context.Background(),
			NewRateLimitInterceptor(&fakeLimiter{allow: true}),
			chatv1connect.ChatServiceSendMessageProcedure)

		require.NoError(t, err)
		require.True(t, ran)
	})

	t.Run("refuses a message over the limit without reaching the domain", func(t *testing.T) {
		ran, err := run(context.Background(),
			NewRateLimitInterceptor(&fakeLimiter{allow: false}),
			chatv1connect.ChatServiceSendMessageProcedure)

		require.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err))
		require.False(t, ran, "a refused message must not reach the domain")
	})

	t.Run("keys on the source IP from the context", func(t *testing.T) {
		limiter := &fakeLimiter{allow: true}
		ctx := ctxutil.AddIPToContext(context.Background(), "1.2.3.4")

		_, err := run(ctx, NewRateLimitInterceptor(limiter),
			chatv1connect.ChatServiceSendMessageProcedure)

		require.NoError(t, err)
		require.Equal(t, []string{"1.2.3.4"}, limiter.keys)
	})

	t.Run("does not throttle the history read", func(t *testing.T) {
		limiter := &fakeLimiter{allow: false}

		ran, err := run(context.Background(), NewRateLimitInterceptor(limiter),
			chatv1connect.ChatServiceGetHistoryProcedure)

		require.NoError(t, err)
		require.True(t, ran)
		require.Empty(t, limiter.keys, "a join is never even offered to the limiter")
	})
}

func TestBlocklistInterceptor(t *testing.T) {
	blocklistOf(t, nil) // sanity: an empty list builds

	t.Run("cuts a blocked sender off from every procedure", func(t *testing.T) {
		blocklist := blocklistOf(t, []string{"9.9.9.9/32"})
		ctx := ctxutil.AddIPToContext(context.Background(), "9.9.9.9")

		for _, procedure := range []string{
			chatv1connect.ChatServiceSendMessageProcedure,
			chatv1connect.ChatServiceGetHistoryProcedure,
		} {
			ran, err := run(ctx, NewBlocklistInterceptor(blocklist), procedure)

			require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
			require.False(t, ran)
		}
	})

	// The point of building on ipblock rather than matching strings: one line
	// covers a range, so a spammer cycling addresses inside a /24 goes in one
	// entry instead of 256.
	t.Run("blocks a whole prefix", func(t *testing.T) {
		blocklist := blocklistOf(t, []string{"203.0.113.0/24"})

		for _, ip := range []string{"203.0.113.1", "203.0.113.254"} {
			ctx := ctxutil.AddIPToContext(context.Background(), ip)
			_, err := run(ctx, NewBlocklistInterceptor(blocklist),
				chatv1connect.ChatServiceSendMessageProcedure)

			require.Equalf(t, connect.CodePermissionDenied, connect.CodeOf(err), "%s should be refused", ip)
		}

		ctx := ctxutil.AddIPToContext(context.Background(), "203.0.114.1")
		ran, err := run(ctx, NewBlocklistInterceptor(blocklist),
			chatv1connect.ChatServiceSendMessageProcedure)

		require.NoError(t, err, "an address outside the prefix is untouched")
		require.True(t, ran)
	})

	t.Run("an empty list blocks nobody", func(t *testing.T) {
		ctx := ctxutil.AddIPToContext(context.Background(), "9.9.9.9")

		ran, err := run(ctx, NewBlocklistInterceptor(blocklistOf(t, nil)),
			chatv1connect.ChatServiceSendMessageProcedure)

		require.NoError(t, err)
		require.True(t, ran)
	})

	// A caller whose address the middleware could not resolve carries "". The
	// list exists to refuse the known-bad, not everything it cannot read.
	t.Run("an unresolved address is passed through", func(t *testing.T) {
		ran, err := run(context.Background(), NewBlocklistInterceptor(blocklistOf(t, []string{"9.9.9.9/32"})),
			chatv1connect.ChatServiceSendMessageProcedure)

		require.NoError(t, err)
		require.True(t, ran)
	})
}

// A malformed entry fails at startup rather than silently blocking nobody,
// which is the same bargain the vendored lists make.
func TestABareAddressIsRejectedAtStartup(t *testing.T) {
	_, err := ipblock.NewDenyList([]string{"9.9.9.9"})
	require.Error(t, err, "entries are prefixes; a single address needs /32")
}

func blocklistOf(t *testing.T, prefixes []string) SenderBlocklist {
	t.Helper()

	blocklist, err := ipblock.NewDenyList(prefixes)
	require.NoError(t, err)

	return blocklist
}
