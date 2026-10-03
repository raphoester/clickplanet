package marketingv1controller_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	marketingv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/marketing/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/marketing/v1/marketingv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/marketingv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/marketingv1controller/get_subscription_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/marketingv1controller/subscribe_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/marketingv1controller/unsubscribe_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/marketingv1controller/unsubscribed_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/subscriptions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpconnect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const secret = "the-webhook-secret" //nolint:gosec // a test value, not a credential.

var now = time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)

type withdrawals struct {
	withdrawn []subscriptions.Address
}

type statusUseCase struct{}

func (statusUseCase) Execute(context.Context, subscriptions.AccountID) (subscriptions.Status, error) {
	return subscriptions.Status{State: subscriptions.StateNone, Address: "ada@example.com"}, nil
}

type subscribeUseCase struct{}

func (subscribeUseCase) Execute(context.Context, subscriptions.AccountID, string) (subscriptions.State, error) {
	return subscriptions.StateActive, nil
}

type unsubscribeUseCase struct{}

func (unsubscribeUseCase) Execute(context.Context, subscriptions.AccountID) error {
	return nil
}

func (s *withdrawals) Execute(_ context.Context, address subscriptions.Address) error {
	s.withdrawn = append(s.withdrawn, address)
	return nil
}

type stack struct {
	url       string
	signer    *cpsession.Signer
	webhook   *withdrawals
	clock     *cptime.FixedClock
	subscribe marketingv1connect.SubscriptionServiceClient
}

func serve(t *testing.T) stack {
	t.Helper()

	secretKey, public := cpsession.TestKeyPair()
	signer, err := cpsession.NewSigner(cpsession.SignerConfig{Enabled: true, Secret: secretKey, TTL: time.Hour})
	require.NoError(t, err)
	verifier, err := cpsession.NewVerifier(public)
	require.NoError(t, err)

	clock := cptime.NewFixedClock(now)
	webhook := &withdrawals{}
	limiter := cpratelimit.New("test", cpratelimit.Config{Burst: 5, PerSecond: 1.0 / 60}, clock)

	mux := http.NewServeMux()
	mux.Handle(marketingv1connect.NewSubscriptionServiceHandler(marketingv1controller.SubscriptionService{
		GetSubscriptionHandler: get_subscription_handler.New(statusUseCase{}),
		SubscribeHandler:       subscribe_handler.New(subscribeUseCase{}),
		UnsubscribeHandler:     unsubscribe_handler.New(unsubscribeUseCase{}),
	}, connect.WithInterceptors(
		marketingv1controller.NewSessionInterceptor(verifier, clock, prometheus.NewRegistry()),
		marketingv1controller.NewRateLimitInterceptor(limiter),
	)))
	mux.Handle(marketingv1connect.NewBrevoServiceHandler(marketingv1controller.BrevoService{
		UnsubscribedHandler: unsubscribed_handler.New(webhook),
	}, connect.WithInterceptors(marketingv1controller.NewWebhookInterceptor(secret))))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return stack{
		url: server.URL, signer: signer, webhook: webhook, clock: clock,
		subscribe: marketingv1connect.NewSubscriptionServiceClient(server.Client(), server.URL),
	}
}

func (s stack) token(t *testing.T, account byte) string {
	t.Helper()

	token, err := s.signer.Mint("", cpsession.Holder{Account: cpsession.AccountID{0: 1, 15: account}, Linked: true}, now)
	require.NoError(t, err)
	return token.Value
}

func request[T any](message *T, token string) *connect.Request[T] {
	req := connect.NewRequest(message)
	if token != "" {
		req.Header().Set(cpconnect.SessionHeader, token)
	}
	return req
}

func TestEveryCallNeedsASessionWithAnAccount(t *testing.T) {
	s := serve(t)

	_, err := s.subscribe.GetSubscription(t.Context(), request(&marketingv1.GetSubscriptionRequest{}, ""))
	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	_, err = s.subscribe.Subscribe(t.Context(), request(&marketingv1.SubscribeRequest{Address: "ada@example.com"}, "forged"))
	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))

	nobody, err := s.signer.Mint("", cpsession.Nobody, now)
	require.NoError(t, err)
	_, err = s.subscribe.Unsubscribe(t.Context(), request(&marketingv1.UnsubscribeRequest{}, nobody.Value))
	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err), "a token minted for nobody answers for no account")

	res, err := s.subscribe.GetSubscription(t.Context(), request(&marketingv1.GetSubscriptionRequest{}, s.token(t, 1)))
	require.NoError(t, err)
	assert.Equal(t, "ada@example.com", res.Msg.GetAddress())
	assert.Equal(t, "no-store", res.Header().Get("Cache-Control"))
}

