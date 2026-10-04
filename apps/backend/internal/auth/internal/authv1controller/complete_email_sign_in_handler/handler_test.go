package complete_email_sign_in_handler_test

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
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/complete_email_sign_in_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin/usecases/complete_email_sign_in_usecase"
)

type stubUseCase struct {
	out   *complete_email_sign_in_usecase.Out
	err   error
	asked []complete_email_sign_in_usecase.In
}

func (s *stubUseCase) Execute(_ context.Context, in complete_email_sign_in_usecase.In) (*complete_email_sign_in_usecase.Out, error) {
	s.asked = append(s.asked, in)
	return s.out, s.err
}

func completeEmailSignIn(useCase *stubUseCase) (*connect.Response[authv1.CompleteEmailSignInResponse], error) {
	req := connect.NewRequest(&authv1.CompleteEmailSignInRequest{Code: "000001"})
	req.Header().Set("Cookie", "cp_email=sealed")

	res, err := complete_email_sign_in_handler.New(useCase, slog.New(slog.DiscardHandler)).CompleteEmailSignIn(context.Background(), req)
	if err != nil {
		return nil, fmt.Errorf("CompleteEmailSignIn failed: %w", err)
	}
	return res, nil
}

func TestTheAccountAndOutcomeAreAnsweredWithTheSessionCookieAndTheChallengeCleared(t *testing.T) {
	useCase := &stubUseCase{out: signin.AdmissionOf(accounts.AccountID{15: 1}, accounts.Created, "cp_sid=token-1")}

	res, err := completeEmailSignIn(useCase)
	require.NoError(t, err)

	assert.Equal(t, []complete_email_sign_in_usecase.In{{Code: "000001", CookieHeader: "cp_email=sealed"}}, useCase.asked)
	assert.Equal(t, accounts.AccountID{15: 1}.String(), res.Msg.GetAccountId())
	assert.Equal(t, authv1.SignInOutcome_SIGN_IN_OUTCOME_CREATED, res.Msg.GetOutcome())
	assert.Equal(t, []string{"cp_sid=token-1", signin.ExpiredChallengeCookie()}, res.Header().Values("Set-Cookie"))
	assert.Equal(t, "no-store", res.Header().Get("Cache-Control"))
}

func TestAWrongCodeKeepsTheChallenge(t *testing.T) {
	_, err := completeEmailSignIn(&stubUseCase{err: fmt.Errorf("failed to check the code: %w", signin.ErrWrongCode)})

	var connectErr *connect.Error
	require.ErrorAs(t, err, &connectErr)
	assert.Equal(t, connect.CodeInvalidArgument, connectErr.Code())
	assert.Empty(t, connectErr.Meta().Values("Set-Cookie"))
}

func TestAChallengeToStartAgainIsClearedAndSaysNothingAboutWhy(t *testing.T) {
	_, err := completeEmailSignIn(&stubUseCase{err: fmt.Errorf("%w: too many codes were wrong", signin.ErrFlowInvalid)})

	var connectErr *connect.Error
	require.ErrorAs(t, err, &connectErr)
	assert.Equal(t, connect.CodeFailedPrecondition, connectErr.Code())
	assert.Equal(t, []string{signin.ExpiredChallengeCookie()}, connectErr.Meta().Values("Set-Cookie"))
	assert.NotContains(t, err.Error(), "too many")
}

func TestARefusedLinkSaysWhyInADetail(t *testing.T) {
	for want, err := range map[authv1.LinkRefusalReason]error{
		authv1.LinkRefusalReason_LINK_REFUSAL_REASON_IDENTITY_LINKED_ELSEWHERE: fmt.Errorf("failed to link email: %w", accounts.ErrIdentityLinkedElsewhere),
		authv1.LinkRefusalReason_LINK_REFUSAL_REASON_PROVIDER_ALREADY_LINKED:   fmt.Errorf("failed to link email: %w", accounts.ErrProviderAlreadyLinked),
	} {
		t.Run(want.String(), func(t *testing.T) {
			_, got := completeEmailSignIn(&stubUseCase{err: err})

			var connectErr *connect.Error
			require.ErrorAs(t, got, &connectErr)
			assert.Equal(t, connect.CodeAlreadyExists, connectErr.Code())
			assert.Equal(t, []string{signin.ExpiredChallengeCookie()}, connectErr.Meta().Values("Set-Cookie"))
			require.Len(t, connectErr.Details(), 1)
			detail, err := connectErr.Details()[0].Value()
			require.NoError(t, err)
			assert.Equal(t, want, detail.(*authv1.LinkRefusal).GetReason())
		})
	}
}

func TestEmailSignInOffIsUnimplemented(t *testing.T) {
	_, err := completeEmailSignIn(&stubUseCase{err: signin.ErrSignInOff})

	assert.Equal(t, connect.CodeUnimplemented, connect.CodeOf(err))
}

func TestAFailureIsLeftToTheErrorNet(t *testing.T) {
	_, err := completeEmailSignIn(&stubUseCase{err: errors.New("postgres is down")})

	assert.Equal(t, connect.CodeUnknown, connect.CodeOf(err))
}
