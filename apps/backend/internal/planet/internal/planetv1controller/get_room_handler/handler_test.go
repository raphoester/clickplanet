package get_room_handler_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_room_handler"
)

const (
	full  = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"
	empty = "7c1d2e3f-4a5b-4c6d-8e7f-9a0b1c2d3e4f"
)

type hands map[charges.Holder]charges.Held

func (h hands) Held(_ context.Context, holder charges.Holder) (charges.Held, error) {
	return h[holder], nil
}

func room(t *testing.T, rules charges.Rules, accounts ...string) []*planetv1.Room {
	t.Helper()

	held := hands{full: {Refill: true, Shields: 30}}
	res, err := get_room_handler.New(held, func() charges.Rules { return rules }).
		GetRoom(t.Context(), connect.NewRequest(&planetv1.GetRoomRequest{AccountIds: accounts}))
	require.NoError(t, err)
	return res.Msg.GetRooms()
}

func TestEachAccountIsToldTheKindsItHasNoRoomFor(t *testing.T) {
	rooms := room(t, charges.TestRules(), full, empty)

	require.Len(t, rooms, 1, "an account with room for every kind is left out")
	assert.Equal(t, full, rooms[0].GetAccountId())
	assert.Equal(t, []planetv1.BonusKind{planetv1.BonusKind_BONUS_KIND_REFILL, planetv1.BonusKind_BONUS_KIND_SHIELDS},
		rooms[0].GetFull())
}

func TestWhileBombsAreOffNoHandHasRoomForOne(t *testing.T) {
	off := charges.TestRules()
	off.Bomb.Off = true

	rooms := room(t, off, empty)

	require.Len(t, rooms, 1)
	assert.Equal(t, []planetv1.BonusKind{planetv1.BonusKind_BONUS_KIND_BOMB}, rooms[0].GetFull())
}

func TestAnIdThatNamesNoAccountIsRefused(t *testing.T) {
	_, err := get_room_handler.New(hands{}, charges.TestRules).
		GetRoom(t.Context(), connect.NewRequest(&planetv1.GetRoomRequest{AccountIds: []string{"a-guest"}}))

	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}
