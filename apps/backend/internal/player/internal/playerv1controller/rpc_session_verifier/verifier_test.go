package rpc_session_verifier_test

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1/authv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/rpc_session_verifier"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

const ip = "203.0.113.7"

var now = time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)

// stubAuth answers what the auth module answers, and counts how often it is asked.
type stubAuth struct {
	mu    sync.Mutex
	calls int
	key   string
	err   error
}

func (s *stubAuth) GetVerifyingKey(
	_ context.Context,
	_ *connect.Request[authv1.GetVerifyingKeyRequest],
) (*connect.Response[authv1.GetVerifyingKeyResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.calls++
	if s.err != nil {
		return nil, s.err
	}

	return connect.NewResponse(&authv1.GetVerifyingKeyResponse{PublicKey: s.key}), nil
}

func (s *stubAuth) asked() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.calls
}

func (s *stubAuth) answerWith(key string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.key, s.err = key, err
}

type dialer struct {
	client connect.HTTPClient
	url    string
	err    error
}

func (d dialer) Dial() (connect.HTTPClient, string, error) {
	return d.client, d.url, d.err
}

func setUp(t *testing.T) (*rpc_session_verifier.Verifier, *stubAuth, *cpsession.Signer) {
	t.Helper()

	secret, public := cpsession.TestKeyPair()
	auth := &stubAuth{key: public}

	mux := http.NewServeMux()
	mux.Handle(authv1connect.NewInternalServiceHandler(auth))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	signer, err := cpsession.NewSigner(cpsession.SignerConfig{Enabled: true, Secret: secret, TTL: time.Hour})
	require.NoError(t, err)

	return rpc_session_verifier.New(dialer{client: server.Client(), url: server.URL}, slog.New(slog.DiscardHandler)),
		auth, signer
}

func TestItAsksAuthOnceAndKeepsTheKey(t *testing.T) {
	verifier, auth, signer := setUp(t)

	token, err := signer.Mint(ip, cpsession.NoAccount, now)
	require.NoError(t, err)

	for range 5 {
		_, err := verifier.Verify(token.Value, ip, now)
		require.NoError(t, err)
	}

	assert.Equal(t, 1, auth.asked(), "the key does not change while the process runs")
}

func TestConcurrentFirstClicksAskOnce(t *testing.T) {
	verifier, auth, signer := setUp(t)

	token, err := signer.Mint(ip, cpsession.NoAccount, now)
	require.NoError(t, err)

	var wg sync.WaitGroup
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := verifier.Verify(token.Value, ip, now)
			assert.NoError(t, err)
		}()
	}
	wg.Wait()

	assert.Equal(t, 1, auth.asked(), "a cold start is one fetch, not one per caller")
}

func TestAFailedFetchIsNotRememberedAndTheNextClickTriesAgain(t *testing.T) {
	verifier, auth, signer := setUp(t)
	secret, public := cpsession.TestKeyPair()
	require.NotEmpty(t, secret)

	auth.answerWith("", errors.New("auth is not up yet"))

	token, err := signer.Mint(ip, cpsession.NoAccount, now)
	require.NoError(t, err)

	_, err = verifier.Verify(token.Value, ip, now)
	require.Error(t, err)

	auth.answerWith(public, nil)

	claims, err := verifier.Verify(token.Value, ip, now)
	require.NoError(t, err)
	assert.Equal(t, token.ID, claims.ID)
	assert.Equal(t, 2, auth.asked())
}

func TestAKeyThisServerCannotUseIsAnError(t *testing.T) {
	verifier, auth, signer := setUp(t)
	auth.answerWith("not-a-key", nil)

	token, err := signer.Mint(ip, cpsession.NoAccount, now)
	require.NoError(t, err)

	_, err = verifier.Verify(token.Value, ip, now)
	assert.ErrorContains(t, err, "verifying key")
}

func TestAnUnreachableAuthIsAnErrorRatherThanAPanic(t *testing.T) {
	verifier := rpc_session_verifier.New(
		dialer{err: errors.New("no internal listener")}, slog.New(slog.DiscardHandler))

	_, err := verifier.Verify("whatever", ip, now)
	assert.ErrorContains(t, err, "failed to reach the auth module")
}
