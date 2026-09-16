// Package e2e_test boots modules the way cmd/api does, on a test postgres, and checks a path over the wire.
package e2e_test

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1/authv1connect"
	sessionv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/session/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/session/v1/sessionv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

const callerIP = "203.0.113.7"

func freeAddress(t *testing.T) string {
	t.Helper()

	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	address := listener.Addr().String()
	require.NoError(t, listener.Close())

	return address
}

type authStack struct {
	baseURL  string
	verifier *cpsession.Verifier
}

func startAuth(t *testing.T) authStack {
	t.Helper()

	return startAuthModule(t, auth.NewModule)
}

// startAuthModule boots the module newModule builds from a config for the test postgres.
func startAuthModule(t *testing.T, newModule func(auth.Config) cpbootstrap.Module) authStack {
	t.Helper()

	secret, public := cpsession.TestKeyPair()
	server := cpbootstrap.ServerConfig{BindAddress: freeAddress(t)}
	config := auth.Config{
		SignerConfig: cpsession.SignerConfig{Enabled: true, Secret: secret, TTL: time.Hour},
		Database:     cppg.StartTestServer(t).ConfigFor("auth"),
	}
	config.RateLimiter.PerSecond = 100
	config.RateLimiter.Burst = 100

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() {
		done <- cpbootstrap.Run(ctx, cpbootstrap.Options{
			Server:         server,
			Logger:         slog.New(slog.DiscardHandler),
			StartupTimeout: time.Minute,
			Modules:        []cpbootstrap.Module{newModule(config)},
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

	return authStack{baseURL: "http://" + server.BindAddress, verifier: verifier}
}

func (s authStack) accountIn(t *testing.T, token string) uuid.UUID {
	t.Helper()

	claims, err := s.verifier.Verify(token, callerIP, time.Now())
	require.NoError(t, err)
	return claims.Account
}

func (s authStack) createSession(t *testing.T, cookie string) (uuid.UUID, *http.Cookie) {
	t.Helper()

	req := connect.NewRequest(&authv1.CreateSessionRequest{AttestationToken: "unused"})
	req.Header().Set("X-Real-IP", callerIP)
	if cookie != "" {
		req.Header().Set("Cookie", cookie)
	}

	res, err := authv1connect.NewAuthServiceClient(http.DefaultClient, s.baseURL).CreateSession(t.Context(), req)
	require.NoError(t, err)

	var setCookie *http.Cookie
	if header := res.Header().Get("Set-Cookie"); header != "" {
		setCookie, err = http.ParseSetCookie(header)
		require.NoError(t, err)
	}

	return s.accountIn(t, res.Msg.GetToken()), setCookie
}

func TestANewPlayerGetsAGuestAndItsCookieBringsItBack(t *testing.T) {
	stack := startAuth(t)

	guest, cookie := stack.createSession(t, "")
	require.NotEqual(t, uuid.Nil, guest)
	require.NotNil(t, cookie)
	assert.True(t, cookie.HttpOnly)

	again, renewed := stack.createSession(t, cookie.Name+"="+cookie.Value)
	assert.Equal(t, guest, again, "the cookie brings the same account back, and no second guest is made")
	assert.Nil(t, renewed, "not extended within the day")

	getMe := connect.NewRequest(&authv1.GetMeRequest{})
	getMe.Header().Set("Cookie", cookie.Name+"="+cookie.Value)
	me, err := authv1connect.NewAuthServiceClient(http.DefaultClient, stack.baseURL).GetMe(t.Context(), getMe)
	require.NoError(t, err)
	assert.Equal(t, guest.String(), me.Msg.GetAccountId())
}

func TestTheDeprecatedPathStillMintsATokenWithNoAccount(t *testing.T) {
	stack := startAuth(t)

	req := connect.NewRequest(&sessionv1.CreateSessionRequest{AttestationToken: "unused"})
	req.Header().Set("X-Real-IP", callerIP)
	res, err := sessionv1connect.NewSessionServiceClient(http.DefaultClient, stack.baseURL).CreateSession(t.Context(), req)
	require.NoError(t, err)

	assert.Equal(t, uuid.Nil, stack.accountIn(t, res.Msg.GetToken()))
	assert.Empty(t, res.Header().Get("Set-Cookie"))
}
