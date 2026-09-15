package sessionv1controller_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sessionv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/session/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/session/v1/sessionv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/session/internal/adapters/primary/http/sessionv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/session/internal/adapters/secondary/open_attester"
	"github.com/raphoester/clickplanet.lol-backend/internal/session/internal/domain"
	"github.com/raphoester/clickplanet.lol-backend/internal/session/internal/domain/session_service"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpconnect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cphttpserver"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

type refusingAttester struct{}

func (refusingAttester) Attest(context.Context, string, string) error {
	return errors.New("siteverify said no: invalid-input-response")
}

type allowAll struct{}

func (allowAll) Take(string) (bool, cpratelimit.State) { return true, cpratelimit.State{} }

type refuseAll struct{}

func (refuseAll) Take(string) (bool, cpratelimit.State) { return false, cpratelimit.State{} }

func sessionServer(
	t *testing.T,
	attester domain.Attester,
	signer *cpsession.Signer,
	limiter sessionv1controller.MintLimiter,
) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	mux.Handle(sessionv1connect.NewSessionServiceHandler(
		sessionv1controller.NewSessionService(
			session_service.New(attester, signer, nil),
			nil,
		),
		connect.WithInterceptors(
			// What cpbootstrap wraps every mounted service in.
			cpconnect.NewErrorInterceptor(nil, nil),
			sessionv1controller.NewRateLimitInterceptor(limiter),
		),
	))

	server := httptest.NewServer(cphttpserver.IPReaderMiddleware(mux))
	t.Cleanup(server.Close)

	return server
}

func newSigner(t *testing.T) *cpsession.Signer {
	t.Helper()
	signer, err := cpsession.NewSigner(cpsession.Config{Secret: "a-test-secret", TTL: time.Hour})
	require.NoError(t, err)
	return signer
}

func createSession(server *httptest.Server, ip string, token string) (*sessionv1.CreateSessionResponse, error) {
	req := connect.NewRequest(&sessionv1.CreateSessionRequest{AttestationToken: token})
	req.Header().Set("X-Real-IP", ip)

	res, err := sessionv1connect.NewSessionServiceClient(server.Client(), server.URL).
		CreateSession(context.Background(), req)
	if err != nil {
		return nil, err
	}

	return res.Msg, nil
}

func TestAMintedTokenIsUsableByTheCallerThatMintedIt(t *testing.T) {
	signer := newSigner(t)
	server := sessionServer(t, open_attester.New(), signer, allowAll{})

	res, err := createSession(server, "203.0.113.7", "a-widget-token")
	require.NoError(t, err)

	require.NotEmpty(t, res.GetToken())
	assert.InDelta(t, time.Now().Add(time.Hour).UnixMilli(), res.GetExpiresAtUnixMs(), float64(time.Minute.Milliseconds()))

	_, err = signer.Verify(res.GetToken(), "203.0.113.7", time.Now())
	require.NoError(t, err)

	_, err = signer.Verify(res.GetToken(), "198.51.100.4", time.Now())
	assert.Error(t, err, "the token does not travel to another address")
}

func TestARefusedAttestationAnswers403AndSaysNothingAboutWhy(t *testing.T) {
	server := sessionServer(t, refusingAttester{}, newSigner(t), allowAll{})

	_, err := createSession(server, "203.0.113.7", "a-widget-token")

	require.Error(t, err)
	assert.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
	assert.Contains(t, err.Error(), sessionv1controller.ErrRefused.Error())
	assert.NotContains(t, err.Error(), "invalid-input-response",
		"the caller learns that it was refused, not which check to work on next")
}

func TestMintingIsThrottled(t *testing.T) {
	server := sessionServer(t, open_attester.New(), newSigner(t), refuseAll{})

	_, err := createSession(server, "203.0.113.7", "a-widget-token")

	require.Error(t, err)
	assert.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err))
}

func TestARefusalIsA403OverHTTPAndAMintIsNeverCached(t *testing.T) {
	signer := newSigner(t)

	post := func(server *httptest.Server) *http.Response {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodPost,
			server.URL+sessionv1connect.SessionServiceCreateSessionProcedure,
			strings.NewReader(`{"attestationToken":"a-widget-token"}`))
		require.NoError(t, err)

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Connect-Protocol-Version", "1")
		req.Header.Set("X-Real-IP", "203.0.113.7")

		res, err := server.Client().Do(req)
		require.NoError(t, err)
		t.Cleanup(func() { _ = res.Body.Close() })

		return res
	}

	//nolint:bodyclose // post() closes the body via t.Cleanup.
	minted := post(sessionServer(t, open_attester.New(), signer, allowAll{}))
	assert.Equal(t, http.StatusOK, minted.StatusCode)
	assert.Equal(t, "no-store", minted.Header.Get("Cache-Control"))

	//nolint:bodyclose // post() closes the body via t.Cleanup.
	refused := post(sessionServer(t, refusingAttester{}, signer, allowAll{}))
	assert.Equal(t, http.StatusForbidden, refused.StatusCode)
}
