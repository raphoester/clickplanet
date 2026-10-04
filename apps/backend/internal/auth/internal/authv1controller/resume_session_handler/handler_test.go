package resume_session_handler_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/usecases/resume_session_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/resume_session_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

type stubUseCase struct {
	out   *resume_session_usecase.Out
	err   error
	asked []resume_session_usecase.In
}

func (s *stubUseCase) Execute(_ context.Context, in resume_session_usecase.In) (*resume_session_usecase.Out, error) {
	s.asked = append(s.asked, in)
	return s.out, s.err
}

func resume(useCase *stubUseCase) (*connect.Response[authv1.ResumeSessionResponse], error) {
	req := connect.NewRequest(&authv1.ResumeSessionRequest{})
	req.Header().Set("Cookie", "cp_sid=token-1")
	ctx := cpctx.AddIPToContext(context.Background(), "203.0.113.7")
	return resume_session_handler.New(useCase).ResumeSession(ctx, req) //nolint:wrapcheck // a test helper.
}

func TestTheTokenItsCookieAndTheCallersAddressAreMapped(t *testing.T) {
	expiresAt := time.Date(2026, 10, 4, 13, 0, 0, 0, time.UTC)
	useCase := &stubUseCase{out: &resume_session_usecase.Out{
		Token:     &cpsession.Token{Value: "identity", ExpiresAt: expiresAt},
		SetCookie: "cp_sid=token-1; Path=/",
	}}

	res, err := resume(useCase)

	require.NoError(t, err)
	assert.Equal(t, "identity", res.Msg.GetToken())
	assert.Equal(t, expiresAt.UnixMilli(), res.Msg.GetExpiresAtUnixMs())
	assert.Equal(t, "cp_sid=token-1; Path=/", res.Header().Get("Set-Cookie"))
	assert.Equal(t, "no-store", res.Header().Get("Cache-Control"))
	assert.Equal(t, []resume_session_usecase.In{{IP: "203.0.113.7", CookieHeader: "cp_sid=token-1"}}, useCase.asked)
}

func TestNoLiveSessionIsAnEmptyAnswerAndNotAnError(t *testing.T) {
	res, err := resume(&stubUseCase{out: &resume_session_usecase.Out{}})

	require.NoError(t, err)
	assert.Empty(t, res.Msg.GetToken())
	assert.Empty(t, res.Header().Get("Set-Cookie"))
}

func TestNoAddressIsARefusal(t *testing.T) {
	_, err := resume(&stubUseCase{err: resume_session_usecase.ErrNoAddress})

	assert.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
}

func TestAStoreFailureIsLeftToTheErrorNet(t *testing.T) {
	_, err := resume(&stubUseCase{err: errors.New("postgres is down")})

	assert.ErrorContains(t, err, "postgres is down")
}
