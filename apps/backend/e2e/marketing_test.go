package e2e_test

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	marketingv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/marketing/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/marketing/v1/marketingv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpconnect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

type marketingStack struct {
	authStack

	fakes    auth.FakeProviders
	audience *marketing.FakeAudience
	db       *cppg.Postgres
}

func startMarketing(t *testing.T) marketingStack {
	t.Helper()

	postgres := cppg.StartTestServer(t)
	secret, public := cpsession.TestKeyPair()
	server := cpbootstrap.ServerConfig{BindAddress: freeAddress(t), InternalBindAddress: freeAddress(t)}

	authConfig := auth.Config{
		SignerConfig: cpsession.SignerConfig{Enabled: true, Secret: secret, TTL: time.Hour},
		Database:     postgres.ConfigFor("auth"),
	}
	authConfig.RateLimiter.PerSecond = 100
	authConfig.RateLimiter.Burst = 100
	authModule, fakes := auth.NewModuleWithFakeProviders(authConfig)

	marketingModule, audience := marketing.NewModuleWithFakeAudience(marketing.Config{ //nolint:gosec // a test value, not a credential.
		Enabled: true, Database: postgres.ConfigFor("marketing"), WebhookSecret: "the-webhook-secret",
	})

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() {
		done <- cpbootstrap.Run(ctx, cpbootstrap.Options{
			Server:         server,
			Logger:         slog.New(slog.DiscardHandler),
			StartupTimeout: time.Minute,
			Modules:        []cpbootstrap.Module{authModule, marketingModule},
		})
	}()
	t.Cleanup(func() {
		cancel()
		assert.NoError(t, <-done)
	})

	require.Eventually(t, func() bool {
		conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", server.BindAddress)
		if err != nil {
			return false
		}
		_ = conn.Close()
		return true
	}, time.Minute, 50*time.Millisecond, "the server never came up")

	verifier, err := cpsession.NewVerifier(public)
	require.NoError(t, err)

	db := cppg.New(postgres.ConfigFor("marketing"))
	require.NoError(t, db.ConnectCtx(t.Context()))
	t.Cleanup(func() { _ = db.Close() })

	return marketingStack{
		authStack: authStack{baseURL: "http://" + server.BindAddress, verifier: verifier},
		fakes:     fakes, audience: audience, db: db,
	}
}

func (b *browser) token() string {
	b.t.Helper()

	req := connect.NewRequest(&authv1.CreateSessionRequest{AttestationToken: "unused"})
	b.send(req.Header())
	res, err := b.client.CreateSession(b.t.Context(), req)
	require.NoError(b.t, err)
	b.keep(res.Header())
	return res.Msg.GetToken()
}

func (s marketingStack) rows(t *testing.T) int {
	t.Helper()

	var count int
	require.NoError(t, s.db.QueryRowContext(t.Context(), `SELECT count(*) FROM subscriptions`).Scan(&count))
	return count
}

func carrying[T any](message *T, token string) *connect.Request[T] {
	req := connect.NewRequest(message)
	req.Header().Set("X-Real-IP", callerIP)
	req.Header().Set(cpconnect.SessionHeader, token)
	return req
}

func TestALinkedPlayerSubscribesAndDeletingTheAccountForgetsTheContactAndTheRow(t *testing.T) {
	stack := startMarketing(t)
	emails := marketingv1connect.NewSubscriptionServiceClient(http.DefaultClient, stack.baseURL)

	guest := stack.browser(t)
	_, err := emails.Subscribe(t.Context(), carrying(&marketingv1.SubscribeRequest{Address: "guest@example.com"}, guest.token()))
	require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err), "a guest may not ask")

	ada := stack.browser(t)
	ada.mint()
	ada.signIn(authv1.Provider_PROVIDER_GOOGLE, stack.fakes.Google, "code-1",
		auth.Claim{Subject: "google-ada", Email: "ada@example.com", EmailVerified: true})
	token := ada.token()

	before, err := emails.GetSubscription(t.Context(), carrying(&marketingv1.GetSubscriptionRequest{}, token))
	require.NoError(t, err)
	assert.Equal(t, marketingv1.SubscriptionState_SUBSCRIPTION_STATE_NONE, before.Msg.GetState())
	assert.Equal(t, "ada@example.com", before.Msg.GetAddress(), "auth's verified address fills the field")

	subscribed, err := emails.Subscribe(t.Context(), carrying(&marketingv1.SubscribeRequest{Address: "ada@example.com"}, token))
	require.NoError(t, err)
	assert.Equal(t, marketingv1.SubscriptionState_SUBSCRIPTION_STATE_ACTIVE, subscribed.Msg.GetState())
	assert.Equal(t, []marketing.Call{{Method: "Join", Address: "ada@example.com"}}, stack.audience.Calls())
	require.Equal(t, 1, stack.rows(t))

	deletion := connect.NewRequest(&authv1.DeleteAccountRequest{})
	ada.send(deletion.Header())
	_, err = ada.client.DeleteAccount(t.Context(), deletion)
	require.NoError(t, err)

	require.Eventually(t, func() bool { return stack.rows(t) == 0 }, 5*time.Second, 20*time.Millisecond, "the row goes with the account")
	assert.Equal(t, []marketing.Call{
		{Method: "Join", Address: "ada@example.com"},
		{Method: "Forget", Address: "ada@example.com"},
	}, stack.audience.Calls())
}
