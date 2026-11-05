package admin_sign_out_handler_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/adminsignin"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/admin_sign_out_handler"
)

type stubUseCase struct {
	err   error
	asked []string
}

func (s *stubUseCase) Execute(_ context.Context, cookieHeader string) error {
	s.asked = append(s.asked, cookieHeader)
	return s.err
}

func signOut(useCase *stubUseCase) (*connect.Response[authv1.AdminSignOutResponse], error) {
	req := connect.NewRequest(&authv1.AdminSignOutRequest{})
	req.Header().Set("Cookie", "cp_sid=game-token; cp_admin=admin-token")
	res, err := admin_sign_out_handler.New(useCase).SignOut(context.Background(), req)
	if err != nil {
		return nil, fmt.Errorf("SignOut failed: %w", err)
	}
	return res, nil
}

func TestSigningOutClearsTheAdminCookieAlone(t *testing.T) {
	useCase := &stubUseCase{}

	res, err := signOut(useCase)
	require.NoError(t, err)

	assert.Equal(t, []string{"cp_sid=game-token; cp_admin=admin-token"}, useCase.asked)
	assert.Equal(t, []string{adminsignin.ExpiredCookie()}, res.Header().Values("Set-Cookie"), "cp_sid is not touched")
	assert.Equal(t, "no-store", res.Header().Get("Cache-Control"))
}

func TestAFailureIsAnError(t *testing.T) {
	_, err := signOut(&stubUseCase{err: errors.New("postgres is down")})

	assert.Equal(t, connect.CodeUnknown, connect.CodeOf(err))
}
