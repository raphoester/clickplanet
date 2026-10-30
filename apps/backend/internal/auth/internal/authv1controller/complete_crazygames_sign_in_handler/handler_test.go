package complete_crazygames_sign_in_handler_test

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
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/arrivals"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/complete_crazygames_sign_in_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin/usecases/complete_crazygames_sign_in_usecase"
)

type stubUseCase struct {
	out   *complete_crazygames_sign_in_usecase.Out
	err   error
	asked []complete_crazygames_sign_in_usecase.In
}

func (s *stubUseCase) Execute(_ context.Context, in complete_crazygames_sign_in_usecase.In) (*complete_crazygames_sign_in_usecase.Out, error) {
	s.asked = append(s.asked, in)
	return s.out, s.err
}

func signIn(useCase *stubUseCase) (*connect.Response[authv1.CompleteCrazyGamesSignInResponse], error) {
	req := connect.NewRequest(&authv1.CompleteCrazyGamesSignInRequest{
		Token: "the-jwt", Landing: &authv1.Landing{Tag: "cg", Referrer: "www.crazygames.com"},
	})
	req.Header().Set("Cookie", "cp_sid=guest")

	res, err := complete_crazygames_sign_in_handler.New(useCase, slog.New(slog.DiscardHandler)).CompleteCrazyGamesSignIn(context.Background(), req)
	if err != nil {
		return nil, fmt.Errorf("CompleteCrazyGamesSignIn failed: %w", err)
	}
	return res, nil
}

func TestTheAccountAndOutcomeAreAnsweredWithTheSessionCookie(t *testing.T) {
	useCase := &stubUseCase{out: signin.AdmissionOf(accounts.AccountID{15: 1}, accounts.Linked, "cp_sid=token-1")}

	res, err := signIn(useCase)
	require.NoError(t, err)

	assert.Equal(t, []complete_crazygames_sign_in_usecase.In{{
		Token: "the-jwt", CookieHeader: "cp_sid=guest", Landing: arrivals.Landing{Tag: "cg", Referrer: "www.crazygames.com"},
	}}, useCase.asked)
	assert.Equal(t, accounts.AccountID{15: 1}.String(), res.Msg.GetAccountId())
	assert.Equal(t, authv1.SignInOutcome_SIGN_IN_OUTCOME_LINKED, res.Msg.GetOutcome())
	assert.Equal(t, []string{"cp_sid=token-1"}, res.Header().Values("Set-Cookie"))
	assert.Equal(t, "no-store", res.Header().Get("Cache-Control"))
}

func TestEachRefusalHasItsCodeAndSaysNothingAboutWhy(t *testing.T) {
	for want, err := range map[connect.Code]error{
		connect.CodeUnimplemented:    signin.ErrSignInOff,
		connect.CodePermissionDenied: fmt.Errorf("%w: crazygames: the token expired", signin.ErrProviderRefused),
		connect.CodeUnknown:          errors.New("the portal is down"),
	} {
		t.Run(want.String(), func(t *testing.T) {
			_, got := signIn(&stubUseCase{err: err})

			require.Error(t, got)
			assert.Equal(t, want, connect.CodeOf(got))
			assert.NotContains(t, got.Error(), "expired")
		})
	}
}
