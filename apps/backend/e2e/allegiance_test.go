package e2e_test

import (
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
)

func (p *gamer) allegiance() *planetv1.Allegiance {
	p.t.Helper()

	req := connect.NewRequest(&planetv1.GetAllegianceRequest{})
	p.send(req.Header())
	res, err := planetv1connect.NewClickServiceClient(http.DefaultClient, p.stack.baseURL).GetAllegiance(p.t.Context(), req)
	require.NoError(p.t, err)

	return res.Msg.GetAllegiance()
}

func TestAPlayerChangesItsFlagOnceAndIsThenHeldToIt(t *testing.T) {
	game := startGame(t)
	ada := game.newPlayer(t)

	assert.Nil(t, ada.allegiance(), "a player that never clicked plays for nobody")

	ada.click(1, "fr")
	assert.Zero(t, ada.allegiance().GetChangeableAtUnixMs(), "the first flag may change at once")

	ada.click(2, "de")
	changed := ada.allegiance()
	assert.Equal(t, "de", changed.GetCountryId())
	assert.NotZero(t, changed.GetChangeableAtUnixMs())

	err := ada.tryClick(3, "fr")
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	var refused *connect.Error
	require.ErrorAs(t, err, &refused)
	require.Len(t, refused.Details(), 1)
	detail, err := refused.Details()[0].Value()
	require.NoError(t, err)
	assert.Equal(t, "de", detail.(*planetv1.Allegiance).GetCountryId())

	require.NoError(t, ada.tryClick(3, "de"), "the flag held still plays")
	require.NoError(t, game.newPlayer(t).tryClick(4, "fr"), "another player is not held to it")
}
