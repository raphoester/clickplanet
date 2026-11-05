package admin_complete_sign_in_handler_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/adminsignin"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/adminsignin/usecases/complete_admin_sign_in_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/admin_complete_sign_in_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin"
)

type stubUseCase struct {
	out   *complete_admin_sign_in_usecase.Out
	err   error
	asked []complete_admin_sign_in_usecase.In
}

func (s *stubUseCase) Execute(_ context.Context, in complete_admin_sign_in_usecase.In) (*complete_admin_sign_in_usecase.Out, error) {
	s.asked = append(s.asked, in)
	return s.out, s.err
}

func completeSignIn(useCase *stubUseCase) (*connect.Response[authv1.AdminCompleteSignInResponse], error) {
	req := connect.NewRequest(&authv1.AdminCompleteSignInRequest{Code: "the-code", State: "the-state"})
	req.Header().Set("Cookie", "cp_admin_oauth=sealed")

	res, err := admin_complete_sign_in_handler.New(useCase, slog.New(slog.DiscardHandler)).CompleteSignIn(context.Background(), req)
	if err != nil {
		return nil, fmt.Errorf("CompleteSignIn failed: %w", err)
	}
	return res, nil
}

func TestTheOperatorIsAnsweredWithTheAdminCookieAndTheFlowCleared(t *testing.T) {
	useCase := &stubUseCase{out: &complete_admin_sign_in_usecase.Out{Account: accounts.AccountID{15: 1}, SetCookie: "cp_admin=token-1"}}

	res, err := completeSignIn(useCase)
	require.NoError(t, err)

	assert.Equal(t, []complete_admin_sign_in_usecase.In{{Code: "the-code", State: "the-state", CookieHeader: "cp_admin_oauth=sealed"}}, useCase.asked)
	assert.Equal(t, accounts.AccountID{15: 1}.String(), res.Msg.GetAccountId())
	assert.Equal(t, []string{"cp_admin=token-1", adminsignin.ExpiredFlowCookie()}, res.Header().Values("Set-Cookie"))
	assert.Equal(t, "no-store", res.Header().Get("Cache-Control"))
}

func TestARefusalClearsTheFlowOpensNoSessionAndSaysNothingAboutWhy(t *testing.T) {
	for name, tc := range map[string]struct {
		err  error
		want connect.Code
	}{
		"flow invalid":     {fmt.Errorf("%w: the-reason", signin.ErrFlowInvalid), connect.CodeFailedPrecondition},
		"provider refused": {fmt.Errorf("%w: the-reason", signin.ErrProviderRefused), connect.CodePermissionDenied},
		"not an operator":  {fmt.Errorf("%w: the-reason", adminsignin.ErrNotAnOperator), connect.CodePermissionDenied},
	} {
		t.Run(name, func(t *testing.T) {
			_, got := completeSignIn(&stubUseCase{err: tc.err})

			var connectErr *connect.Error
			require.ErrorAs(t, got, &connectErr)
			assert.Equal(t, tc.want, connectErr.Code())
			assert.Equal(t, []string{adminsignin.ExpiredFlowCookie()}, connectErr.Meta().Values("Set-Cookie"), "no cp_admin")
			assert.NotContains(t, got.Error(), "the-reason")
		})
	}
}

func TestAdminSignInOffIsUnimplemented(t *testing.T) {
	_, err := completeSignIn(&stubUseCase{err: adminsignin.ErrSignInOff})

	assert.Equal(t, connect.CodeUnimplemented, connect.CodeOf(err))
}

func TestAFailureIsLeftToTheErrorNet(t *testing.T) {
	_, err := completeSignIn(&stubUseCase{err: errors.New("postgres is down")})

	assert.Equal(t, connect.CodeUnknown, connect.CodeOf(err))
}
