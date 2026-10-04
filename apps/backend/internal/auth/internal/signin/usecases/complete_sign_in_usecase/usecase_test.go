package complete_sign_in_usecase_test

import (
	"bytes"
	"errors"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/inmemory_account_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin/aes_flow_sealer"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin/usecases/complete_sign_in_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin/usecases/start_sign_in_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	start    = time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	lifetime = accounts.Lifetime{}.WithDefaults()
	claim    = accounts.ClaimOf("google-user", "a@example.com", true)
)

type fixture struct {
	sealer     *aes_flow_sealer.Sealer
	google     *signin.FakeProvider
	store      *inmemory_account_store.Store
	clock      *cptime.FixedClock
	events     *cpbootstrap.RecordedEvents
	start      *start_sign_in_usecase.UseCase
	completion *complete_sign_in_usecase.UseCase
}

func setUp(t *testing.T) *fixture {
	t.Helper()

	sealer, err := aes_flow_sealer.New(bytes.Repeat([]byte{1}, 32))
	require.NoError(t, err)
	f := &fixture{sealer: sealer, google: signin.NewFakeProvider(signin.Google), store: inmemory_account_store.New(), clock: cptime.NewFixedClock(start), events: cpbootstrap.NewRecordedEvents()}
	providers := signin.Providers{signin.Google: f.google}
	f.start = start_sign_in_usecase.New(providers, f.store, &signin.SequentialSecrets{}, sealer, f.clock)
	f.completion = complete_sign_in_usecase.New(providers, sealer, f.admitter(), f.clock)
	return f
}

func (f *fixture) admitter() *signin.Admitter {
	return signin.NewAdmitter(f.store, &accounts.SequentialIDs{}, &accounts.SequentialTokens{}, lifetime, f.events)
}

func (f *fixture) guest(t *testing.T, account byte, token string) {
	t.Helper()

	require.NoError(t, f.store.CreateGuest(t.Context(), accounts.GuestSession(accounts.AccountID{15: account}, accounts.TokenOf(token), lifetime, start)))
}

func (f *fixture) began(t *testing.T, intent accounts.Intent, sessionCookie string) (string, string) {
	t.Helper()

	out, err := f.start.Execute(t.Context(), start_sign_in_usecase.In{Provider: signin.Google, Intent: intent, CookieHeader: sessionCookie})
	require.NoError(t, err)
	cookie, err := http.ParseSetCookie(out.SetCookie)
	require.NoError(t, err)
	authorization, err := url.Parse(out.AuthorizationURL)
	require.NoError(t, err)
	return cookie.Name + "=" + cookie.Value, authorization.Query().Get("state")
}

func (f *fixture) complete(t *testing.T, sessionCookie string) (*complete_sign_in_usecase.Out, error) {
	t.Helper()

	return f.finish(t, accounts.IntentSignIn, sessionCookie, claim)
}

func (f *fixture) link(t *testing.T, sessionCookie string, linked accounts.Claim) (*complete_sign_in_usecase.Out, error) {
	t.Helper()

	return f.finish(t, accounts.IntentLink, sessionCookie, linked)
}

func (f *fixture) finish(t *testing.T, intent accounts.Intent, sessionCookie string, granted accounts.Claim) (*complete_sign_in_usecase.Out, error) {
	t.Helper()

	flowCookie, state := f.began(t, intent, sessionCookie)
	f.google.Grant("the-code", granted)
	cookies := flowCookie
	if sessionCookie != "" {
		cookies += "; " + sessionCookie
	}
	return f.completion.Execute(t.Context(), complete_sign_in_usecase.In{Code: "the-code", State: state, CookieHeader: cookies})
}

func (f *fixture) linkedAccount(t *testing.T, account byte, provider string, subject string, token string) {
	t.Helper()

	identity := accounts.NewIdentity(provider, accounts.ClaimOf(subject, "", false), accounts.AccountID{15: account}, start)
	linked := accounts.LinkedSession(identity.Account(), accounts.TokenOf(token), lifetime, start)
	require.NoError(t, f.store.SaveSignIn(t.Context(), accounts.NewSignIn(linked).WithNewAccount().WithIdentity(identity)))
}

