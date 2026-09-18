package get_charges_handler_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_charges_handler"
)

type stubUseCase bonuses.Held

func (s stubUseCase) Execute(context.Context) bonuses.Held { return bonuses.Held(s) }

func TestTheChargesHeldAreAnswered(t *testing.T) {
	res, err := get_charges_handler.New(stubUseCase{Bomb: true, SpreadClicks: 4}).
		GetCharges(t.Context(), connect.NewRequest(&planetv1.GetChargesRequest{}))
	require.NoError(t, err)

	assert.True(t, res.Msg.GetCharges().GetBomb())
	assert.Zero(t, res.Msg.GetCharges().GetEnclosures())
	assert.Equal(t, uint32(4), res.Msg.GetCharges().GetSpreadClicksLeft())
}
