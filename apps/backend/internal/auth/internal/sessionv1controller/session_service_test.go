package sessionv1controller_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sessionv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/session/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/session/v1/sessionv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/usecases/create_anonymous_session_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/attestation"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/sessionv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpconnect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cphttpserver"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

type stubUseCase struct {
	token *cpsession.Token
	err   error
	asked []create_anonymous_session_usecase.In
}

func (s *stubUseCase) Execute(_ context.Context, in create_anonymous_session_usecase.In) (*cpsession.Token, error) {
	s.asked = append(s.asked, in)
	return s.token, s.err
}

type allowAll struct{}

func (allowAll) Take(string) (bool, cpratelimit.State) { return true, cpratelimit.State{} }

type refuseAll struct{}

func (refuseAll) Take(string) (bool, cpratelimit.State) { return false, cpratelimit.State{} }

var minted = &cpsession.Token{Value: "the-token", ExpiresAt: time.UnixMilli(1_800_000_000_000)}

func server(t *testing.T, useCase *stubUseCase, limiter sessionv1controller.MintLimiter) *httptest.Server {
	t.Helper()

	logger := slog.New(slog.DiscardHandler)
	mux := http.NewServeMux()
	mux.Handle(sessionv1connect.NewSessionServiceHandler(
		sessionv1controller.NewSessionService(useCase, logger),
		connect.WithInterceptors(
			cpconnect.NewErrorInterceptor(logger, nil),
			sessionv1controller.NewRateLimitInterceptor(limiter),
		),
	))

	httpServer := httptest.NewServer(cphttpserver.IPReaderMiddleware(mux))
	t.Cleanup(httpServer.Close)
	return httpServer
}

func createSession(t *testing.T, httpServer *httptest.Server) (*sessionv1.CreateSessionResponse, error) {
	t.Helper()

	req := connect.NewRequest(&sessionv1.CreateSessionRequest{AttestationToken: "widget"})
	req.Header().Set("X-Real-IP", "203.0.113.7")

	res, err := sessionv1connect.NewSessionServiceClient(httpServer.Client(), httpServer.URL).CreateSession(t.Context(), req)
	if err != nil {
		return nil, fmt.Errorf("CreateSession failed: %w", err)
	}
	return res.Msg, nil
}

func TestTheCallerReachesTheUseCaseAndGetsItsToken(t *testing.T) {
	useCase := &stubUseCase{token: minted}

	res, err := createSession(t, server(t, useCase, allowAll{}))
	require.NoError(t, err)

	assert.Equal(t, []create_anonymous_session_usecase.In{{AttestationToken: "widget", IP: "203.0.113.7"}}, useCase.asked)
	assert.Equal(t, "the-token", res.GetToken())
	assert.Equal(t, int64(1_800_000_000_000), res.GetExpiresAtUnixMs())
}

func TestARefusedAttestationAnswers403AndSaysNothingAboutWhy(t *testing.T) {
	useCase := &stubUseCase{err: fmt.Errorf("%w: invalid-input-response", attestation.ErrAttestationFailed)}

	_, err := createSession(t, server(t, useCase, allowAll{}))

	require.Error(t, err)
	assert.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
	assert.Contains(t, err.Error(), sessionv1controller.ErrRefused.Error())
	assert.NotContains(t, err.Error(), "invalid-input-response")
}

func TestMintingIsThrottled(t *testing.T) {
	_, err := createSession(t, server(t, &stubUseCase{token: minted}, refuseAll{}))

	assert.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err))
}

func TestARefusalIsA403OverHTTPAndAMintIsNeverCached(t *testing.T) {
	post := func(httpServer *httptest.Server) *http.Response {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodPost,
			httpServer.URL+sessionv1connect.SessionServiceCreateSessionProcedure,
			strings.NewReader(`{"attestationToken":"widget"}`))
		require.NoError(t, err)

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Connect-Protocol-Version", "1")
		req.Header.Set("X-Real-IP", "203.0.113.7")

		res, err := httpServer.Client().Do(req)
		require.NoError(t, err)
		t.Cleanup(func() { _ = res.Body.Close() })

		return res
	}

	//nolint:bodyclose // post() closes the body via t.Cleanup.
	ok := post(server(t, &stubUseCase{token: minted}, allowAll{}))
	assert.Equal(t, http.StatusOK, ok.StatusCode)
	assert.Equal(t, "no-store", ok.Header.Get("Cache-Control"))

	//nolint:bodyclose // post() closes the body via t.Cleanup.
	refused := post(server(t, &stubUseCase{err: attestation.ErrAttestationFailed}, allowAll{}))
	assert.Equal(t, http.StatusForbidden, refused.StatusCode)
}

func TestAnUnexpectedFailureIsInternal(t *testing.T) {
	_, err := createSession(t, server(t, &stubUseCase{err: errors.New("disk on fire")}, allowAll{}))

	assert.Equal(t, connect.CodeInternal, connect.CodeOf(err))
}