func TestAGoogleSignInJoinsTheAccountThatHoldsItsAddress(t *testing.T) {
	f := setUp(t)
	identity := accounts.NewIdentity(signin.Email, accounts.ClaimOf("a@example.com", "a@example.com", true), accounts.AccountID{15: 9}, start)
	linked := accounts.LinkedSession(identity.Account(), accounts.TokenOf("email-token"), lifetime, start)
	require.NoError(t, f.store.SaveSignIn(t.Context(), accounts.NewSignIn(linked).WithNewAccount().WithIdentity(identity)))

	out, err := f.complete(t, "")
	require.NoError(t, err)

	assert.Equal(t, accounts.AccountID{15: 9}, out.Account())
	assert.Equal(t, accounts.Joined, out.Outcome())
	google, err := f.store.Identity(t.Context(), signin.Google, "google-user")
	require.NoError(t, err)
	assert.Equal(t, accounts.AccountID{15: 9}, google.Account())
}

func TestAnUnverifiedAddressJoinsNothing(t *testing.T) {
	f := setUp(t)
	identity := accounts.NewIdentity(signin.Email, accounts.ClaimOf("a@example.com", "a@example.com", true), accounts.AccountID{15: 9}, start)
	linked := accounts.LinkedSession(identity.Account(), accounts.TokenOf("email-token"), lifetime, start)
	require.NoError(t, f.store.SaveSignIn(t.Context(), accounts.NewSignIn(linked).WithNewAccount().WithIdentity(identity)))

	out, err := f.finish(t, accounts.IntentSignIn, "", accounts.ClaimOf("google-user", "a@example.com", false))
	require.NoError(t, err)

	assert.Equal(t, accounts.Created, out.Outcome())
	assert.NotEqual(t, accounts.AccountID{15: 9}, out.Account())
}

func (f *fixture) assertPublished(t *testing.T, want ...proto.Message) {
	t.Helper()

	published := f.events.Published()
	require.Len(t, published, len(want))
	for i := range want {
		assert.True(t, proto.Equal(want[i], published[i]), "event %d: want %v, got %v", i, want[i], published[i])
	}
}

func cookieValue(t *testing.T, setCookie string) string {
	t.Helper()

	cookie, err := http.ParseSetCookie(setCookie)
	require.NoError(t, err)
	return cookie.Value
}

func TestANewIdentityLinksToTheGuestAndReplacesItsSession(t *testing.T) {
	f := setUp(t)
	f.guest(t, 7, "guest-token")

	out, err := f.complete(t, "cp_sid=guest-token")
	require.NoError(t, err)

	assert.Equal(t, accounts.AccountID{15: 7}, out.Account())
	assert.Equal(t, accounts.Linked, out.Outcome())
	assert.Equal(t, "token-1", cookieValue(t, out.SetCookie()))
	session, err := f.store.Session(t.Context(), accounts.TokenOf("token-1").Hash())
	require.NoError(t, err)
	assert.Equal(t, start.Add(30*24*time.Hour), session.ExpiresAt(), "a linked session lasts the linked lifetime")
	_, err = f.store.Session(t.Context(), accounts.TokenOf("guest-token").Hash())
	require.ErrorIs(t, err, accounts.ErrSessionNotFound)
	identity, err := f.store.Identity(t.Context(), signin.Google, "google-user")
	require.NoError(t, err)
	assert.Equal(t, accounts.IdentityOf("google", "google-user", accounts.AccountID{15: 7}, "a@example.com", true, start), identity)
	f.assertPublished(t, &authv1.SignedIn{PreviousAccountId: accounts.AccountID{15: 7}.String(), AccountId: accounts.AccountID{15: 7}.String()})
}

func TestAKnownIdentitySignsInToItsAccountAndLeavesTheGuestAsItWas(t *testing.T) {
	f := setUp(t)
	_, err := f.complete(t, "")
	require.NoError(t, err)
	f.guest(t, 7, "guest-token")

	out, err := f.complete(t, "cp_sid=guest-token")
	require.NoError(t, err)

	assert.Equal(t, accounts.AccountID{15: 1}, out.Account())
	assert.Equal(t, accounts.SignedIn, out.Outcome())
	guest, err := f.store.Account(t.Context(), accounts.AccountID{15: 7})
	require.NoError(t, err)
	assert.Empty(t, guest.Providers(), "nothing is merged, and the guest gains no identity")
	f.assertPublished(
		t,
		&authv1.SignedIn{AccountId: accounts.AccountID{15: 1}.String()},
		&authv1.SignedIn{PreviousAccountId: accounts.AccountID{15: 7}.String(), AccountId: accounts.AccountID{15: 1}.String()},
	)
}

