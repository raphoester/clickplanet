package complete_email_sign_in_usecase_test

import (
	"bytes"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/inmemory_account_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/attestation/open_attester"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin/aes_flow_sealer"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin/usecases/complete_email_sign_in_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin/usecases/start_email_sign_in_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const address = "player@example.com"

var (
	start    = time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	lifetime = accounts.Lifetime{}.WithDefaults()
)

type fixture struct {
	sealer     *aes_flow_sealer.Sealer
	store      *inmemory_account_store.Store
	clock      *cptime.FixedClock
	events     *cpbootstrap.RecordedEvents
	start      *start_email_sign_in_usecase.UseCase
	completion *complete_email_sign_in_usecase.UseCase
}

func setUp(t *testing.T) *fixture {
	t.Helper()

	sealer, err := aes_flow_sealer.New(bytes.Repeat([]byte{1}, 32))
	require.NoError(t, err)
	f := &fixture{sealer: sealer, store: inmemory_account_store.New(), clock: cptime.NewFixedClock(start), events: cpbootstrap.NewRecordedEvents()}
	post := signin.NewPost(signin.BlockedDomains{}, cpratelimit.New("sends", cpratelimit.Config{Burst: 100, PerSecond: 1}, f.clock), &signin.FakeMailer{})
	f.start = start_email_sign_in_usecase.New(open_attester.New(), f.store, f.challenges(true), post, f.clock)
	f.completion = f.useCase(true)
	return f
}

func (f *fixture) challenges(offered bool) *signin.Challenges {
	guesses := cpratelimit.New("guesses", cpratelimit.Config{Burst: signin.MaxAttempts, PerSecond: 1 / signin.ChallengeTTL.Seconds()}, f.clock)
	return signin.NewChallenges(offered, &signin.SequentialSecrets{}, &signin.SequentialCodes{}, f.sealer, guesses)
}

func (f *fixture) useCase(offered bool) *complete_email_sign_in_usecase.UseCase {
	admitter := signin.NewAdmitter(f.store, &accounts.SequentialIDs{}, &accounts.SequentialTokens{}, lifetime, f.events)
	return complete_email_sign_in_usecase.New(f.challenges(offered), admitter, f.clock)
}

func (f *fixture) guest(t *testing.T, account byte, token string) {
	t.Helper()

	require.NoError(t, f.store.CreateGuest(t.Context(), accounts.GuestSession(accounts.AccountID{15: account}, accounts.TokenOf(token), lifetime, start)))
}

func (f *fixture) began(t *testing.T, intent accounts.Intent, sessionCookie string) string {
	t.Helper()

	out, err := f.start.Execute(t.Context(), start_email_sign_in_usecase.In{
		Address: address, Intent: intent, AttestationToken: "widget", IP: "192.0.2.1", CookieHeader: sessionCookie,
	})
	require.NoError(t, err)
	cookie, err := http.ParseSetCookie(out.SetCookie)
	require.NoError(t, err)
	if sessionCookie == "" {
		return cookie.Name + "=" + cookie.Value
	}
	return cookie.Name + "=" + cookie.Value + "; " + sessionCookie
}

func (f *fixture) complete(t *testing.T, cookies string, code string) (*complete_email_sign_in_usecase.Out, error) {
	t.Helper()

	return f.completion.Execute(t.Context(), complete_email_sign_in_usecase.In{Code: code, CookieHeader: cookies})
}

func cookieValue(t *testing.T, setCookie string) string {
	t.Helper()

	cookie, err := http.ParseSetCookie(setCookie)
	require.NoError(t, err)
	return cookie.Value
}

func TestTheRightCodeMakesAnAccountForANewAddress(t *testing.T) {
	f := setUp(t)

	out, err := f.complete(t, f.began(t, accounts.IntentSignIn, ""), "000001")
	require.NoError(t, err)

	assert.Equal(t, accounts.AccountID{15: 1}, out.Account)
	assert.Equal(t, accounts.Created, out.Outcome)
	identity, err := f.store.Identity(t.Context(), signin.Email, address)
	require.NoError(t, err)
	assert.Equal(t, out.Account, identity.Account)
	assert.Equal(t, address, identity.Email)
	assert.True(t, identity.EmailVerified)
	session, err := f.store.Session(t.Context(), accounts.TokenOf(cookieValue(t, out.SetCookie)).Hash)
	require.NoError(t, err)
	assert.True(t, session.Linked, "an email account clicks as fast as one signed in with a provider")
	require.Len(t, f.events.Published(), 1)
	assert.True(t, proto.Equal(&authv1.SignedIn{AccountId: out.Account.String()}, f.events.Published()[0]))
}

func TestTheRightCodeLinksTheAddressToTheGuest(t *testing.T) {
	f := setUp(t)
	f.guest(t, 7, "guest")

	out, err := f.complete(t, f.began(t, accounts.IntentSignIn, "cp_sid=guest"), "000001")
	require.NoError(t, err)

	assert.Equal(t, accounts.AccountID{15: 7}, out.Account)
	assert.Equal(t, accounts.Linked, out.Outcome)
	_, err = f.store.Session(t.Context(), accounts.TokenOf("guest").Hash)
	assert.ErrorIs(t, err, accounts.ErrSessionNotFound, "the guest's session is replaced")
}

func TestAKnownAddressSignsInToItsAccount(t *testing.T) {
	f := setUp(t)
	first, err := f.complete(t, f.began(t, accounts.IntentSignIn, ""), "000001")
	require.NoError(t, err)
	f.guest(t, 9, "elsewhere")

	out, err := f.complete(t, f.began(t, accounts.IntentSignIn, "cp_sid=elsewhere"), "000002")
	require.NoError(t, err)

	assert.Equal(t, first.Account, out.Account)
	assert.Equal(t, accounts.SignedIn, out.Outcome)
}

func TestAWrongCodeMayBeTypedAgain(t *testing.T) {
	f := setUp(t)
	cookies := f.began(t, accounts.IntentSignIn, "")

	_, err := f.complete(t, cookies, "999999")
	require.ErrorIs(t, err, signin.ErrWrongCode)

	_, err = f.complete(t, cookies, "000001")
	assert.NoError(t, err)
}

func TestFiveWrongCodesEndTheChallenge(t *testing.T) {
	f := setUp(t)
	cookies := f.began(t, accounts.IntentSignIn, "")
	for range signin.MaxAttempts {
		_, err := f.complete(t, cookies, "999999")
		require.ErrorIs(t, err, signin.ErrWrongCode)
	}

	_, err := f.complete(t, cookies, "000001")
	require.ErrorIs(t, err, signin.ErrFlowInvalid, "even the right code is refused once the guesses are spent")

	_, err = f.complete(t, f.began(t, accounts.IntentSignIn, ""), "000002")
	assert.NoError(t, err, "a new challenge has guesses of its own")
}

func TestALapsedChallengeOrNoCookieIsStartedAgain(t *testing.T) {
	f := setUp(t)
	cookies := f.began(t, accounts.IntentSignIn, "")
	f.clock.Advance(signin.ChallengeTTL)

	_, err := f.complete(t, cookies, "000001")
	require.ErrorIs(t, err, signin.ErrFlowInvalid)
	_, err = f.complete(t, "", "000001")
	require.ErrorIs(t, err, signin.ErrFlowInvalid)
	_, err = f.complete(t, "cp_email=forged", "000001")
	assert.ErrorIs(t, err, signin.ErrFlowInvalid)
}

func TestALinkOfAnAddressAnotherAccountUsesIsRefusedAndChangesNothing(t *testing.T) {
	f := setUp(t)
	_, err := f.complete(t, f.began(t, accounts.IntentSignIn, ""), "000001")
	require.NoError(t, err)
	f.guest(t, 7, "guest")

	_, err = f.complete(t, f.began(t, accounts.IntentLink, "cp_sid=guest"), "000002")

	require.ErrorIs(t, err, accounts.ErrIdentityLinkedElsewhere)
	_, err = f.store.Session(t.Context(), accounts.TokenOf("guest").Hash)
	assert.NoError(t, err, "the browser keeps its session")
}

func TestALinkWhoseBrowserLeftTheAccountIsStartedAgain(t *testing.T) {
	f := setUp(t)
	f.guest(t, 7, "guest")
	f.guest(t, 8, "other")
	challenge := f.began(t, accounts.IntentLink, "cp_sid=guest")
	cookies := challenge[:len(challenge)-len("; cp_sid=guest")] + "; cp_sid=other"

	_, err := f.complete(t, cookies, "000001")

	assert.ErrorIs(t, err, signin.ErrFlowInvalid)
}

func TestEmailSignInOffCompletesNothing(t *testing.T) {
	f := setUp(t)
	cookies := f.began(t, accounts.IntentSignIn, "")

	_, err := f.useCase(false).Execute(t.Context(), complete_email_sign_in_usecase.In{Code: "000001", CookieHeader: cookies})

	assert.ErrorIs(t, err, signin.ErrSignInOff)
}
