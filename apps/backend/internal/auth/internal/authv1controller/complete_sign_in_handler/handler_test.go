package complete_sign_in_handler_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/complete_sign_in_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin/usecases/complete_sign_in_usecase"
)

type stubUseCase struct {
	out   *complete_sign_in_usecase.Out
	err   error
	asked []complete_sign_in_usecase.In
}

func (s *stubUseCase) Execute(_ context.Context, in complete_sign_in_usecase.In) (*complete_sign_in_usecase.Out, error) {
	s.asked = append(s.asked, in)
	return s.out, s.err
}

func completeSignIn(useCase *stubUseCase) (*connect.Response[authv1.CompleteSignInResponse], error) {
	req := connect.NewRequest(&authv1.CompleteSignInRequest{Code: "the-code", State: "the-state"})
	req.Header().Set("Cookie", "cp_oauth=sealed")

	res, err := complete_sign_in_handler.New(useCase, slog.New(slog.DiscardHandler)).CompleteSignIn(context.Background(), req)
	if err != nil {
		return nil, fmt.Errorf("CompleteSignIn failed: %w", err)
	}
	return res, nil
}

func TestTheAccountAndOutcomeAreAnsweredWithTheSessionCookieAndTheFlowCleared(t *testing.T) {
	useCase := &stubUseCase{out: &complete_sign_in_usecase.Out{Account: uuid.UUID{15: 1}, Outcome: accounts.Linked, SetCookie: "cp_sid=token-1"}}

	res, err := completeSignIn(useCase)
	require.NoError(t, err)

	assert.Equal(t, []complete_sign_in_usecase.In{{Code: "the-code", State: "the-state", CookieHeader: "cp_oauth=sealed"}}, useCase.asked)
	assert.Equal(t, uuid.UUID{15: 1}.String(), res.Msg.GetAccountId())
	assert.Equal(t, authv1.SignInOutcome_SIGN_IN_OUTCOME_LINKED, res.Msg.GetOutcome())
	assert.Equal(t, []string{"cp_sid=token-1", signin.ClearFlowCookie()}, res.Header().Values("Set-Cookie"))
	assert.Equal(t, "no-store", res.Header().Get("Cache-Control"))
}

func TestARefusalClearsTheFlowAndSaysNothingAboutWhy(t *testing.T) {
	for want, err := range map[connect.Code]error{
		connect.CodeFailedPrecondition: fmt.Errorf("%w: the state does not match", signin.ErrFlowInvalid),
		connect.CodePermissionDenied:   fmt.Errorf("%w: invalid_grant", signin.ErrProviderRefused),
	} {
		t.Run(want.String(), func(t *testing.T) {
			_, got := completeSignIn(&stubUseCase{err: err})

			var connectErr *connect.Error
			require.ErrorAs(t, got, &connectErr)
			assert.Equal(t, want, connectErr.Code())
			assert.Equal(t, []string{signin.ClearFlowCookie()}, connectErr.Meta().Values("Set-Cookie"))
			assert.NotContains(t, got.Error(), "state")
			assert.NotContains(t, got.Error(), "invalid_grant")
		})
	}
}

func TestSignInOffIsUnimplemented(t *testing.T) {
	_, err := completeSignIn(&stubUseCase{err: signin.ErrSignInOff})

	assert.Equal(t, connect.CodeUnimplemented, connect.CodeOf(err))
}

func TestAFailureIsLeftToTheErrorNet(t *testing.T) {
	_, err := completeSignIn(&stubUseCase{err: errors.New("postgres is down")})

	assert.Equal(t, connect.CodeUnknown, connect.CodeOf(err))
}
