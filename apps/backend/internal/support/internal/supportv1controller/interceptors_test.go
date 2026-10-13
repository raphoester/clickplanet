package supportv1controller

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	supportv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1/supportv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpconnect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
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

type tokenVerifier struct {
	claims cpsession.Claims
}

func (v tokenVerifier) Verify(context.Context, string, string, time.Time) (*cpsession.Claims, error) {
	return &v.claims, nil
}

func run(ctx context.Context, interceptor connect.Interceptor, procedure string, token string) (string, bool, error) {
	var (
		account string
		ran     bool
	)
	next := connect.UnaryFunc(func(ctx context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
		account, ran = cpctx.GetAccount(ctx), true
		return connect.NewResponse(&supportv1.SuggestFeatureResponse{}), nil
	})

	inner := connect.NewRequest(&supportv1.SuggestFeatureRequest{})
	if token != "" {
		inner.Header().Set(cpconnect.SessionHeader, token)
	}
	_, err := interceptor.WrapUnary(next)(ctx, fakeRequest{AnyRequest: inner, spec: connect.Spec{Procedure: procedure}})

	return account, ran, err
}

var procedures = []string{
	supportv1connect.SupportServiceSuggestFeatureProcedure,
	supportv1connect.SupportServiceReportBugProcedure,
}

func TestEveryProcedureIsThrottledOnTheSourceAddress(t *testing.T) {
	for _, procedure := range procedures {
		limiter := &fakeLimiter{allow: false}
		ctx := cpctx.AddIPToContext(t.Context(), "1.2.3.4")

		_, ran, err := run(ctx, NewRateLimitInterceptor(limiter), procedure, "")

		require.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err), procedure)
		require.False(t, ran, procedure)
		require.Equal(t, []string{"1.2.3.4"}, limiter.keys, procedure)
	}
}

func TestAnAllowedCallGoesThrough(t *testing.T) {
	_, ran, err := run(t.Context(), NewRateLimitInterceptor(&fakeLimiter{allow: true}), procedures[0], "")

	require.NoError(t, err)
	require.True(t, ran)
}

func TestEveryProcedureNeedsATokenMintedAfterACheck(t *testing.T) {
	ada := cpsession.AccountID{15: 1}

	for _, procedure := range procedures {
		attested := NewSessionInterceptor(
			tokenVerifier{claims: cpsession.Claims{ID: "a-mint", Account: ada, Attested: true}},
			cptime.SystemClock{}, prometheus.NewRegistry())
		identified := NewSessionInterceptor(
			tokenVerifier{claims: cpsession.Claims{ID: "a-resume", Account: ada}},
			cptime.SystemClock{}, prometheus.NewRegistry())

		account, ran, err := run(t.Context(), attested, procedure, "a-click-token")
		require.NoError(t, err, procedure)
		require.True(t, ran, procedure)
		require.Equal(t, ada.String(), account, procedure)

		_, ran, err = run(t.Context(), identified, procedure, "an-identity-token")
		require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err), procedure)
		require.False(t, ran, procedure)

		_, ran, err = run(t.Context(), attested, procedure, "")
		require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err), procedure)
		require.False(t, ran, procedure)
	}
}
