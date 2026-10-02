package start_email_sign_in_handler_test

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
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/attestation"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/start_email_sign_in_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin/usecases/start_email_sign_in_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

type stubUseCase struct {
	out   *start_email_sign_in_usecase.Out
	err   error
	asked []start_email_sign_in_usecase.In
}

func (s *stubUseCase) Execute(_ context.Context, in start_email_sign_in_usecase.In) (*start_email_sign_in_usecase.Out, error) {
	s.asked = append(s.asked, in)
	return s.out, s.err
}

func startEmailSignIn(useCase *stubUseCase, intent authv1.SignInIntent) (*connect.Response[authv1.StartEmailSignInResponse], error) {
	req := connect.NewRequest(&authv1.StartEmailSignInRequest{Email: "player@example.com", Intent: intent, AttestationToken: "widget"})
	req.Header().Set("Cookie", "cp_sid=guest")
	ctx := cpctx.AddIPToContext(context.Background(), "192.0.2.1")

	res, err := start_email_sign_in_handler.New(useCase, slog.New(slog.DiscardHandler)).StartEmailSignIn(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("StartEmailSignIn failed: %w", err)
	}
	return res, nil
}

func TestTheRequestReachesTheUseCaseAndTheChallengeCookieIsSet(t *testing.T) {
	useCase := &stubUseCase{out: &start_email_sign_in_usecase.Out{SetCookie: "cp_email=sealed"}}

	res, err := startEmailSignIn(useCase, authv1.SignInIntent_SIGN_IN_INTENT_LINK)
	require.NoError(t, err)

	assert.Equal(t, []start_email_sign_in_usecase.In{{
		Address: "player@example.com", Intent: accounts.IntentLink, AttestationToken: "widget", IP: "192.0.2.1", CookieHeader: "cp_sid=guest",
	}}, useCase.asked)
	assert.Equal(t, []string{"cp_email=sealed"}, res.Header().Values("Set-Cookie"))
	assert.Equal(t, "no-store", res.Header().Get("Cache-Control"))
}

func TestAnUnsetIntentSignsIn(t *testing.T) {
	useCase := &stubUseCase{out: &start_email_sign_in_usecase.Out{SetCookie: "cp_email=sealed"}}

	_, err := startEmailSignIn(useCase, authv1.SignInIntent_SIGN_IN_INTENT_UNSPECIFIED)
	require.NoError(t, err)

	assert.Equal(t, accounts.IntentSignIn, useCase.asked[0].Intent)
}

func TestARefusedAddressSaysWhyInADetail(t *testing.T) {
	for want, err := range map[authv1.EmailRefusalReason]error{
		authv1.EmailRefusalReason_EMAIL_REFUSAL_REASON_INVALID:    fmt.Errorf("failed to read the address: %w", signin.ErrAddressInvalid),
		authv1.EmailRefusalReason_EMAIL_REFUSAL_REASON_DISPOSABLE: fmt.Errorf("%w: mailinator.com", signin.ErrAddressDisposable),
	} {
		t.Run(want.String(), func(t *testing.T) {
			_, got := startEmailSignIn(&stubUseCase{err: err}, authv1.SignInIntent_SIGN_IN_INTENT_SIGN_IN)

			var connectErr *connect.Error
			require.ErrorAs(t, got, &connectErr)
			assert.Equal(t, connect.CodeInvalidArgument, connectErr.Code())
			require.Len(t, connectErr.Details(), 1)
			detail, err := connectErr.Details()[0].Value()
			require.NoError(t, err)
			assert.Equal(t, want, detail.(*authv1.EmailRefusal).GetReason())
		})
	}
}

func TestEachRefusalHasItsCode(t *testing.T) {
	for want, err := range map[connect.Code]error{
		connect.CodeUnimplemented:     signin.ErrSignInOff,
		connect.CodePermissionDenied:  fmt.Errorf("%w: siteverify said no", attestation.ErrAttestationFailed),
		connect.CodeUnauthenticated:   fmt.Errorf("failed to find the account to link to: %w", accounts.ErrNoAccount),
		connect.CodeResourceExhausted: signin.ErrTooManyCodes,
		connect.CodeUnknown:           errors.New("cloudflare answered 503"),
	} {
		t.Run(want.String(), func(t *testing.T) {
			_, got := startEmailSignIn(&stubUseCase{err: err}, authv1.SignInIntent_SIGN_IN_INTENT_SIGN_IN)

			assert.Equal(t, want, connect.CodeOf(got))
			assert.NotContains(t, got.Error(), "siteverify")
		})
	}
}
