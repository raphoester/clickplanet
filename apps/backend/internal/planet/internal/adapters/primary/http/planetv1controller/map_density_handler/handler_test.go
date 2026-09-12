package map_density_handler_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/adapters/primary/http/planetv1controller/map_density_handler"
)

type stubUseCase uint32

func (s stubUseCase) Execute(context.Context) uint32 { return uint32(s) }

func TestMapDensityMapsTheAnswer(t *testing.T) {
	res, err := map_density_handler.New(stubUseCase(257_948)).
		MapDensity(t.Context(), connect.NewRequest(&planetv1.MapDensityRequest{}))

	require.NoError(t, err)
	require.Equal(t, uint32(257_948), res.Msg.GetDensity())
}
