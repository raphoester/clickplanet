package chatv1controller

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/prometheus/client_golang/prometheus"
	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1/chatv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpconnect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpipblock"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
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

type fakeBlocklist struct {
	blocked map[string]cpipblock.List
	asked   []string
}

func (b *fakeBlocklist) Blocked(ip string) (cpipblock.List, bool) {
	b.asked = append(b.asked, ip)
	list, ok := b.blocked[ip]
	return list, ok
}

func TestVPNBlockInterceptor(t *testing.T) {
	t.Run("refuses a post and a reaction from a listed address, and counts them by list", func(t *testing.T) {
		registry := prometheus.NewRegistry()
		interceptor := NewVPNBlockInterceptor(
			&fakeBlocklist{blocked: map[string]cpipblock.List{"1.2.3.4": cpipblock.ListWARP}}, registry)
		ctx := cpctx.AddIPToContext(t.Context(), "1.2.3.4")

		for _, procedure := range []string{
			chatv1connect.ChatServiceSendMessageProcedure,
			chatv1connect.ChatServiceReactProcedure,
		} {
			ran, err := run(ctx, interceptor, procedure)

			require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err), procedure)
			require.ErrorIs(t, err, ErrSenderBlocked, "the operator's blocklist's answer: no list is named")
			require.False(t, ran, "a refused post or reaction must not reach the domain")
		}

		require.InDelta(t, 2.0, blockedChat(t, registry, "warp"), 1e-9)
	})

	t.Run("leaves reading the chat alone", func(t *testing.T) {
		blocklist := &fakeBlocklist{blocked: map[string]cpipblock.List{"1.2.3.4": cpipblock.ListVPN}}
		ctx := cpctx.AddIPToContext(t.Context(), "1.2.3.4")

		for _, procedure := range []string{
			chatv1connect.ChatServiceGetHistoryProcedure,
			chatv1connect.ChatServiceListenForEventsProcedure,
			chatv1connect.ChatServiceMarkSeenProcedure,
		} {
			ran, err := run(ctx, NewVPNBlockInterceptor(blocklist, prometheus.NewRegistry()), procedure)

			require.NoError(t, err, "a VPN user still reads the chat")
			require.True(t, ran)
		}

		require.Empty(t, blocklist.asked, "a read is never even offered to the lists")
	})

	t.Run("lets an address on no list through", func(t *testing.T) {
		blocklist := &fakeBlocklist{blocked: map[string]cpipblock.List{"1.2.3.4": cpipblock.ListVPN}}
		ctx := cpctx.AddIPToContext(t.Context(), "5.6.7.8")

		ran, err := run(ctx, NewVPNBlockInterceptor(blocklist, prometheus.NewRegistry()),
			chatv1connect.ChatServiceSendMessageProcedure)

		require.NoError(t, err)
		require.True(t, ran)
	})

	t.Run("an unresolved address is passed through", func(t *testing.T) {
		blocklist := &fakeBlocklist{blocked: map[string]cpipblock.List{"": cpipblock.ListVPN}}

		ran, err := run(t.Context(), NewVPNBlockInterceptor(blocklist, prometheus.NewRegistry()),
			chatv1connect.ChatServiceReactProcedure)

		require.NoError(t, err)
		require.True(t, ran)
	})

	t.Run("the lists turned off refuse nobody", func(t *testing.T) {
		ctx := cpctx.AddIPToContext(t.Context(), "2a09:bac5:d594:2682::1")

		ran, err := run(ctx, NewVPNBlockInterceptor(cpipblock.New(cpipblock.Config{}), prometheus.NewRegistry()),
			chatv1connect.ChatServiceSendMessageProcedure)

		require.NoError(t, err)
		require.True(t, ran)
	})
}

func blockedChat(t *testing.T, gatherer prometheus.Gatherer, list string) float64 {
	t.Helper()

	families, err := gatherer.Gather()
	require.NoError(t, err)

	for _, family := range families {
		if family.GetName() != "blocked_chat" {
			continue
		}
		for _, metric := range family.GetMetric() {
			for _, label := range metric.GetLabel() {
				if label.GetName() == "list" && label.GetValue() == list {
					return metric.GetCounter().GetValue()
				}
			}
		}
	}

	return 0
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

type identityVerifier struct{}

func (identityVerifier) Verify(context.Context, string, string, time.Time) (*cpsession.Claims, error) {
	return &cpsession.Claims{ID: "a-mint", Account: cpsession.AccountID{15: 1}}, nil
}

func TestAnIdentityTokenNamesTheReaderAndTheSeenMarkOnly(t *testing.T) {
	for procedure, named := range map[string]bool{
		chatv1connect.ChatServiceGetHistoryProcedure:  true,
		chatv1connect.ChatServiceMarkSeenProcedure:    true,
		chatv1connect.ChatServiceSendMessageProcedure: false,
		chatv1connect.ChatServiceReactProcedure:       false,
	} {
		var account string
		next := connect.UnaryFunc(func(ctx context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
			account = cpctx.GetAccount(ctx)
			return connect.NewResponse(&chatv1.GetHistoryResponse{}), nil
		})
		inner := connect.NewRequest(&chatv1.GetHistoryRequest{})
		inner.Header().Set(cpconnect.SessionHeader, "an-identity-token")

		_, err := NewSessionInterceptor(identityVerifier{}, cptime.SystemClock{}).WrapUnary(next)(t.Context(),
			fakeRequest{AnyRequest: inner, spec: connect.Spec{Procedure: procedure}})

		require.NoError(t, err)
		require.Equal(t, named, account != "", procedure)
	}
}
