package rpc_planet_room_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/boxes/internal/boxes"
	"github.com/raphoester/clickplanet.lol-backend/internal/boxes/internal/boxes/rpc_planet_room"
)

type fakePlanet struct {
	asked   []*planetv1.GetRoomRequest
	rooms   []*planetv1.Room
	failing error
}

func (f *fakePlanet) GetRoom(
	_ context.Context, req *connect.Request[planetv1.GetRoomRequest],
) (*connect.Response[planetv1.GetRoomResponse], error) {
	f.asked = append(f.asked, req.Msg)
	if f.failing != nil {
		return nil, f.failing
	}
	return connect.NewResponse(&planetv1.GetRoomResponse{Rooms: f.rooms}), nil
}

var everyKind = []planetv1.BonusKind{
	planetv1.BonusKind_BONUS_KIND_REFILL, planetv1.BonusKind_BONUS_KIND_SPREAD_CLICKS, planetv1.BonusKind_BONUS_KIND_BOMB,
	planetv1.BonusKind_BONUS_KIND_ENCLOSE_CLICKS, planetv1.BonusKind_BONUS_KIND_SHIELDS,
}

func TestEveryAccountIsAskedInPlanetsWords(t *testing.T) {
	planet := &fakePlanet{}

	room, err := rpc_planet_room.New(planet).Any(t.Context(), []boxes.Account{"acc-1", "acc-2"})

	require.NoError(t, err)
	assert.True(t, room, "planet leaves out an account with room for every kind")
	require.Len(t, planet.asked, 1)
	assert.Equal(t, []string{"acc-1", "acc-2"}, planet.asked[0].GetAccountIds())
}

func TestAHandFullOfEveryKindHasNoRoom(t *testing.T) {
	planet := &fakePlanet{rooms: []*planetv1.Room{{AccountId: "acc-1", Full: everyKind}}}

	room, err := rpc_planet_room.New(planet).Any(t.Context(), []boxes.Account{"acc-1"})

	require.NoError(t, err)
	assert.False(t, room)
}

func TestAHandWithRoomForOneKindHasRoom(t *testing.T) {
	planet := &fakePlanet{rooms: []*planetv1.Room{{AccountId: "acc-1", Full: everyKind[1:]}}}

	room, err := rpc_planet_room.New(planet).Any(t.Context(), []boxes.Account{"acc-1"})

	require.NoError(t, err)
	assert.True(t, room)
}

func TestTheHandsBehindOneCallerAreReadTogether(t *testing.T) {
	planet := &fakePlanet{rooms: []*planetv1.Room{
		{AccountId: "acc-1", Full: everyKind[:3]},
		{AccountId: "acc-2", Full: everyKind[3:]},
	}}

	room, err := rpc_planet_room.New(planet).Any(t.Context(), []boxes.Account{"acc-1", "acc-2"})

	require.NoError(t, err)
	assert.False(t, room, "whichever account claims it, a box must fit its hand")
}

func TestARoomPlanetCannotTellIsAnError(t *testing.T) {
	_, err := rpc_planet_room.New(&fakePlanet{failing: errors.New("planet is gone")}).
		Any(t.Context(), []boxes.Account{"acc-1"})

	require.ErrorContains(t, err, "failed to ask planet the room")
}
