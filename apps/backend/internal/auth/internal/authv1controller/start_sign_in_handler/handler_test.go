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
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/start_sign_in_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin/usecases/start_sign_in_usecase"
)

type stubUseCase struct {
	out   *start_sign_in_usecase.Out
	err   error
	asked []start_sign_in_usecase.In
}

func (s *stubUseCase) Execute(_ context.Context, in start_sign_in_usecase.In) (*start_sign_in_usecase.Out, error) {
	s.asked = append(s.asked, in)
	return s.out, s.err
}

func startSignIn(useCase *stubUseCase, msg *authv1.StartSignInRequest) (*connect.Response[authv1.StartSignInResponse], error) {
	req := connect.NewRequest(msg)
	req.Header().Set("Cookie", "cp_sid=guest-token")
	res, err := start_sign_in_handler.New(useCase).StartSignIn(context.Background(), req)
	if err != nil {
		return nil, fmt.Errorf("StartSignIn failed: %w", err)
	}
	return res, nil
}

func TestTheProviderURLIsAnsweredWithTheFlowCookie(t *testing.T) {
	useCase := &stubUseCase{out: &start_sign_in_usecase.Out{AuthorizationURL: "https://discord.example/authorize", SetCookie: "cp_oauth=sealed"}}

	res, err := startSignIn(useCase, &authv1.StartSignInRequest{Provider: authv1.Provider_PROVIDER_DISCORD})
	require.NoError(t, err)

	assert.Equal(t, []start_sign_in_usecase.In{{Provider: "discord", CookieHeader: "cp_sid=guest-token"}}, useCase.asked)
	assert.Equal(t, "https://discord.example/authorize", res.Msg.GetAuthorizationUrl())
	assert.Equal(t, "cp_oauth=sealed", res.Header().Get("Set-Cookie"))
	assert.Equal(t, "no-store", res.Header().Get("Cache-Control"))
}

func TestOnlyALinkIsReadAsALink(t *testing.T) {
	for wire, want := range map[authv1.SignInIntent]accounts.Intent{
		authv1.SignInIntent_SIGN_IN_INTENT_UNSPECIFIED: accounts.IntentSignIn,
		authv1.SignInIntent_SIGN_IN_INTENT_SIGN_IN:     accounts.IntentSignIn,
		authv1.SignInIntent_SIGN_IN_INTENT_LINK:        accounts.IntentLink,
	} {
		t.Run(wire.String(), func(t *testing.T) {
			useCase := &stubUseCase{out: &start_sign_in_usecase.Out{}}

			_, err := startSignIn(useCase, &authv1.StartSignInRequest{Provider: authv1.Provider_PROVIDER_GOOGLE, Intent: wire})
			require.NoError(t, err)

			require.Len(t, useCase.asked, 1)
			assert.Equal(t, want, useCase.asked[0].Intent)
		})
	}
}

func TestEachRefusalHasItsCode(t *testing.T) {
	for want, err := range map[connect.Code]error{
		connect.CodeUnimplemented:   signin.ErrSignInOff,
		connect.CodeInvalidArgument: fmt.Errorf("%w: %q", signin.ErrUnknownProvider, ""),
		connect.CodeUnauthenticated: fmt.Errorf("failed to find the account to link to: %w", accounts.ErrNoAccount),
		connect.CodeUnknown:         errors.New("no entropy"),
	} {
		t.Run(want.String(), func(t *testing.T) {
			_, got := startSignIn(&stubUseCase{err: err}, &authv1.StartSignInRequest{})

			assert.Equal(t, want, connect.CodeOf(got))
		})
	}
}
