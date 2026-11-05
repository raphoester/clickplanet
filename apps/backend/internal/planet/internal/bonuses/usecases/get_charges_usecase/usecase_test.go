package get_charges_usecase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/get_charges_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

type stubCharges map[bonuses.Holder]bonuses.Held

func (s stubCharges) Held(_ context.Context, holder bonuses.Holder) (bonuses.Held, error) {
	return s[holder], nil
}

func TestTheChargesReadAreTheAccounts(t *testing.T) {
	charges := stubCharges{
		"a-guest":        {Bomb: true},
		bonuses.NoHolder: {SpreadClicks: 3},
	}
	ctx := cpctx.AddAccountToContext(cpctx.AddIPToContext(t.Context(), "1.2.3.4"), "a-guest")

	held, err := get_charges_usecase.New(charges).Execute(ctx)
	require.NoError(t, err)
	assert.Equal(t, bonuses.Held{Bomb: true}, held)
}

func TestWithNoAccountNothingIsHeld(t *testing.T) {
	charges := stubCharges{bonuses.NoHolder: {SpreadClicks: 3}}

	held, err := get_charges_usecase.New(charges).Execute(cpctx.AddIPToContext(t.Context(), "1.2.3.4"))
	require.NoError(t, err)
	assert.Equal(t, bonuses.Held{SpreadClicks: 3}, held, "the read goes to NoHolder, which the storage never grants anything")
}