func TestABrowserWithNoAccountGetsANewOne(t *testing.T) {
	f := setUp(t)

	out, err := f.complete(t, "cp_sid=made-up")
	require.NoError(t, err)

	assert.Equal(t, accounts.AccountID{15: 1}, out.Account())
	assert.Equal(t, accounts.Created, out.Outcome())
	f.assertPublished(t, &authv1.SignedIn{AccountId: accounts.AccountID{15: 1}.String()})
	made, err := f.store.Account(t.Context(), out.Account())
	require.NoError(t, err)
	identity := accounts.NewIdentity(signin.Google, claim, out.Account(), start)
	assert.Equal(t, accounts.AccountOf(out.Account(), start, []accounts.Identity{*identity}), made, "the account is made, on the day of the sign-in")
}

func TestAnAccountHoldingTheProviderAlreadyIsNotGivenASecondUserOfIt(t *testing.T) {
	f := setUp(t)
	f.linkedAccount(t, 7, signin.Google, "someone-else", "linked")

	out, err := f.complete(t, "cp_sid=linked")
	require.NoError(t, err)

	assert.Equal(t, accounts.Created, out.Outcome())
	assert.NotEqual(t, accounts.AccountID{15: 7}, out.Account())
}

func TestAGoogleSignInJoinsTheDiscordAccountWithItsVerifiedAddress(t *testing.T) {
	f := setUp(t)
	identity := accounts.NewIdentity("discord", accounts.ClaimOf("discord-user", claim.VerifiedEmail(), true), accounts.AccountID{15: 7}, start)
	linked := accounts.LinkedSession(identity.Account(), accounts.TokenOf("discord"), lifetime, start)
	require.NoError(t, f.store.SaveSignIn(t.Context(), accounts.NewSignIn(linked).WithNewAccount().WithIdentity(identity)))

	out, err := f.complete(t, "")
	require.NoError(t, err)

	assert.Equal(t, accounts.Joined, out.Outcome())
	assert.Equal(t, accounts.AccountID{15: 7}, out.Account())
	joined, err := f.store.Account(t.Context(), accounts.AccountID{15: 7})
	require.NoError(t, err)
	assert.Equal(t, []string{"discord", "google"}, joined.Providers())
}

func TestLinkingANewIdentityLinksItToTheAccountTheBrowserIsOn(t *testing.T) {
	f := setUp(t)
	f.linkedAccount(t, 7, signin.Discord, "discord-user", "discord-token")

	out, err := f.link(t, "cp_sid=discord-token", claim)
	require.NoError(t, err)

	assert.Equal(t, accounts.AccountID{15: 7}, out.Account())
	assert.Equal(t, accounts.Linked, out.Outcome())
	account, err := f.store.Account(t.Context(), accounts.AccountID{15: 7})
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"discord", "google"}, account.Providers())
}

func TestLinkingAnIdentityTheAccountAlreadyHasKeepsTheBrowserThere(t *testing.T) {
	f := setUp(t)
	f.linkedAccount(t, 7, signin.Google, claim.Subject(), "google-token")

	out, err := f.link(t, "cp_sid=google-token", claim)
	require.NoError(t, err)

	assert.Equal(t, accounts.AccountID{15: 7}, out.Account())
	assert.Equal(t, accounts.SignedIn, out.Outcome())
}

func TestLinkingAnIdentityAnotherAccountUsesIsRefusedAndWritesNothing(t *testing.T) {
	f := setUp(t)
	f.linkedAccount(t, 1, signin.Google, claim.Subject(), "other-browser")
	f.linkedAccount(t, 7, signin.Discord, "discord-user", "discord-token")

	out, err := f.link(t, "cp_sid=discord-token", claim)

	require.ErrorIs(t, err, accounts.ErrIdentityLinkedElsewhere)
	assert.Nil(t, out)
	session, err := f.store.Session(t.Context(), accounts.TokenOf("discord-token").Hash())
	require.NoError(t, err, "the browser keeps its session")
	assert.Equal(t, accounts.AccountID{15: 7}, session.Account())
	identity, err := f.store.Identity(t.Context(), signin.Google, claim.Subject())
	require.NoError(t, err)
	assert.Equal(t, accounts.AccountID{15: 1}, identity.Account())
	current, err := f.store.Account(t.Context(), accounts.AccountID{15: 7})
	require.NoError(t, err)
	assert.Equal(t, []string{"discord"}, current.Providers())
	assert.Empty(t, f.events.Published(), "a refusal moves no browser")
}

