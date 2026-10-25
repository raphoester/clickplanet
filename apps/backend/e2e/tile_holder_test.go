package e2e_test

import (
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1/authv1connect"
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
	res, err := players.GetPlayer(t.Context(), connect.NewRequest(&playerv1.GetPlayerRequest{AccountId: holder.GetAccountId()}))

	require.NoError(t, err)
	assert.True(t, res.Msg.GetPlayer().GetGuest())
	assert.Regexp(t, `^guest_[0-9a-f]{6}$`, res.Msg.GetPlayer().GetName())
	assert.Nil(t, res.Msg.GetPlayer().GetWornTitle())
	assert.Nil(t, res.Msg.GetPlayer().GetStats())
}

func TestATileTakenByADeletedAccountStillHasAHolderWithNoAccount(t *testing.T) {
	game := startGame(t)
	ada := game.newPlayer(t)
	ada.click(1, "fr")

	deletion := connect.NewRequest(&authv1.DeleteAccountRequest{})
	ada.send(deletion.Header())
	_, err := authv1connect.NewAuthServiceClient(http.DefaultClient, game.baseURL).DeleteAccount(t.Context(), deletion)
	require.NoError(t, err)

	planet := planetv1connect.NewClickServiceClient(http.DefaultClient, game.baseURL, connect.WithHTTPGet())
	var holder *planetv1.TileHolder
	require.Eventually(t, func() bool {
		res, err := planet.GetTileHolder(t.Context(), connect.NewRequest(&planetv1.GetTileHolderRequest{TileId: 1}))
		if err != nil {
			return false
		}
		holder = res.Msg.GetHolder()
		return holder != nil && holder.GetAccountId() == ""
	}, 5*time.Second, 20*time.Millisecond, "the take is kept, and names the account no more")
	assert.Equal(t, "fr", holder.GetCountryId())
	assert.NotZero(t, holder.GetTakenAtUnixMs())
}
