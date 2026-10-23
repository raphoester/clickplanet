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

func (p *gamer) clickAsSent(tile uint32, country string) error {
	p.t.Helper()

	req := connect.NewRequest(&planetv1.ClickRequest{TileId: tile, CountryId: country})
	p.send(req.Header())
	_, err := planetv1connect.NewClickServiceClient(http.DefaultClient, p.stack.baseURL).Click(p.t.Context(), req)
	return err //nolint:wrapcheck // the test reads the code.
}

func heldIn(t *testing.T, err error) *planetv1.Allegiance {
	t.Helper()

	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	var refused *connect.Error
	require.ErrorAs(t, err, &refused)
	require.Len(t, refused.Details(), 1)
	detail, err := refused.Details()[0].Value()
	require.NoError(t, err)
	return detail.(*planetv1.Allegiance)
}

func TestAPlayerPlaysOnlyTheFlagItPledged(t *testing.T) {
	game := startGame(t)
	ada := game.newPlayer(t)

	assert.Nil(t, ada.allegiance(), "a new player plays for nobody")
	assert.Empty(t, heldIn(t, ada.clickAsSent(1, "fr")).GetCountryId(), "a click pledges nothing")
	assert.Nil(t, ada.allegiance())

	require.NoError(t, ada.tryPledge("fr"))
	assert.Zero(t, ada.allegiance().GetChangeableAtUnixMs(), "the first flag starts no wait")
	require.NoError(t, ada.clickAsSent(1, "fr"))

	assert.Equal(t, "fr", heldIn(t, ada.clickAsSent(2, "de")).GetCountryId())
	assert.Equal(t, "fr", ada.allegiance().GetCountryId(), "a click changes no flag")

	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(ada.tryPledge("zz")))
	require.NoError(t, game.newPlayer(t).tryClick(4, "de"), "another player is not held to it")
}

func (p *gamer) tryPledge(flag string) error {
	p.t.Helper()

	req := connect.NewRequest(&planetv1.PledgeAllegianceRequest{CountryId: flag})
	p.send(req.Header())
	_, err := planetv1connect.NewClickServiceClient(http.DefaultClient, p.stack.baseURL).PledgeAllegiance(p.t.Context(), req)
	return err //nolint:wrapcheck // the test reads the code.
}

func TestAPlayerChangesItsFlagBeforeItClicks(t *testing.T) {
	game := startGame(t)
	ada := game.newPlayer(t)
	ada.click(1, "fr")

	require.NoError(t, ada.tryPledge("de"))
	changed := ada.allegiance()
	assert.Equal(t, "de", changed.GetCountryId())
	assert.NotZero(t, changed.GetChangeableAtUnixMs(), "the change starts the wait, with no click")

	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(ada.tryPledge("fr")))
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(ada.tryClick(2, "fr")))
	require.NoError(t, ada.tryClick(2, "de"))
}
