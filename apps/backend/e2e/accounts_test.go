// Package e2e_test boots several modules together, the way cmd/api does, and checks a path across them.
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
	"github.com/raphoester/clickplanet.lol-backend/internal/session"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

func freeAddress(t *testing.T) string {
	t.Helper()

	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	address := listener.Addr().String()
	require.NoError(t, listener.Close())

	return address
}

type accountsStack struct {
	address string
	signer  *cpsession.Signer
}

func startAccountsStack(t *testing.T) accountsStack {
	t.Helper()

	server := cpbootstrap.ServerConfig{BindAddress: freeAddress(t), InternalBindAddress: freeAddress(t)}
	authConfig := auth.Config{Enabled: true, Database: cppg.StartTestServer(t).ConfigFor("auth")}
	sessionConfig := session.Config{Config: cpsession.Config{Enabled: true, Secret: "a-test-secret", TTL: time.Hour}}
	sessionConfig.RateLimiter.PerSecond = 100
	sessionConfig.RateLimiter.Burst = 100
	sessionConfig.Accounts.Enabled = true

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() {
		done <- cpbootstrap.Run(ctx, cpbootstrap.Options{
			Server:         server,
			Logger:         slog.New(slog.DiscardHandler),
			StartupTimeout: time.Minute,
			Modules:        []cpbootstrap.Module{auth.NewModule(authConfig), session.NewModule(sessionConfig)},
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

	signer, err := cpsession.NewSigner(sessionConfig.Config)
	require.NoError(t, err)

	return accountsStack{address: "http://" + server.BindAddress, signer: signer}
}

func (s accountsStack) mint(t *testing.T, cookie string, createAccount bool) (uuid.UUID, *http.Cookie) {
	t.Helper()

	req := connect.NewRequest(&sessionv1.CreateSessionRequest{AttestationToken: "unused", CreateAccount: createAccount})
	req.Header().Set("X-Real-IP", "203.0.113.7")
	if cookie != "" {
		req.Header().Set("Cookie", cookie)
	}

	res, err := sessionv1connect.NewSessionServiceClient(http.DefaultClient, s.address).CreateSession(t.Context(), req)
	require.NoError(t, err)

	claims, err := s.signer.Verify(res.Msg.GetToken(), "203.0.113.7", time.Now())
	require.NoError(t, err)

	var setCookie *http.Cookie
	if header := res.Header().Get("Set-Cookie"); header != "" {
		setCookie, err = http.ParseSetCookie(header)
		require.NoError(t, err)
	}

	return claims.Account, setCookie
}

func TestAMintGivesAGuestAccountAndItsCookieBringsItBack(t *testing.T) {
	stack := startAccountsStack(t)

	none, noCookie := stack.mint(t, "", false)
	assert.Equal(t, uuid.Nil, none, "a mint that asks for nothing is not given an account")
	assert.Nil(t, noCookie)

	guest, cookie := stack.mint(t, "", true)
	require.NotEqual(t, uuid.Nil, guest)
	require.NotNil(t, cookie)
	assert.True(t, cookie.HttpOnly)

	again, renewed := stack.mint(t, cookie.Name+"="+cookie.Value, true)
	assert.Equal(t, guest, again, "the cookie brings the same account back, and no second guest is made")
	assert.Nil(t, renewed, "not extended within the day")

	getMe := connect.NewRequest(&authv1.GetMeRequest{})
	getMe.Header().Set("Cookie", cookie.Name+"="+cookie.Value)
	me, err := authv1connect.NewAuthServiceClient(http.DefaultClient, stack.address).GetMe(t.Context(), getMe)
	require.NoError(t, err)
	assert.Equal(t, guest.String(), me.Msg.GetAccountId())
}
