package e2e_test

import (
	"errors"
	"fmt"
	"regexp"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth"
)

var sixDigits = regexp.MustCompile(`[0-9]{6}`)

func (b *browser) askCode(mailer *auth.FakeMailer, email string, intent authv1.SignInIntent) string {
	b.t.Helper()

	req := connect.NewRequest(&authv1.StartEmailSignInRequest{Email: email, Intent: intent, AttestationToken: "unused"})
	b.send(req.Header())
	res, err := b.client.StartEmailSignIn(b.t.Context(), req)
	require.NoError(b.t, err)
	b.keep(res.Header())

	sent := mailer.Sent()
	require.NotEmpty(b.t, sent)
	return sixDigits.FindString(sent[len(sent)-1].Letter.Subject())
}

func (b *browser) typeCode(code string) (*authv1.CompleteEmailSignInResponse, error) {
	b.t.Helper()

	req := connect.NewRequest(&authv1.CompleteEmailSignInRequest{Code: code})
	b.send(req.Header())
	res, err := b.client.CompleteEmailSignIn(b.t.Context(), req)
	var refused *connect.Error
	if errors.As(err, &refused) {
		b.keep(refused.Meta())
	}
	if err != nil {
		return nil, fmt.Errorf("CompleteEmailSignIn failed: %w", err)
	}
	b.keep(res.Header())
	return res.Msg, nil
}

func TestAGuestSignsInWithAnEmailCodeAndKeepsItsAccount(t *testing.T) {
	stack, fakes := startSignIn(t)
	player := stack.browser(t)
	guest := player.mint()

	code := player.askCode(fakes.Mailer, "Player@Example.com", authv1.SignInIntent_SIGN_IN_INTENT_SIGN_IN)
	linked, err := player.typeCode(code)
	require.NoError(t, err)

	assert.Equal(t, "player@example.com", string(fakes.Mailer.Sent()[0].To))
	assert.Equal(t, authv1.SignInOutcome_SIGN_IN_OUTCOME_LINKED, linked.GetOutcome())
	assert.Equal(t, guest.String(), linked.GetAccountId())
	assert.NotContains(t, player.cookies, "cp_email", "the challenge cookie is cleared")
	assert.Equal(t, guest, player.mint())
	me, err := player.me()
	require.NoError(t, err)
	assert.Equal(t, authv1.AccountKind_ACCOUNT_KIND_LINKED, me.GetKind())
	assert.Equal(t, []authv1.Provider{authv1.Provider_PROVIDER_EMAIL}, me.GetProviders())
}

func TestTheSameAddressOnAnotherBrowserSignsInToTheSameAccount(t *testing.T) {
	stack, fakes := startSignIn(t)
	first := stack.browser(t)
	owner, err := first.typeCode(first.askCode(fakes.Mailer, "player@example.com", authv1.SignInIntent_SIGN_IN_INTENT_SIGN_IN))
	require.NoError(t, err)

	second := stack.browser(t)
	second.mint()
	signedIn, err := second.typeCode(second.askCode(fakes.Mailer, "player@example.com", authv1.SignInIntent_SIGN_IN_INTENT_SIGN_IN))
	require.NoError(t, err)

	assert.Equal(t, authv1.SignInOutcome_SIGN_IN_OUTCOME_SIGNED_IN, signedIn.GetOutcome())
	assert.Equal(t, owner.GetAccountId(), signedIn.GetAccountId())
}

func TestAWrongCodeKeepsTheChallengeForTheRightOne(t *testing.T) {
	stack, fakes := startSignIn(t)
	player := stack.browser(t)
	code := player.askCode(fakes.Mailer, "player@example.com", authv1.SignInIntent_SIGN_IN_INTENT_SIGN_IN)

	_, err := player.typeCode("999999")
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	require.Contains(t, player.cookies, "cp_email")

	created, err := player.typeCode(code)
	require.NoError(t, err)
	assert.Equal(t, authv1.SignInOutcome_SIGN_IN_OUTCOME_CREATED, created.GetOutcome())
}

func TestADisposableAddressIsRefusedWithItsReason(t *testing.T) {
	stack, fakes := startSignIn(t)
	player := stack.browser(t)

	req := connect.NewRequest(&authv1.StartEmailSignInRequest{Email: "player@mailinator.com", AttestationToken: "unused"})
	player.send(req.Header())
	_, err := player.client.StartEmailSignIn(t.Context(), req)

	var refused *connect.Error
	require.ErrorAs(t, err, &refused)
	assert.Equal(t, connect.CodeInvalidArgument, refused.Code())
	require.Len(t, refused.Details(), 1)
	detail, err := refused.Details()[0].Value()
	require.NoError(t, err)
	assert.Equal(t, authv1.EmailRefusalReason_EMAIL_REFUSAL_REASON_DISPOSABLE, detail.(*authv1.EmailRefusal).GetReason())
	assert.Empty(t, fakes.Mailer.Sent())
}

func TestEmailSignInIsAbsentWhileItIsOff(t *testing.T) {
	stack := startAuth(t)

	_, err := stack.browser(t).client.StartEmailSignIn(t.Context(), connect.NewRequest(&authv1.StartEmailSignInRequest{Email: "player@example.com"}))

	assert.Equal(t, connect.CodeUnimplemented, connect.CodeOf(err))
}

func TestAnEmailCodeForAGoogleAddressSignsInToTheGoogleAccount(t *testing.T) {
	stack, fakes := startSignIn(t)
	google := stack.browser(t).signIn(authv1.Provider_PROVIDER_GOOGLE, fakes.Google, "code-1",
		auth.ClaimOf("google-1", "player@example.com", true))

	player := stack.browser(t)
	player.mint()
	signedIn, err := player.typeCode(player.askCode(fakes.Mailer, "Player@Example.com", authv1.SignInIntent_SIGN_IN_INTENT_SIGN_IN))
	require.NoError(t, err)

	assert.Equal(t, authv1.SignInOutcome_SIGN_IN_OUTCOME_SIGNED_IN, signedIn.GetOutcome())
	assert.Equal(t, google.GetAccountId(), signedIn.GetAccountId())
	me, err := player.me()
	require.NoError(t, err)
	assert.Equal(t, []authv1.Provider{authv1.Provider_PROVIDER_GOOGLE, authv1.Provider_PROVIDER_EMAIL}, me.GetProviders())
}
