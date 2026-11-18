package get_country_sizes_handler_test

import (
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_country_sizes_handler"
)

func TestEveryGroundIsAnsweredWithItsTiles(t *testing.T) {
	borders := clicks.BordersOf(10, []uint32{1, 2}, []uint32{5})

	played := clicks.PlayedOn(clicks.NamedGround("earth-1", borders, clicks.NewCapitals(nil)))

	res, err := get_country_sizes_handler.New(played).GetCountrySizes(t.Context(), connect.NewRequest(&planetv1.GetCountrySizesRequest{}))

	require.NoError(t, err)
	assert.True(t, proto.Equal(&planetv1.GetCountrySizesResponse{CountrySizes: []*planetv1.CountrySize{
		{CountryId: "l1", Tiles: 2}, {CountryId: "l2", Tiles: 1},
	}, Board: "earth-1"}, res.Msg), "counted on the board in play, which it names")
}
