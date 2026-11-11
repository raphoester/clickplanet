package rpc_planet_wallet_test

import (
	"context"
	"errors"
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
	asked   []*planetv1.GetRoomRequest
	read    []string
	rooms   []*planetv1.Room
	failing error
}

func (f *fakePlanet) GrantCharges(
	_ context.Context, req *connect.Request[planetv1.InternalGrantChargesRequest],
) (*connect.Response[planetv1.InternalGrantChargesResponse], error) {
	f.granted = append(f.granted, req.Msg)
	if f.failing != nil {
		return nil, f.failing
	}
	return connect.NewResponse(&planetv1.InternalGrantChargesResponse{
		Kept: 2, Charges: &planetv1.ChargesHeld{Shields: 7},
	}), nil
}

func (f *fakePlanet) GetRoom(
	_ context.Context, req *connect.Request[planetv1.GetRoomRequest],
) (*connect.Response[planetv1.GetRoomResponse], error) {
	f.asked = append(f.asked, req.Msg)
	return connect.NewResponse(&planetv1.GetRoomResponse{Rooms: f.rooms}), f.failing
}

func (f *fakePlanet) GetChargesOf(
	_ context.Context, req *connect.Request[planetv1.GetChargesOfRequest],
) (*connect.Response[planetv1.GetChargesOfResponse], error) {
	f.read = append(f.read, req.Msg.GetAccountId())
	return connect.NewResponse(&planetv1.GetChargesOfResponse{Charges: &planetv1.ChargesHeld{Bomb: true}}), nil
}

var payment = bonuses.Payment{
	Grant: bonuses.OfferGrantOf("a-token"), Account: "acc", Kind: bonuses.KindShields, Amount: 5,
}

func TestAnOfferIsPaidInPlanetsWordsAndKnownForAWhile(t *testing.T) {
	planet := &fakePlanet{}

	paid, err := rpc_planet_wallet.New(planet, rpc_planet_wallet.ForAWhile).Pay(t.Context(), payment)

	require.NoError(t, err)
	assert.Equal(t, bonuses.Paid{Kept: 2, Hand: bonuses.Hand{Shields: 7}}, paid)
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

	paid, err := wallet.Pay(t.Context(), bonuses.Payment{Grant: payment.Grant, Kind: bonuses.KindRefill, Amount: 1})
	require.NoError(t, err)
	hand, err := wallet.Hand(t.Context(), bonuses.NoAccount)
	require.NoError(t, err)

	assert.Equal(t, bonuses.Paid{}, paid)
	assert.Equal(t, bonuses.Hand{}, hand)
	assert.Empty(t, planet.granted)
	assert.Empty(t, planet.read)
}

func TestAPaymentPlanetRefusedIsAnError(t *testing.T) {
	planet := &fakePlanet{failing: errors.New("planet is gone")}

	_, err := rpc_planet_wallet.New(planet, rpc_planet_wallet.ForAWhile).Pay(t.Context(), payment)

	require.Error(t, err)
}

func TestTheRoomIsEachAccountsFullKinds(t *testing.T) {
	planet := &fakePlanet{rooms: []*planetv1.Room{
		{AccountId: "a", Full: []planetv1.BonusKind{planetv1.BonusKind_BONUS_KIND_BOMB, planetv1.BonusKind_BONUS_KIND_REFILL}},
	}}

	full, err := rpc_planet_wallet.New(planet, rpc_planet_wallet.ForAWhile).Full(t.Context(), []bonuses.Account{"a", "b"})

	require.NoError(t, err)
	assert.Equal(t, map[bonuses.Account][]bonuses.Kind{"a": {bonuses.KindBomb, bonuses.KindRefill}}, full)
	assert.Equal(t, []string{"a", "b"}, planet.asked[0].GetAccountIds(), "one call for every account")
}

func TestAHandIsReadFromPlanet(t *testing.T) {
	planet := &fakePlanet{}

	hand, err := rpc_planet_wallet.New(planet, rpc_planet_wallet.ForAWhile).Hand(t.Context(), "acc")

	require.NoError(t, err)
	assert.Equal(t, bonuses.Hand{Bomb: true}, hand)
	assert.Equal(t, []string{"acc"}, planet.read)
}
