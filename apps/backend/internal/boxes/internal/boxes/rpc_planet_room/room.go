package rpc_planet_room

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/boxes/internal/boxes"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

type Planet interface {
	GetRoom(context.Context, *connect.Request[planetv1.GetRoomRequest]) (*connect.Response[planetv1.GetRoomResponse], error)
}

func New(planet Planet) Room {
	return Room{planet: planet}
}

// Room answers the boxes' room with planet's GetRoom: the kinds each hand has no room for, read together, against every
// kind planet names.
type Room struct {
	planet Planet
}

var _ boxes.Room = Room{}

func (r Room) Any(ctx context.Context, accounts []boxes.Account) (bool, error) {
	ids := make([]string, 0, len(accounts))
	for _, account := range accounts {
		ids = append(ids, string(account))
	}

	res, err := r.planet.GetRoom(ctx, connect.NewRequest(&planetv1.GetRoomRequest{AccountIds: ids}))
	if err != nil {
		return false, fmt.Errorf("failed to ask planet the room of %d accounts: %w", len(accounts), err)
	}

	full := cpcolls.NewSet[planetv1.BonusKind]()
	for _, room := range res.Msg.GetRooms() {
		full.Add(room.GetFull()...)
	}
	for _, kind := range kinds() {
		if !full.Contains(kind) {
			return true, nil
		}
	}
	return false, nil
}

// kinds is every kind of charge planet names.
func kinds() []planetv1.BonusKind {
	values := planetv1.BonusKind_BONUS_KIND_UNSPECIFIED.Descriptor().Values()
	named := make([]planetv1.BonusKind, 0, values.Len())
	for i := range values.Len() {
		if kind := planetv1.BonusKind(values.Get(i).Number()); kind != planetv1.BonusKind_BONUS_KIND_UNSPECIFIED {
			named = append(named, kind)
		}
	}
	return named
}
