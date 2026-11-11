package get_charges_of_handler_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_charges_of_handler"
)

type hands map[charges.Holder]charges.Held

func (h hands) Held(_ context.Context, holder charges.Holder) (charges.Held, error) {
	return h[holder], nil
}

const account = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"

func TestAnAccountsHandIsAnswered(t *testing.T) {
	res, err := get_charges_of_handler.New(hands{account: {Bomb: true, Enclosures: 2}}).
		GetChargesOf(t.Context(), connect.NewRequest(&planetv1.GetChargesOfRequest{AccountId: account}))

	require.NoError(t, err)
	assert.True(t, res.Msg.GetCharges().GetBomb())
	assert.Equal(t, uint32(2), res.Msg.GetCharges().GetEnclosures())
}

func TestAnIdThatNamesNoAccountIsRefused(t *testing.T) {
	_, err := get_charges_of_handler.New(hands{}).
		GetChargesOf(t.Context(), connect.NewRequest(&planetv1.GetChargesOfRequest{AccountId: "nope"}))

	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}
