package e2e_test

import (
	"net/http"
	"net/url"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1/authv1connect"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth"
)

func (p *gamer) becomeOperator() {
	p.t.Helper()

	_, err := p.stack.schema(p.t, "auth").ExecContext(p.t.Context(),
		`UPDATE accounts SET operator = true WHERE id = $1`, p.account())
	require.NoError(p.t, err)
}

func (p *gamer) desk() authv1connect.AdminServiceClient {
	return authv1connect.NewAdminServiceClient(http.DefaultClient, p.stack.adminURL)
}

func (p *gamer) adminSignIn(googleSubject string) error {
	p.t.Helper()

	start := connect.NewRequest(&authv1.AdminStartSignInRequest{})
	start.Header().Set("Origin", adminOrigin)
	started, err := p.desk().StartSignIn(p.t.Context(), start)
	if err != nil {
		return err //nolint:wrapcheck // the tests read the connect code.
	}
	flow, err := http.ParseSetCookie(started.Header().Get("Set-Cookie"))
	require.NoError(p.t, err)
	authorization, err := url.Parse(started.Msg.GetAuthorizationUrl())
	require.NoError(p.t, err)

	p.stack.fakes.AdminGoogle.Grant(googleSubject, auth.ClaimOf(googleSubject, "", false))
	complete := connect.NewRequest(&authv1.AdminCompleteSignInRequest{
		Code: googleSubject, State: authorization.Query().Get("state"),
	})
	complete.Header().Set("Origin", adminOrigin)
	complete.Header().Set("Cookie", p.cookie+"; "+flow.Name+"="+flow.Value)
	completed, err := p.desk().CompleteSignIn(p.t.Context(), complete)
	if err != nil {
		return err //nolint:wrapcheck // the tests read the connect code.
	}

	for _, line := range completed.Header().Values("Set-Cookie") {
		cookie, err := http.ParseSetCookie(line)
		require.NoError(p.t, err)
		if cookie.Name == "cp_admin" {
			p.adminCookie = cookie.Name + "=" + cookie.Value
		}
	}
	require.NotEmpty(p.t, p.adminCookie, "a completed admin sign-in sets the admin session")
	return nil
}

func (p *gamer) fromThePage(header http.Header, origin string) {
	header.Set("Origin", origin)
	header.Set("Cookie", p.cookie+"; "+p.adminCookie)
}

func (p *gamer) grantFromThePage(origin string, account string) (*planetv1.GrantChargesResponse, error) {
	p.t.Helper()

	req := connect.NewRequest(&planetv1.GrantChargesRequest{AccountId: account, Bomb: true})
	p.fromThePage(req.Header(), origin)
	res, err := planetv1connect.NewAdminServiceClient(http.DefaultClient, p.stack.adminURL).GrantCharges(p.t.Context(), req)
	if err != nil {
		return nil, err //nolint:wrapcheck // the tests read the connect code.
	}
	return res.Msg, nil
}

func (p *gamer) whoAmIOnThePage() (string, error) {
	p.t.Helper()

	req := connect.NewRequest(&authv1.WhoAmIRequest{})
	p.fromThePage(req.Header(), adminOrigin)
	res, err := p.desk().WhoAmI(p.t.Context(), req)
	if err != nil {
		return "", err //nolint:wrapcheck // the tests read the connect code.
	}
	return res.Msg.GetAccountId(), nil
}

func (p *gamer) adminSignOut() {
	p.t.Helper()

	req := connect.NewRequest(&authv1.AdminSignOutRequest{})
	p.fromThePage(req.Header(), adminOrigin)
	_, err := p.desk().SignOut(p.t.Context(), req)
	require.NoError(p.t, err)
}

func newOperator(t *testing.T, game gameStack) *gamer {
	t.Helper()

	operator := game.newPlayer(t)
	operator.link("the-operator")
	operator.becomeOperator()
	return operator
}

func TestAnOperatorSignsInWithGoogleAndGrantsItselfABomb(t *testing.T) {
	game := startGame(t)
	operator := newOperator(t, game)

	require.NoError(t, operator.adminSignIn("the-operator"))

	account, err := operator.whoAmIOnThePage()
	require.NoError(t, err)
	assert.Equal(t, operator.account(), account, "the admin account is the player account")

	granted, err := operator.grantFromThePage(adminOrigin, account)
	require.NoError(t, err)
	assert.True(t, granted.GetAfter().GetBomb())
}

func TestTheGameSessionNeverOpensTheAdminPage(t *testing.T) {
	game := startGame(t)
	operator := newOperator(t, game)

	_, err := operator.grantFromThePage(adminOrigin, operator.account())

	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err), "cp_sid alone is no admin session")
}

func TestOnlyAnOperatorsGoogleAccountGetsAnAdminSession(t *testing.T) {
	game := startGame(t)
	player := game.newPlayer(t)
	player.link("a-player")
	guest := game.newPlayer(t)
	guest.becomeOperator()

	for name, refusal := range map[string]struct {
		caller  *gamer
		subject string
	}{
		"a player":              {player, "a-player"},
		"a Google nobody holds": {guest, "a-stranger"},
	} {
		err := refusal.caller.adminSignIn(refusal.subject)
		assert.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err), name)
		assert.Empty(t, refusal.caller.adminCookie, name)
	}
}

func TestTheAdminPageAloneMayCall(t *testing.T) {
	game := startGame(t)
	operator := newOperator(t, game)
	require.NoError(t, operator.adminSignIn("the-operator"))

	for _, origin := range []string{"https://clickplanet.lol", "", "https://clips.clickplanet.lol"} {
		_, err := operator.grantFromThePage(origin, operator.account())
		assert.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err), origin)
	}

	start := connect.NewRequest(&authv1.AdminStartSignInRequest{})
	start.Header().Set("Origin", "https://clickplanet.lol")
	_, err := operator.desk().StartSignIn(t.Context(), start)
	assert.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err), "the sign-in too")
}

func TestAnAdminSignOutEndsTheAdminSessionAndKeepsTheGameOne(t *testing.T) {
	game := startGame(t)
	operator := newOperator(t, game)
	require.NoError(t, operator.adminSignIn("the-operator"))

	operator.adminSignOut()

	_, err := operator.whoAmIOnThePage()
	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))

	me := connect.NewRequest(&authv1.GetMeRequest{})
	operator.send(me.Header())
	_, err = authv1connect.NewAuthServiceClient(http.DefaultClient, game.baseURL).GetMe(t.Context(), me)
	require.NoError(t, err, "the game session stays")
}

func TestAGameSignOutKeepsTheAdminSession(t *testing.T) {
	game := startGame(t)
	operator := newOperator(t, game)
	require.NoError(t, operator.adminSignIn("the-operator"))

	operator.signOut()

	_, err := operator.whoAmIOnThePage()
	require.NoError(t, err, "two apps, two sessions")
}

func TestTheConsoleListenerNamesNoOperator(t *testing.T) {
	game := startGame(t)

	_, err := authv1connect.NewAdminServiceClient(http.DefaultClient, game.consoleURL).
		WhoAmI(t.Context(), connect.NewRequest(&authv1.WhoAmIRequest{}))

	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}
