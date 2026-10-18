package e2e_test

import (
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
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
