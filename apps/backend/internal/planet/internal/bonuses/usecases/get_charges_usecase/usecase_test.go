package get_charges_usecase_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/get_charges_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

type stubCharges map[bonuses.Holder]bonuses.Held

func (s stubCharges) Held(holder bonuses.Holder) bonuses.Held { return s[holder] }

func TestTheChargesReadAreTheAccounts(t *testing.T) {
	charges := stubCharges{
		"a-guest":        {Bomb: true},
		bonuses.NoHolder: {SpreadClicks: 3},
	}
	ctx := cpctx.AddAccountToContext(cpctx.AddIPToContext(t.Context(), "1.2.3.4"), "a-guest")

	assert.Equal(t, bonuses.Held{Bomb: true}, get_charges_usecase.New(charges).Execute(ctx))
}

func TestWithNoAccountNothingIsHeld(t *testing.T) {
	charges := stubCharges{bonuses.NoHolder: {SpreadClicks: 3}}

	assert.Equal(t, bonuses.Held{SpreadClicks: 3},
		get_charges_usecase.New(charges).Execute(cpctx.AddIPToContext(t.Context(), "1.2.3.4")),
		"the read goes to NoHolder, which the storage never grants anything")
}