func TestLinkingASecondUserOfAProviderIsRefused(t *testing.T) {
	f := setUp(t)
	f.linkedAccount(t, 7, signin.Google, "someone-else", "linked")

	out, err := f.link(t, "cp_sid=linked", claim)

	require.ErrorIs(t, err, accounts.ErrProviderAlreadyLinked)
	assert.Nil(t, out)
	_, err = f.store.Identity(t.Context(), signin.Google, claim.Subject())
	assert.ErrorIs(t, err, accounts.ErrIdentityNotFound)
}

func TestALinkWhoseBrowserLeftTheAccountIsRefusedBeforeTheProviderIsAsked(t *testing.T) {
	f := setUp(t)
	f.guest(t, 7, "guest-token")
	f.guest(t, 8, "other-guest")
	flowCookie, state := f.began(t, accounts.IntentLink, "cp_sid=guest-token")
	f.google.Grant("the-code", claim)

	for name, session := range map[string]string{"signed out": "", "on another account": "; cp_sid=other-guest"} {
		t.Run(name, func(t *testing.T) {
			out, err := f.completion.Execute(t.Context(), complete_sign_in_usecase.In{Code: "the-code", State: state, CookieHeader: flowCookie + session})

			require.ErrorIs(t, err, signin.ErrFlowInvalid)
			assert.Nil(t, out)
		})
	}

	_, err := f.completion.Execute(t.Context(), complete_sign_in_usecase.In{Code: "the-code", State: state, CookieHeader: flowCookie + "; cp_sid=guest-token"})
	assert.NoError(t, err, "the code was never spent on a refused callback")
}

func TestACallbackThatDoesNotMatchTheFlowIsRefusedBeforeTheProviderIsAsked(t *testing.T) {
	f := setUp(t)
	flowCookie, state := f.began(t, accounts.IntentSignIn, "")
	f.google.Grant("the-code", claim)

	for name, in := range map[string]complete_sign_in_usecase.In{
		"no flow cookie": {Code: "the-code", State: state},
		"another state":  {Code: "the-code", State: "forged", CookieHeader: flowCookie},
		"forged cookie":  {Code: "the-code", State: state, CookieHeader: "cp_oauth=forged"},
	} {
		t.Run(name, func(t *testing.T) {
			out, err := f.completion.Execute(t.Context(), in)

			require.ErrorIs(t, err, signin.ErrFlowInvalid)
			assert.Nil(t, out)
		})
	}

	f.clock.Advance(signin.FlowTTL)
	_, err := f.completion.Execute(t.Context(), complete_sign_in_usecase.In{Code: "the-code", State: state, CookieHeader: flowCookie})
	require.ErrorIs(t, err, signin.ErrFlowInvalid, "a lapsed flow")

	f.clock.Advance(-signin.FlowTTL)
	_, err = f.completion.Execute(t.Context(), complete_sign_in_usecase.In{Code: "the-code", State: state, CookieHeader: flowCookie})
	assert.NoError(t, err, "the code was never spent on a refused callback")
}

func TestACodeTheProviderRefusesSignsNothingIn(t *testing.T) {
	f := setUp(t)
	flowCookie, state := f.began(t, accounts.IntentSignIn, "")

	out, err := f.completion.Execute(t.Context(), complete_sign_in_usecase.In{Code: "never-granted", State: state, CookieHeader: flowCookie})

	require.ErrorIs(t, err, signin.ErrProviderRefused)
	assert.Nil(t, out)
}

func TestAStoreFailureFailsTheSignIn(t *testing.T) {
	f := setUp(t)
	flowCookie, state := f.began(t, accounts.IntentSignIn, "")
	f.google.Grant("the-code", claim)
	f.store.FailWith(errors.New("postgres is down"))

	_, err := f.completion.Execute(t.Context(), complete_sign_in_usecase.In{Code: "the-code", State: state, CookieHeader: flowCookie + "; cp_sid=guest"})

	require.Error(t, err)
	assert.NotErrorIs(t, err, signin.ErrFlowInvalid)
	assert.NotErrorIs(t, err, signin.ErrProviderRefused)
}

func TestSignInOffCompletesNothing(t *testing.T) {
	f := setUp(t)
	useCase := complete_sign_in_usecase.New(signin.Providers{}, f.sealer, f.admitter(), f.clock)

	_, err := useCase.Execute(t.Context(), complete_sign_in_usecase.In{Code: "the-code", State: "state"})

	assert.ErrorIs(t, err, signin.ErrSignInOff)
}
