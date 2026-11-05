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
	supportv1connect.SupportServiceReportPlayerProcedure,
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

func TestThePageAsksWhatItMayAttachWithNoSessionAndSpendsNothing(t *testing.T) {
	limiter := &fakeLimiter{allow: false}
	session := NewSessionInterceptor(tokenVerifier{}, cptime.SystemClock{}, prometheus.NewRegistry())

	_, ran, err := run(t.Context(), session, supportv1connect.SupportServiceGetAttachmentRulesProcedure, "")
	require.NoError(t, err)
	require.True(t, ran)

	_, ran, err = run(t.Context(), NewRateLimitInterceptor(limiter), supportv1connect.SupportServiceGetAttachmentRulesProcedure, "")
	require.NoError(t, err)
	require.True(t, ran)
	require.Empty(t, limiter.keys)
}

func TestAnUploadNeedsATokenMintedAfterACheck(t *testing.T) {
	ada := cpsession.AccountID{15: 1}
	attested := NewSessionInterceptor(
		tokenVerifier{claims: cpsession.Claims{ID: "a-mint", Account: ada, Attested: true}},
		cptime.SystemClock{}, prometheus.NewRegistry())
	identified := NewSessionInterceptor(
		tokenVerifier{claims: cpsession.Claims{ID: "a-resume", Account: ada}},
		cptime.SystemClock{}, prometheus.NewRegistry())

	_, ran, err := run(t.Context(), attested, supportv1connect.SupportServiceCreateUploadProcedure, "a-click-token")
	require.NoError(t, err)
	require.True(t, ran)

	_, ran, err = run(t.Context(), identified, supportv1connect.SupportServiceCreateUploadProcedure, "an-identity-token")
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	require.False(t, ran)

	_, ran, err = run(t.Context(), attested, supportv1connect.SupportServiceCreateUploadProcedure, "")
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	require.False(t, ran)
}

// Two a day.
func dailyLimiter() *cpratelimit.Limiter {
	return cpratelimit.New("daily", cpratelimit.Config{Burst: 2, PerSecond: 2.0 / 86400},
		cptime.NewFixedClock(time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)))
}

func caller(ctx context.Context, account string, ip string) context.Context {
	return cpctx.AddAccountToContext(cpctx.AddIPToContext(ctx, ip), account)
}

func TestADaysAllowanceIsTheAccountsOnEveryAddressAndTheAddresssForEveryAccount(t *testing.T) {
	interceptor := NewNoteLimitInterceptor(dailyLimiter())
	sent := func(account string, ip string) error {
		_, _, err := run(caller(t.Context(), account, ip), interceptor, supportv1connect.SupportServiceReportBugProcedure, "")
		return err
	}

	require.NoError(t, sent("ada", "1.2.3.4"))
	require.NoError(t, sent("ada", "1.2.3.4"))
	require.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(sent("ada", "5.6.7.8")), "the account's, on another address")
	require.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(sent("bob", "1.2.3.4")), "the address's, for another account")

	require.NoError(t, sent("bob", "5.6.7.8"), "a refusal spent nothing of the address")
	require.NoError(t, sent("eve", "5.6.7.8"))
	require.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(sent("zoe", "5.6.7.8")))
}

func TestTheNotesOfADayAreSuggestionsAndBugReportsAndTheFilesAreUploads(t *testing.T) {
	limiter := dailyLimiter()
	notes, uploads := NewNoteLimitInterceptor(limiter), NewUploadLimitInterceptor(limiter)
	ctx := caller(t.Context(), "ada", "1.2.3.4")

	for _, procedure := range []string{
		supportv1connect.SupportServiceReportPlayerProcedure,
		supportv1connect.SupportServiceCreateUploadProcedure,
		supportv1connect.SupportServiceGetAttachmentRulesProcedure,
	} {
		_, ran, err := run(ctx, notes, procedure, "")
		require.NoError(t, err, procedure)
		require.True(t, ran, procedure)
	}
	for _, procedure := range []string{
		supportv1connect.SupportServiceSuggestFeatureProcedure,
		supportv1connect.SupportServiceReportBugProcedure,
		supportv1connect.SupportServiceReportPlayerProcedure,
	} {
		_, ran, err := run(ctx, uploads, procedure, "")
		require.NoError(t, err, procedure)
		require.True(t, ran, procedure)
	}

	_, _, err := run(ctx, notes, supportv1connect.SupportServiceSuggestFeatureProcedure, "")
	require.NoError(t, err)
	_, _, err = run(ctx, notes, supportv1connect.SupportServiceReportBugProcedure, "")
	require.NoError(t, err)
	_, ran, err := run(ctx, notes, supportv1connect.SupportServiceSuggestFeatureProcedure, "")
	require.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err))
	require.ErrorIs(t, err, ErrTooManyNotes)
	require.False(t, ran)
}

func TestACallerWithNoAccountSpendsTheAddresssAllowanceAlone(t *testing.T) {
	interceptor := NewUploadLimitInterceptor(dailyLimiter())
	ctx := cpctx.AddIPToContext(t.Context(), "1.2.3.4")

	for range 2 {
		_, _, err := run(ctx, interceptor, supportv1connect.SupportServiceCreateUploadProcedure, "")
		require.NoError(t, err)
	}
	_, _, err := run(ctx, interceptor, supportv1connect.SupportServiceCreateUploadProcedure, "")
	require.ErrorIs(t, err, ErrTooManyUploads)
}
