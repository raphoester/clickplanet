package get_shares_handler_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/get_shares_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_shares_handler"
)

type stubUseCase struct{ out get_shares_usecase.Out }

func (s stubUseCase) Execute(context.Context) get_shares_usecase.Out { return s.out }

func TestTheSharesAreAnsweredAsTheyAreHeld(t *testing.T) {
	useCase := stubUseCase{out: get_shares_usecase.Out{
		MapTiles: 262119,
		Holdings: []clicks.Holding{{Country: "dz", Tiles: 40000}, {Country: "fr", Tiles: 39000}},
	}}

	res, err := get_shares_handler.New(useCase).GetShares(t.Context(), connect.NewRequest(&planetv1.GetSharesRequest{}))

	require.NoError(t, err)
	assert.True(t, proto.Equal(&planetv1.GetSharesResponse{
		MapTiles: 262119,
		Countries: []*planetv1.CountryTiles{
			{Country: "dz", Tiles: 40000},
			{Country: "fr", Tiles: 39000},
		},
	}, res.Msg))
}
