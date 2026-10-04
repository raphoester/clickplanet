package e2e_test

import (
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1/playerv1connect"
)

func (p *gamer) name() string {
	p.t.Helper()

	req := connect.NewRequest(&playerv1.GetProfileRequest{})
	p.send(req.Header())
	res, err := p.players().GetProfile(p.t.Context(), req)
	require.NoError(p.t, err)
	return res.Msg.GetProfile().GetName()
}

func TestASignedInPlayerIsGivenAUsernameItMayChange(t *testing.T) {
	game := startGame(t)
	ada := game.newPlayer(t)
	assert.Empty(t, ada.name(), "a guest has no username")

	ada.link("google-ada")

	require.Eventually(t, func() bool { return generatedName.MatchString(ada.name()) }, 5*time.Second, 20*time.Millisecond)
	_, err := ada.setName("Ada_L")
	require.NoError(t, err)
	assert.Equal(t, "Ada_L", ada.name())
}

func TestAnOperatorNamesTheLinkedAccountsOnTheAdminListenerOnly(t *testing.T) {
	game := startGame(t)
	ada := game.newPlayer(t)
	ada.link("google-ada")
	ada.click(1, "fr")
	require.Eventually(t, func() bool { return ada.stats().GetTilesTaken() == 1 && ada.name() != "" },
		5*time.Second, 20*time.Millisecond)

	request := connect.NewRequest(&playerv1.NameAccountsRequest{})
	_, err := playerv1connect.NewAdminServiceClient(http.DefaultClient, game.baseURL).NameAccounts(t.Context(), request)
	require.Equal(t, connect.CodeUnimplemented, connect.CodeOf(err), "the public router does not serve it")

	res, err := playerv1connect.NewAdminServiceClient(http.DefaultClient, game.adminURL).NameAccounts(t.Context(), request)
	require.NoError(t, err)
	assert.Zero(t, res.Msg.GetNamed(), "the sign-in already named it")
}
