package sign_out_handler_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/sign_out_handler"
)

type stubUseCase struct {
	setCookie string
	err       error
	asked     []string
}

func (s *stubUseCase) Execute(_ context.Context, cookieHeader string) (string, error) {
	s.asked = append(s.asked, cookieHeader)
	return s.setCookie, s.err
}

func signOut(useCase *stubUseCase) (*connect.Response[authv1.SignOutResponse], error) {
	req := connect.NewRequest(&authv1.SignOutRequest{})
	req.Header().Set("Cookie", "cp_sid=abc")
	res, err := sign_out_handler.New(useCase).SignOut(context.Background(), req)
	if err != nil {
		return nil, fmt.Errorf("SignOut failed: %w", err)
	}
	return res, nil
}

func TestSigningOutAnswersTheClearedCookie(t *testing.T) {
	useCase := &stubUseCase{setCookie: "cp_sid=; Max-Age=0"}

	res, err := signOut(useCase)
	require.NoError(t, err)

	assert.Equal(t, []string{"cp_sid=abc"}, useCase.asked)
	assert.Equal(t, "cp_sid=; Max-Age=0", res.Header().Get("Set-Cookie"))
	assert.Equal(t, "no-store", res.Header().Get("Cache-Control"))
}

func TestAFailureIsAnError(t *testing.T) {
	_, err := signOut(&stubUseCase{err: errors.New("postgres is down")})

	assert.Error(t, err)
}
