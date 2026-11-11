package get_charges_usecase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges/usecases/get_charges_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

type stubCharges map[charges.Holder]charges.Held

func (s stubCharges) Held(_ context.Context, holder charges.Holder) (charges.Held, error) {
	return s[holder], nil
}

func TestTheChargesReadAreTheAccounts(t *testing.T) {
	wallet := stubCharges{
		"a-guest":        {Bomb: true},
		charges.NoHolder: {SpreadClicks: 3},
	}
	ctx := cpctx.AddAccountToContext(cpctx.AddIPToContext(t.Context(), "1.2.3.4"), "a-guest")

	held, err := get_charges_usecase.New(wallet).Execute(ctx)
	require.NoError(t, err)
	assert.Equal(t, charges.Held{Bomb: true}, held)
}

func TestWithNoAccountNothingIsHeld(t *testing.T) {
	wallet := stubCharges{charges.NoHolder: {SpreadClicks: 3}}

	held, err := get_charges_usecase.New(wallet).Execute(cpctx.AddIPToContext(t.Context(), "1.2.3.4"))
	require.NoError(t, err)
	assert.Equal(t, charges.Held{SpreadClicks: 3}, held, "the read goes to NoHolder, which the storage never grants anything")
}
