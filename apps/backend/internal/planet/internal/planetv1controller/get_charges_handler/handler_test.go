package get_charges_handler_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_charges_handler"
)

type stubUseCase charges.Held

func (s stubUseCase) Execute(context.Context) (charges.Held, error) { return charges.Held(s), nil }

func TestTheChargesHeldAreAnswered(t *testing.T) {
	res, err := get_charges_handler.New(stubUseCase{Bombs: 1, SpreadClicks: 4}).
		GetCharges(t.Context(), connect.NewRequest(&planetv1.GetChargesRequest{}))
	require.NoError(t, err)

	assert.Equal(t, uint32(1), res.Msg.GetCharges().GetBombs())
	assert.Zero(t, res.Msg.GetCharges().GetEnclosures())
	assert.Equal(t, uint32(4), res.Msg.GetCharges().GetSpreadClicksLeft())
}