func TestTheWritesSpendFiveThenOneAMinutePerAccount(t *testing.T) {
	s := serve(t)
	ada, bob := s.token(t, 1), s.token(t, 2)

	for range 2 {
		_, err := s.subscribe.Subscribe(t.Context(), request(&marketingv1.SubscribeRequest{Address: "ada@example.com"}, ada))
		require.NoError(t, err)
		_, err = s.subscribe.Unsubscribe(t.Context(), request(&marketingv1.UnsubscribeRequest{}, ada))
		require.NoError(t, err)
	}
	_, err := s.subscribe.Subscribe(t.Context(), request(&marketingv1.SubscribeRequest{Address: "ada@example.com"}, ada))
	require.NoError(t, err)

	_, err = s.subscribe.Subscribe(t.Context(), request(&marketingv1.SubscribeRequest{Address: "ada@example.com"}, ada))
	require.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err))

	_, err = s.subscribe.GetSubscription(t.Context(), request(&marketingv1.GetSubscriptionRequest{}, ada))
	require.NoError(t, err, "a read spends nothing")
	_, err = s.subscribe.Subscribe(t.Context(), request(&marketingv1.SubscribeRequest{Address: "bob@example.com"}, bob))
	require.NoError(t, err, "another account has a bucket of its own")

	s.clock.Advance(time.Minute)
	_, err = s.subscribe.Subscribe(t.Context(), request(&marketingv1.SubscribeRequest{Address: "ada@example.com"}, ada))
	assert.NoError(t, err)
}

func brevoPost(t *testing.T, s stack, authorization string, body string) *http.Response {
	t.Helper()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost,
		s.url+marketingv1connect.BrevoServiceUnsubscribedProcedure, strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { _ = res.Body.Close() })
	return res
}

const brevoUnsubscribe = `{"id":12345,"camp_id":7,"email":"Ada@Example.com","campaign_name":"Finale","date_sent":"2026-10-20 10:00:00",
"date_event":"2026-10-20 10:05:00","event":"unsubscribe","tag":"","sending_ip":"203.0.113.9","list_id":[3,42],
"segment_ids":[1,10],"ts_sent":1792490400,"ts_event":1792490700,"ts":1792494300}`

func TestBrevosPlainJSONPostWithTheSecretWithdrawsTheAddress(t *testing.T) {
	s := serve(t)

	res := brevoPost(t, s, "Bearer "+secret, brevoUnsubscribe) //nolint:bodyclose // closed by the helper's t.Cleanup.

	require.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, []subscriptions.Address{"ada@example.com"}, s.webhook.withdrawn)
}

func TestAWebhookCallWithoutTheSecretIsRefused(t *testing.T) {
	s := serve(t)

	for name, authorization := range map[string]string{
		"none":       "",
		"wrong":      "Bearer not-the-secret",
		"bare":       secret,
		"other kind": "Basic " + secret,
	} {
		t.Run(name, func(t *testing.T) {
			res := brevoPost(t, s, authorization, brevoUnsubscribe) //nolint:bodyclose // closed by the helper's t.Cleanup.

			assert.Equal(t, http.StatusUnauthorized, res.StatusCode)
		})
	}
	assert.Empty(t, s.webhook.withdrawn)
}

func TestAnEmptySecretRefusesEveryWebhookCall(t *testing.T) {
	ran := false
	next := connect.UnaryFunc(func(context.Context, connect.AnyRequest) (connect.AnyResponse, error) {
		ran = true
		return connect.NewResponse(&marketingv1.UnsubscribedResponse{}), nil
	})
	req := connect.NewRequest(&marketingv1.UnsubscribedRequest{})
	req.Header().Set("Authorization", "Bearer ")

	_, err := marketingv1controller.NewWebhookInterceptor("").WrapUnary(next)(t.Context(), req)

	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	assert.False(t, ran)
}
