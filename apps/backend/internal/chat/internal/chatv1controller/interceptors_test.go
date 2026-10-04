package chatv1controller

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1/chatv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpipblock"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
	"github.com/stretchr/testify/require"
)

type fakeLimiter struct {
	allow bool
	keys  []string
}

func (l *fakeLimiter) Take(key string) (bool, cpratelimit.State) {
	l.keys = append(l.keys, key)
	return l.allow, cpratelimit.State{}
}

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
		ran, err := run(t.Context(),
			NewRateLimitInterceptor(&fakeLimiter{allow: true}),
			chatv1connect.ChatServiceSendMessageProcedure)

		require.NoError(t, err)
		require.True(t, ran)
	})

	t.Run("refuses a message over the limit without reaching the domain", func(t *testing.T) {
		ran, err := run(t.Context(),
			NewRateLimitInterceptor(&fakeLimiter{allow: false}),
			chatv1connect.ChatServiceSendMessageProcedure)

		require.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err))
		require.False(t, ran, "a refused message must not reach the domain")
	})

	t.Run("keys on the source IP from the context", func(t *testing.T) {
		limiter := &fakeLimiter{allow: true}
		ctx := cpctx.AddIPToContext(t.Context(), "1.2.3.4")

		_, err := run(ctx, NewRateLimitInterceptor(limiter),
			chatv1connect.ChatServiceSendMessageProcedure)

		require.NoError(t, err)
		require.Equal(t, []string{"1.2.3.4"}, limiter.keys)
	})

	t.Run("does not throttle the history read", func(t *testing.T) {
		limiter := &fakeLimiter{allow: false}

		ran, err := run(t.Context(), NewRateLimitInterceptor(limiter),
			chatv1connect.ChatServiceGetHistoryProcedure)

		require.NoError(t, err)
		require.True(t, ran)
		require.Empty(t, limiter.keys, "a join is never even offered to the limiter")
	})
}

func TestReactionRateLimitInterceptor(t *testing.T) {
	t.Run("refuses a reaction over its own limit", func(t *testing.T) {
		ran, err := run(t.Context(),
			NewReactionRateLimitInterceptor(&fakeLimiter{allow: false}),
			chatv1connect.ChatServiceReactProcedure)

		require.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err))
		require.False(t, ran)
	})

	t.Run("leaves messages to the message limiter", func(t *testing.T) {
		limiter := &fakeLimiter{allow: false}

		ran, err := run(t.Context(), NewReactionRateLimitInterceptor(limiter),
			chatv1connect.ChatServiceSendMessageProcedure)

		require.NoError(t, err)
		require.True(t, ran)
		require.Empty(t, limiter.keys)
	})
}

func TestBlocklistInterceptor(t *testing.T) {
	blocklistOf(t, nil)

	t.Run("cuts a blocked sender off from every procedure", func(t *testing.T) {
		blocklist := blocklistOf(t, []string{"9.9.9.9/32"})
		ctx := cpctx.AddIPToContext(t.Context(), "9.9.9.9")

		for _, procedure := range []string{
			chatv1connect.ChatServiceSendMessageProcedure,
			chatv1connect.ChatServiceGetHistoryProcedure,
			chatv1connect.ChatServiceReactProcedure,
		} {
			ran, err := run(ctx, NewBlocklistInterceptor(blocklist), procedure)

			require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
			require.False(t, ran)
		}
	})

	t.Run("blocks a whole prefix", func(t *testing.T) {
		blocklist := blocklistOf(t, []string{"203.0.113.0/24"})

		for _, ip := range []string{"203.0.113.1", "203.0.113.254"} {
			ctx := cpctx.AddIPToContext(t.Context(), ip)
			_, err := run(ctx, NewBlocklistInterceptor(blocklist),
				chatv1connect.ChatServiceSendMessageProcedure)

			require.Equalf(t, connect.CodePermissionDenied, connect.CodeOf(err), "%s should be refused", ip)
		}

		ctx := cpctx.AddIPToContext(t.Context(), "203.0.114.1")
		ran, err := run(ctx, NewBlocklistInterceptor(blocklist),
			chatv1connect.ChatServiceSendMessageProcedure)

		require.NoError(t, err, "an address outside the prefix is untouched")
		require.True(t, ran)
	})

	t.Run("an empty list blocks nobody", func(t *testing.T) {
		ctx := cpctx.AddIPToContext(t.Context(), "9.9.9.9")

		ran, err := run(ctx, NewBlocklistInterceptor(blocklistOf(t, nil)),
			chatv1connect.ChatServiceSendMessageProcedure)

		require.NoError(t, err)
		require.True(t, ran)
	})

	t.Run("an unresolved address is passed through", func(t *testing.T) {
		ran, err := run(t.Context(), NewBlocklistInterceptor(blocklistOf(t, []string{"9.9.9.9/32"})),
			chatv1connect.ChatServiceSendMessageProcedure)

		require.NoError(t, err)
		require.True(t, ran)
	})
}

