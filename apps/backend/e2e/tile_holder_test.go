package e2e_test

import (
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1/playerv1connect"
)

func TestAnybodyReadsWhoHoldsATileAndNamesItWithNoToken(t *testing.T) {
	game := startGame(t)
	ada := game.newPlayer(t)
	require.NoError(t, ada.announce("fr"))
	ada.click(1, "fr")

	planet := planetv1connect.NewClickServiceClient(http.DefaultClient, game.baseURL, connect.WithHTTPGet())
	var holder *planetv1.TileHolder
	require.Eventually(t, func() bool {
		res, err := planet.GetTileHolder(t.Context(), connect.NewRequest(&planetv1.GetTileHolderRequest{TileId: 1}))
		if err != nil {
			return false
		}
		holder = res.Msg.GetHolder()
		return holder != nil
	}, 5*time.Second, 20*time.Millisecond, "the ledger is flushed")
	assert.Equal(t, ada.account(), holder.GetAccountId())
	assert.Equal(t, "fr", holder.GetCountryId())

	players := playerv1connect.NewPlayerServiceClient(http.DefaultClient, game.baseURL, connect.WithHTTPGet())
	res, err := players.GetAuthors(t.Context(), connect.NewRequest(&playerv1.GetAuthorsRequest{AccountIds: []string{holder.GetAccountId()}}))

	require.NoError(t, err)
	require.Len(t, res.Msg.GetAuthors(), 1)
	assert.True(t, res.Msg.GetAuthors()[0].GetGuest())
	assert.Regexp(t, `^guest_[0-9a-f]{6}$`, res.Msg.GetAuthors()[0].GetName())
	assert.Nil(t, res.Msg.GetAuthors()[0].GetWornTitle())
}
