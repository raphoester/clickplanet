package admin_start_sign_in_handler_test

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
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/adminsignin/usecases/start_admin_sign_in_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/admin_start_sign_in_handler"
)

type stubUseCase struct {
	out *start_admin_sign_in_usecase.Out
	err error
}

func (s *stubUseCase) Execute(context.Context) (*start_admin_sign_in_usecase.Out, error) {
	return s.out, s.err
}

func startSignIn(useCase *stubUseCase) (*connect.Response[authv1.AdminStartSignInResponse], error) {
	res, err := admin_start_sign_in_handler.New(useCase).StartSignIn(context.Background(), connect.NewRequest(&authv1.AdminStartSignInRequest{}))
	if err != nil {
		return nil, fmt.Errorf("StartSignIn failed: %w", err)
	}
	return res, nil
}

func TestGooglesURLIsAnsweredWithTheAdminFlowCookie(t *testing.T) {
	useCase := &stubUseCase{out: &start_admin_sign_in_usecase.Out{AuthorizationURL: "https://google.example/authorize", SetCookie: "cp_admin_oauth=sealed"}}

	res, err := startSignIn(useCase)
	require.NoError(t, err)

	assert.Equal(t, "https://google.example/authorize", res.Msg.GetAuthorizationUrl())
	assert.Equal(t, []string{"cp_admin_oauth=sealed"}, res.Header().Values("Set-Cookie"))
	assert.Equal(t, "no-store", res.Header().Get("Cache-Control"))
}

func TestEachRefusalHasItsCode(t *testing.T) {
	for want, err := range map[connect.Code]error{
		connect.CodeUnimplemented: adminsignin.ErrSignInOff,
		connect.CodeUnknown:       errors.New("no entropy"),
	} {
		t.Run(want.String(), func(t *testing.T) {
			_, got := startSignIn(&stubUseCase{err: err})

			assert.Equal(t, want, connect.CodeOf(got))
		})
	}
}
