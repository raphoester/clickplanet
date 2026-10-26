package inmemory_tile_holders_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_landmass_handler/landmass_query/inmemory_tile_holders"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

type stubTiles []clicks.Holding

func (s stubTiles) HoldingsOf(clicks.LandmassID) []clicks.Holding {
	return s
}

func TestTheHoldersComeMostTilesFirstAndInAStableOrder(t *testing.T) {
	ada := cpsession.AccountID(uuid.MustParse("0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"))
	bob := cpsession.AccountID(uuid.MustParse("5a0c2f1e-7d3b-4c8a-9e6f-1b2d3c4e5f60"))

	holders := inmemory_tile_holders.New(stubTiles{
		{Account: bob, Flag: "de", Tiles: 3},
		{Account: bob, Flag: "fr", Tiles: 7},
		{Account: ada, Flag: "fr", Tiles: 3},
	}).Holders(1)

	want := []*planetv1.LandmassHolder{
		{AccountId: bob.String(), CountryId: "fr", Tiles: 7},
		{AccountId: ada.String(), CountryId: "fr", Tiles: 3},
		{AccountId: bob.String(), CountryId: "de", Tiles: 3},
	}
	require.Len(t, holders, len(want))
	for i := range want {
		assert.True(t, proto.Equal(want[i], holders[i]), "holder %d is %v", i, holders[i])
	}
}
