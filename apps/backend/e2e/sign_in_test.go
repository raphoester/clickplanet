package e2e_test

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1/authv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

// browser keeps the cookies the API sets, as a web client with credentials does.
type browser struct {
	t       *testing.T
	stack   authStack
	client  authv1connect.AuthServiceClient
	cookies map[string]string
}

func (s authStack) browser(t *testing.T) *browser {
	t.Helper()

	return &browser{t: t, stack: s, client: authv1connect.NewAuthServiceClient(http.DefaultClient, s.baseURL), cookies: map[string]string{}}
}

func (b *browser) send(header http.Header) {
	b.t.Helper()

	header.Set("X-Real-IP", callerIP)
	pairs := make([]string, 0, len(b.cookies))
	for name, value := range b.cookies {
		pairs = append(pairs, name+"="+value)
	}
	header.Set("Cookie", strings.Join(pairs, "; "))
}

func (b *browser) keep(header http.Header) {
	b.t.Helper()

	for _, line := range header.Values("Set-Cookie") {
		cookie, err := http.ParseSetCookie(line)
		require.NoError(b.t, err)
		if cookie.MaxAge < 0 {
			delete(b.cookies, cookie.Name)
			continue
		}
		b.cookies[cookie.Name] = cookie.Value
	}
}

func (b *browser) mint() cpsession.AccountID {
	b.t.Helper()

	req := connect.NewRequest(&authv1.CreateSessionRequest{AttestationToken: "unused"})
	b.send(req.Header())
	res, err := b.client.CreateSession(b.t.Context(), req)
	require.NoError(b.t, err)
	b.keep(res.Header())

	return b.stack.accountIn(b.t, res.Msg.GetToken())
}

func (b *browser) signIn(provider authv1.Provider, fake *auth.FakeProvider, code string, claim auth.Claim) *authv1.CompleteSignInResponse {
	b.t.Helper()

	start := connect.NewRequest(&authv1.StartSignInRequest{Provider: provider})
	b.send(start.Header())
	started, err := b.client.StartSignIn(b.t.Context(), start)
	require.NoError(b.t, err)
	b.keep(started.Header())

	authorization, err := url.Parse(started.Msg.GetAuthorizationUrl())
	require.NoError(b.t, err)
	fake.Grant(code, claim)

	complete := connect.NewRequest(&authv1.CompleteSignInRequest{Code: code, State: authorization.Query().Get("state")})
	b.send(complete.Header())
	completed, err := b.client.CompleteSignIn(b.t.Context(), complete)
	require.NoError(b.t, err)
	b.keep(completed.Header())

	return completed.Msg
}

func (b *browser) me() (*authv1.GetMeResponse, error) {
	b.t.Helper()

	req := connect.NewRequest(&authv1.GetMeRequest{})
	b.send(req.Header())
	res, err := b.client.GetMe(b.t.Context(), req)
	if err != nil {
		return nil, fmt.Errorf("GetMe failed: %w", err)
	}
	return res.Msg, nil
}

func startSignIn(t *testing.T) (authStack, auth.FakeProviders) {
	t.Helper()

	var fakes auth.FakeProviders
	stack := startAuthModule(t, func(config auth.Config) cpbootstrap.Module {
		module, providers := auth.NewModuleWithFakeProviders(config)
		fakes = providers
		return module
	})
	return stack, fakes
}

func TestAGuestLinksAProviderAndKeepsItsAccount(t *testing.T) {
	stack, fakes := startSignIn(t)
	player := stack.browser(t)
	guest := player.mint()

	linked := player.signIn(authv1.Provider_PROVIDER_GOOGLE, fakes.Google, "code-1", auth.Claim{Subject: "google-1", Email: "a@example.com", EmailVerified: true})

	assert.Equal(t, authv1.SignInOutcome_SIGN_IN_OUTCOME_LINKED, linked.GetOutcome())
	assert.Equal(t, guest.String(), linked.GetAccountId())
	assert.NotContains(t, player.cookies, "cp_oauth", "the flow cookie is cleared")
	assert.Equal(t, guest, player.mint(), "the new click token carries the same account")
	me, err := player.me()
	require.NoError(t, err)
	assert.Equal(t, authv1.AccountKind_ACCOUNT_KIND_LINKED, me.GetKind())
	assert.Equal(t, []authv1.Provider{authv1.Provider_PROVIDER_GOOGLE}, me.GetProviders())
}

func TestAKnownIdentitySignsInToItsAccountAndLeavesTheGuestAlone(t *testing.T) {
	stack, fakes := startSignIn(t)
	claim := auth.Claim{Subject: "discord-1"}
	first := stack.browser(t)
	owner := first.mint()
	first.signIn(authv1.Provider_PROVIDER_DISCORD, fakes.Discord, "code-1", claim)

	second := stack.browser(t)
	guest := second.mint()
	signedIn := second.signIn(authv1.Provider_PROVIDER_DISCORD, fakes.Discord, "code-2", claim)

	assert.Equal(t, authv1.SignInOutcome_SIGN_IN_OUTCOME_SIGNED_IN, signedIn.GetOutcome())
	assert.Equal(t, owner.String(), signedIn.GetAccountId())
	assert.NotEqual(t, guest, owner)
	assert.Equal(t, owner, second.mint())
}

