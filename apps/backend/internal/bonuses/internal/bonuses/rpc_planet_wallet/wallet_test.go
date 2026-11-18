package rpc_planet_wallet_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses/rpc_planet_wallet"
)

type fakePlanet struct {
	granted []*planetv1.InternalGrantChargesRequest
	filled  []*planetv1.FillChargesRequest
}

func (f *fakePlanet) FillCharges(
	_ context.Context, req *connect.Request[planetv1.FillChargesRequest],
) (*connect.Response[planetv1.FillChargesResponse], error) {
	f.filled = append(f.filled, req.Msg)
	return connect.NewResponse(&planetv1.FillChargesResponse{}), nil
}

func (f *fakePlanet) GrantCharges(
	_ context.Context, req *connect.Request[planetv1.InternalGrantChargesRequest],
) (*connect.Response[planetv1.InternalGrantChargesResponse], error) {
	f.granted = append(f.granted, req.Msg)
	return connect.NewResponse(&planetv1.InternalGrantChargesResponse{
		Kept: 2, Charges: &planetv1.ChargesHeld{Shields: 7},
	}), nil
}

var payment = bonuses.Payment{
	Grant: bonuses.OfferGrantOf("a-token"), Account: "acc", Kind: bonuses.KindShields, Amount: 5,
}

func TestAWinIsPaidInPlanetsWordsAndKnownForAWhile(t *testing.T) {
	planet := &fakePlanet{}

	kept, err := rpc_planet_wallet.New(planet, rpc_planet_wallet.ForAWhile).Pay(t.Context(), payment)

	require.NoError(t, err)
	assert.Equal(t, 2, kept, "what planet kept")
	require.Len(t, planet.granted, 1)
	assert.Equal(t, payment.Grant.String(), planet.granted[0].GetGrantId())
	assert.Equal(t, "acc", planet.granted[0].GetAccountId())
	assert.Equal(t, planetv1.BonusKind_BONUS_KIND_SHIELDS, planet.granted[0].GetKind())
	assert.Equal(t, uint32(5), planet.granted[0].GetAmount())
	assert.False(t, planet.granted[0].GetLasting())
}

func TestTheSignInRefillIsKnownForGood(t *testing.T) {
	planet := &fakePlanet{}

	_, err := rpc_planet_wallet.New(planet, rpc_planet_wallet.ForGood).Pay(t.Context(), payment)

	require.NoError(t, err)
	assert.True(t, planet.granted[0].GetLasting())
}

func TestACallerWithNoAccountIsPaidNothingAndPlanetIsNotAsked(t *testing.T) {
	planet := &fakePlanet{}
	wallet := rpc_planet_wallet.New(planet, rpc_planet_wallet.ForAWhile)

	kept, err := wallet.Pay(t.Context(), bonuses.Payment{Grant: payment.Grant, Kind: bonuses.KindRefill, Amount: 1})
	require.NoError(t, err)

	assert.Zero(t, kept)
	assert.Empty(t, planet.granted)
}

var stacks = bonuses.Stacks{
	Grant: bonuses.OfferGrantOf("a-level"), Account: "acc", Kinds: []bonuses.Kind{bonuses.KindRefill, bonuses.KindShields},
}

func TestAFillIsAskedInPlanetsWords(t *testing.T) {
	planet := &fakePlanet{}

	err := rpc_planet_wallet.New(planet, rpc_planet_wallet.ForGood).Fill(t.Context(), stacks)

	require.NoError(t, err)
	require.Len(t, planet.filled, 1)
	assert.Equal(t, stacks.Grant.String(), planet.filled[0].GetGrantId())
	assert.Equal(t, "acc", planet.filled[0].GetAccountId())
	assert.Equal(t, []planetv1.BonusKind{planetv1.BonusKind_BONUS_KIND_REFILL, planetv1.BonusKind_BONUS_KIND_SHIELDS},
		planet.filled[0].GetKinds())
}

func TestAFillForNoAccountAsksPlanetNothing(t *testing.T) {
	planet := &fakePlanet{}

	err := rpc_planet_wallet.New(planet, rpc_planet_wallet.ForGood).Fill(t.Context(),
		bonuses.Stacks{Grant: stacks.Grant, Kinds: stacks.Kinds})

	require.NoError(t, err)
	assert.Empty(t, planet.filled)
}