func TestABareAddressIsRejectedAtStartup(t *testing.T) {
	_, err := cpipblock.NewDenyList([]string{"9.9.9.9"})
	require.Error(t, err, "entries are prefixes; a single address needs /32")
}

func blocklistOf(t *testing.T, prefixes []string) SenderBlocklist {
	t.Helper()

	blocklist, err := cpipblock.NewDenyList(prefixes)
	require.NoError(t, err)

	return blocklist
}

type stubCallers struct {
	account messages.AccountID
	err     error
	asked   []string
}

func (s *stubCallers) Caller(_ context.Context, cookie string) (messages.AccountID, error) {
	s.asked = append(s.asked, cookie)
	return s.account, s.err
}

func callerOf(ctx context.Context, callers *stubCallers, procedure string, cookie string) string {
	var account string
	next := connect.UnaryFunc(func(ctx context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
		account = cpctx.GetAccount(ctx)
		return connect.NewResponse(&chatv1.GetHistoryResponse{}), nil
	})

	inner := connect.NewRequest(&chatv1.GetHistoryRequest{})
	if cookie != "" {
		inner.Header().Set("Cookie", cookie)
	}
	_, _ = NewCookieReaderInterceptor(callers).WrapUnary(next)(ctx, fakeRequest{AnyRequest: inner, spec: connect.Spec{Procedure: procedure}})

	return account
}

func TestCookieReaderInterceptor(t *testing.T) {
	ada := messages.AccountID{15: 1}

	t.Run("names the caller the cookie names, for the history and the seen mark", func(t *testing.T) {
		for _, procedure := range []string{chatv1connect.ChatServiceGetHistoryProcedure, chatv1connect.ChatServiceMarkSeenProcedure} {
			callers := &stubCallers{account: ada}

			require.Equal(t, ada.String(), callerOf(t.Context(), callers, procedure, "cp_sid=token-1"), procedure)
			require.Equal(t, []string{"cp_sid=token-1"}, callers.asked)
		}
	})

	t.Run("does not ask when a token already named the caller", func(t *testing.T) {
		callers := &stubCallers{account: ada}
		ctx := cpctx.AddAccountToContext(t.Context(), "from-the-token")

		require.Equal(t, "from-the-token", callerOf(ctx, callers, chatv1connect.ChatServiceGetHistoryProcedure, "cp_sid=token-1"))
		require.Empty(t, callers.asked)
	})

	t.Run("does not ask with no cookie", func(t *testing.T) {
		callers := &stubCallers{account: ada}

		require.Empty(t, callerOf(t.Context(), callers, chatv1connect.ChatServiceGetHistoryProcedure, ""))
		require.Empty(t, callers.asked)
	})

	t.Run("does not ask for another procedure", func(t *testing.T) {
		callers := &stubCallers{account: ada}

		require.Empty(t, callerOf(t.Context(), callers, chatv1connect.ChatServiceSendMessageProcedure, "cp_sid=token-1"))
		require.Empty(t, callers.asked)
	})

	t.Run("goes on with no account when auth names nobody or cannot answer", func(t *testing.T) {
		for _, callers := range []*stubCallers{{account: messages.NoAccount}, {account: ada, err: errors.New("auth is down")}} {
			require.Empty(t, callerOf(t.Context(), callers, chatv1connect.ChatServiceGetHistoryProcedure, "cp_sid=token-1"))
		}
	})
}
