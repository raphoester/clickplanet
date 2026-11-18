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
		"a-guest":        {Bombs: 1},
		charges.NoHolder: {SpreadClicks: 3},
	}
	ctx := cpctx.AddAccountToContext(cpctx.AddIPToContext(t.Context(), "1.2.3.4"), "a-guest")

	held, err := get_charges_usecase.New(wallet).Execute(ctx)
	require.NoError(t, err)
	assert.Equal(t, charges.Held{Bombs: 1}, held)
}

func TestWithNoAccountNothingIsHeld(t *testing.T) {
	store := charges.NewMemoryStore()
	store.Keep(charges.NoHolder, charges.Held{SpreadClicks: 3})

	held, err := get_charges_usecase.New(charges.NewCharges(store, charges.TestRules)).
		Execute(cpctx.AddIPToContext(t.Context(), "1.2.3.4"))
	require.NoError(t, err)
	assert.Equal(t, charges.Held{}, held)
}
