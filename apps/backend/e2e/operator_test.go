package e2e_test

import (
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1/authv1connect"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
)

func (p *gamer) becomeOperator() {
	p.t.Helper()

	_, err := p.stack.schema(p.t, "auth").ExecContext(p.t.Context(),
		`UPDATE accounts SET operator = true WHERE id = $1`, p.account())
	require.NoError(p.t, err)
}

func (p *gamer) grantFromThePage(origin string, account string) (*planetv1.GrantChargesResponse, error) {
	p.t.Helper()

	req := connect.NewRequest(&planetv1.GrantChargesRequest{AccountId: account, Bomb: true})
	req.Header().Set("Origin", origin)
	if p.cookie != "" {
		req.Header().Set("Cookie", p.cookie)
	}
	res, err := planetv1connect.NewAdminServiceClient(http.DefaultClient, p.stack.adminURL).GrantCharges(p.t.Context(), req)
	if err != nil {
		return nil, err //nolint:wrapcheck // the tests read the connect code.
	}
	return res.Msg, nil
}

func (p *gamer) whoAmIOnThePage() (string, error) {
	p.t.Helper()

	req := connect.NewRequest(&authv1.WhoAmIRequest{})
	req.Header().Set("Origin", adminOrigin)
	req.Header().Set("Cookie", p.cookie)
	res, err := authv1connect.NewAdminServiceClient(http.DefaultClient, p.stack.adminURL).WhoAmI(p.t.Context(), req)
	if err != nil {
		return "", err //nolint:wrapcheck // the tests read the connect code.
	}
	return res.Msg.GetAccountId(), nil
}

func TestTheAdminPageLearnsWhichAccountItIs(t *testing.T) {
	game := startGame(t)
	operator := game.newPlayer(t)
	operator.link("the-operator")
	operator.becomeOperator()

	account, err := operator.whoAmIOnThePage()

	require.NoError(t, err)
	assert.Equal(t, operator.account(), account)
}

func TestTheConsoleListenerNamesNoOperator(t *testing.T) {
	game := startGame(t)

	_, err := authv1connect.NewAdminServiceClient(http.DefaultClient, game.consoleURL).
		WhoAmI(t.Context(), connect.NewRequest(&authv1.WhoAmIRequest{}))

	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

func TestAnOperatorGrantsItselfABombFromTheAdminPage(t *testing.T) {
	game := startGame(t)
	operator := game.newPlayer(t)
	operator.link("the-operator")
	operator.becomeOperator()

	granted, err := operator.grantFromThePage(adminOrigin, operator.account())

	require.NoError(t, err)
	assert.True(t, granted.GetAfter().GetBomb())
}

func TestTheOperatorServicesAnswerNobodyElse(t *testing.T) {
	game := startGame(t)
	operator := game.newPlayer(t)
	operator.link("the-operator")
	operator.becomeOperator()
	player := game.newPlayer(t)
	player.link("a-player")
	guest := game.newPlayer(t)
	guest.becomeOperator()
	stranger := game.newPlayer(t)
	stranger.cookie = ""

	for name, call := range map[string]struct {
		caller *gamer
		origin string
		code   connect.Code
	}{
		"a player":                 {player, adminOrigin, connect.CodePermissionDenied},
		"a guest with the flag":    {guest, adminOrigin, connect.CodePermissionDenied},
		"no cookie":                {stranger, adminOrigin, connect.CodeUnauthenticated},
		"the game page":            {operator, "https://clickplanet.lol", connect.CodePermissionDenied},
		"a script with no origin":  {operator, "", connect.CodePermissionDenied},
		"another page of the site": {operator, "https://clips.clickplanet.lol", connect.CodePermissionDenied},
	} {
		_, err := call.caller.grantFromThePage(call.origin, operator.account())
		assert.Equal(t, call.code, connect.CodeOf(err), name)
	}
}

func TestAnOperatorWhoSignedOutIsNoLongerAdmitted(t *testing.T) {
	game := startGame(t)
	operator := game.newPlayer(t)
	operator.link("the-operator")
	operator.becomeOperator()
	_, err := operator.grantFromThePage(adminOrigin, operator.account())
	require.NoError(t, err)

	operator.signOut()

	_, err = operator.grantFromThePage(adminOrigin, operator.account())
	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}
