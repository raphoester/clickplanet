package start_sign_in_handler_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/start_sign_in_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin/usecases/start_sign_in_usecase"
)

type stubUseCase struct {
	out   *start_sign_in_usecase.Out
	err   error
	asked []string
}

func (s *stubUseCase) Execute(_ context.Context, provider string) (*start_sign_in_usecase.Out, error) {
	s.asked = append(s.asked, provider)
	return s.out, s.err
}

func startSignIn(useCase *stubUseCase, provider authv1.Provider) (*connect.Response[authv1.StartSignInResponse], error) {
	res, err := start_sign_in_handler.New(useCase).StartSignIn(context.Background(), connect.NewRequest(&authv1.StartSignInRequest{Provider: provider}))
	if err != nil {
		return nil, fmt.Errorf("StartSignIn failed: %w", err)
	}
	return res, nil
}

func TestTheProviderURLIsAnsweredWithTheFlowCookie(t *testing.T) {
	useCase := &stubUseCase{out: &start_sign_in_usecase.Out{AuthorizationURL: "https://discord.example/authorize", SetCookie: "cp_oauth=sealed"}}

	res, err := startSignIn(useCase, authv1.Provider_PROVIDER_DISCORD)
	require.NoError(t, err)

	assert.Equal(t, []string{"discord"}, useCase.asked)
	assert.Equal(t, "https://discord.example/authorize", res.Msg.GetAuthorizationUrl())
	assert.Equal(t, "cp_oauth=sealed", res.Header().Get("Set-Cookie"))
	assert.Equal(t, "no-store", res.Header().Get("Cache-Control"))
}

func TestEachRefusalHasItsCode(t *testing.T) {
	for want, err := range map[connect.Code]error{
		connect.CodeUnimplemented:   signin.ErrSignInOff,
		connect.CodeInvalidArgument: fmt.Errorf("%w: %q", signin.ErrUnknownProvider, ""),
		connect.CodeUnknown:         errors.New("no entropy"),
	} {
		t.Run(want.String(), func(t *testing.T) {
			_, got := startSignIn(&stubUseCase{err: err}, authv1.Provider_PROVIDER_UNSPECIFIED)

			assert.Equal(t, want, connect.CodeOf(got))
		})
	}
}
