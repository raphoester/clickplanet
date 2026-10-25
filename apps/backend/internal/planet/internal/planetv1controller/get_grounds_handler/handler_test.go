package get_grounds_handler_test

import (
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_grounds_handler"
)

func TestEveryGroundIsAnsweredWithItsTiles(t *testing.T) {
	borders := clicks.BordersOf(10, []uint32{1, 2}, []uint32{5})

	res, err := get_grounds_handler.New(borders).GetGrounds(t.Context(), connect.NewRequest(&planetv1.GetGroundsRequest{}))

	require.NoError(t, err)
	assert.True(t, proto.Equal(&planetv1.GetGroundsResponse{Grounds: []*planetv1.Ground{
		{CountryId: "l1", Tiles: 2}, {CountryId: "l2", Tiles: 1},
	}}, res.Msg))
}
