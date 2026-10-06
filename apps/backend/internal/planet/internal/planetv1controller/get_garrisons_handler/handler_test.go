package get_garrisons_handler_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/garrisons"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_garrisons_handler"
)

type stubUseCase []garrisons.Garrison

func (s stubUseCase) Execute(context.Context) []garrisons.Garrison { return s }

func TestEveryGarrisonStandingIsAnswered(t *testing.T) {
	res, err := get_garrisons_handler.New(stubUseCase{
		{Tile: 3, Country: "fr", Defenders: 2},
		{Tile: 9, Country: "de", Defenders: 10},
	}).GetGarrisons(t.Context(), connect.NewRequest(&planetv1.GetGarrisonsRequest{}))
	require.NoError(t, err)

	got := res.Msg.GetGarrisons()
	require.Len(t, got, 2)
	assert.Equal(t, uint32(3), got[0].GetTileId())
	assert.Equal(t, "de", got[1].GetCountryId())
	assert.Equal(t, uint32(10), got[1].GetDefenders())
}