func TestABrowserWithNoAccountSignsInToANewOne(t *testing.T) {
	stack, fakes := startSignIn(t)
	player := stack.browser(t)

	created := player.signIn(authv1.Provider_PROVIDER_GOOGLE, fakes.Google, "code-1", auth.Claim{Subject: "google-1"})

	assert.Equal(t, authv1.SignInOutcome_SIGN_IN_OUTCOME_CREATED, created.GetOutcome())
	assert.Equal(t, created.GetAccountId(), player.mint().String())
}

func TestACallbackFromAnotherBrowserIsRefused(t *testing.T) {
	stack, fakes := startSignIn(t)
	victim, attacker := stack.browser(t), stack.browser(t)
	victim.mint()

	start := connect.NewRequest(&authv1.StartSignInRequest{Provider: authv1.Provider_PROVIDER_GOOGLE})
	attacker.send(start.Header())
	started, err := attacker.client.StartSignIn(t.Context(), start)
	require.NoError(t, err)
	authorization, err := url.Parse(started.Msg.GetAuthorizationUrl())
	require.NoError(t, err)
	fakes.Google.Grant("attacker-code", auth.Claim{Subject: "attacker"})

	complete := connect.NewRequest(&authv1.CompleteSignInRequest{Code: "attacker-code", State: authorization.Query().Get("state")})
	victim.send(complete.Header())
	_, err = victim.client.CompleteSignIn(t.Context(), complete)

	assert.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
}

func TestSigningOutEndsOneSessionAndEverywhereEndsThemAll(t *testing.T) {
	stack, fakes := startSignIn(t)
	claim := auth.Claim{Subject: "google-1"}
	laptop, phone, tablet := stack.browser(t), stack.browser(t), stack.browser(t)
	laptop.signIn(authv1.Provider_PROVIDER_GOOGLE, fakes.Google, "code-1", claim)
	phone.signIn(authv1.Provider_PROVIDER_GOOGLE, fakes.Google, "code-2", claim)
	tablet.signIn(authv1.Provider_PROVIDER_GOOGLE, fakes.Google, "code-3", claim)

	signOut := connect.NewRequest(&authv1.SignOutRequest{})
	laptop.send(signOut.Header())
	res, err := laptop.client.SignOut(t.Context(), signOut)
	require.NoError(t, err)
	laptop.keep(res.Header())

	assert.NotContains(t, laptop.cookies, "cp_sid")
	_, err = laptop.me()
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	_, err = phone.me()
	require.NoError(t, err, "the other sessions live on")

	everywhere := connect.NewRequest(&authv1.SignOutEverywhereRequest{})
	phone.send(everywhere.Header())
	_, err = phone.client.SignOutEverywhere(t.Context(), everywhere)
	require.NoError(t, err)

	_, err = tablet.me()
	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

func TestADeletedAccountIsGoneAndItsIdentityIsFreeAgain(t *testing.T) {
	stack, fakes := startSignIn(t)
	claim := auth.Claim{Subject: "google-1"}
	player := stack.browser(t)
	deleted := player.signIn(authv1.Provider_PROVIDER_GOOGLE, fakes.Google, "code-1", claim)

	req := connect.NewRequest(&authv1.DeleteAccountRequest{})
	player.send(req.Header())
	res, err := player.client.DeleteAccount(t.Context(), req)
	require.NoError(t, err)
	player.keep(res.Header())

	assert.NotContains(t, player.cookies, "cp_sid")
	again := stack.browser(t).signIn(authv1.Provider_PROVIDER_GOOGLE, fakes.Google, "code-2", claim)
	assert.Equal(t, authv1.SignInOutcome_SIGN_IN_OUTCOME_CREATED, again.GetOutcome())
	assert.NotEqual(t, deleted.GetAccountId(), again.GetAccountId())
}

func TestSignInIsAbsentWhileItIsOff(t *testing.T) {
	stack := startAuth(t)

	_, err := stack.browser(t).client.StartSignIn(t.Context(), connect.NewRequest(&authv1.StartSignInRequest{Provider: authv1.Provider_PROVIDER_GOOGLE}))

	assert.Equal(t, connect.CodeUnimplemented, connect.CodeOf(err))
}

func TestTheOfferedProvidersAreAnsweredAndNoCookieIsSet(t *testing.T) {
	stack, _ := startSignIn(t)
	b := stack.browser(t)

	req := connect.NewRequest(&authv1.GetSignInOptionsRequest{})
	b.send(req.Header())
	res, err := b.client.GetSignInOptions(t.Context(), req)
	require.NoError(t, err)

	assert.Equal(t, []authv1.Provider{authv1.Provider_PROVIDER_DISCORD, authv1.Provider_PROVIDER_GOOGLE}, res.Msg.GetProviders())
	assert.Empty(t, res.Header().Values("Set-Cookie"))
}

func TestNoProviderIsOfferedWhileSignInIsOff(t *testing.T) {
	stack := startAuth(t)

	res, err := stack.browser(t).client.GetSignInOptions(t.Context(), connect.NewRequest(&authv1.GetSignInOptionsRequest{}))
	require.NoError(t, err)

	assert.Empty(t, res.Msg.GetProviders())
}
