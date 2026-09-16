package create_session_handler_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1/authv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/usecases/create_session_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/attestation"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/create_session_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpconnect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cphttpserver"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

type stubUseCase struct {
	out   *create_session_usecase.Out
	err   error
	asked []create_session_usecase.In
}

func (s *stubUseCase) Execute(_ context.Context, in create_session_usecase.In) (*create_session_usecase.Out, error) {
	s.asked = append(s.asked, in)
	return s.out, s.err
}

// onlyCreateSession serves CreateSession, and Unimplemented for every other procedure.
type onlyCreateSession struct {
	unimplemented
	create_session_handler.CreateSessionHandler
}

type unimplemented struct {
	authv1connect.UnimplementedAuthServiceHandler
}

type allowAll struct{}

func (allowAll) Take(string) (bool, cpratelimit.State) { return true, cpratelimit.State{} }

type refuseAll struct{}

func (refuseAll) Take(string) (bool, cpratelimit.State) { return false, cpratelimit.State{} }

var minted = &create_session_usecase.Out{
	Token:     &cpsession.Token{Value: "the-token", ExpiresAt: time.UnixMilli(1_800_000_000_000)},
	Account:   uuid.UUID{15: 1},
	SetCookie: "cp_sid=token-1; Path=/; HttpOnly",
}

func server(t *testing.T, useCase *stubUseCase, limiter authv1controller.MintLimiter) *httptest.Server {
	t.Helper()

	logger := slog.New(slog.DiscardHandler)
	service := onlyCreateSession{CreateSessionHandler: create_session_handler.New(useCase, logger)}

	mux := http.NewServeMux()
	mux.Handle(authv1connect.NewAuthServiceHandler(service, connect.WithInterceptors(
		cpconnect.NewErrorInterceptor(logger, nil),
		authv1controller.NewRateLimitInterceptor(limiter),
	)))

	httpServer := httptest.NewServer(cphttpserver.IPReaderMiddleware(mux))
	t.Cleanup(httpServer.Close)
	return httpServer
}

func createSession(t *testing.T, httpServer *httptest.Server) (*connect.Response[authv1.CreateSessionResponse], error) {
	t.Helper()

	req := connect.NewRequest(&authv1.CreateSessionRequest{AttestationToken: "widget"})
	req.Header().Set("X-Real-IP", "203.0.113.7")
	req.Header().Set("Cookie", "theme=dark")

	res, err := authv1connect.NewAuthServiceClient(httpServer.Client(), httpServer.URL).CreateSession(t.Context(), req)
	if err != nil {
		return nil, fmt.Errorf("CreateSession failed: %w", err)
	}
	return res, nil
}

func TestTheCallerReachesTheUseCaseAndGetsItsTokenAndCookie(t *testing.T) {
	useCase := &stubUseCase{out: minted}

	res, err := createSession(t, server(t, useCase, allowAll{}))
	require.NoError(t, err)

	assert.Equal(t, []create_session_usecase.In{{AttestationToken: "widget", IP: "203.0.113.7", CookieHeader: "theme=dark"}}, useCase.asked)
	assert.Equal(t, "the-token", res.Msg.GetToken())
	assert.Equal(t, int64(1_800_000_000_000), res.Msg.GetExpiresAtUnixMs())
	assert.Equal(t, "cp_sid=token-1; Path=/; HttpOnly", res.Header().Get("Set-Cookie"))
	assert.Equal(t, "no-store", res.Header().Get("Cache-Control"))
}

func TestARefusedAttestationIsPermissionDeniedAndSaysNothingAboutWhy(t *testing.T) {
	useCase := &stubUseCase{err: fmt.Errorf("%w: invalid-input-response", attestation.ErrAttestationFailed)}

	_, err := createSession(t, server(t, useCase, allowAll{}))

	require.Error(t, err)
	assert.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
	assert.NotContains(t, err.Error(), "invalid-input-response")
}

func TestAStoreFailureIsInternalAndSaysNothingAboutWhy(t *testing.T) {
	useCase := &stubUseCase{err: errors.New("postgres is down")}

	_, err := createSession(t, server(t, useCase, allowAll{}))

	require.Error(t, err)
	assert.Equal(t, connect.CodeInternal, connect.CodeOf(err))
	assert.NotContains(t, err.Error(), "postgres")
}

func TestMintingIsThrottled(t *testing.T) {
	useCase := &stubUseCase{out: minted}

	_, err := createSession(t, server(t, useCase, refuseAll{}))

	assert.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err))
	assert.Empty(t, useCase.asked)
}
