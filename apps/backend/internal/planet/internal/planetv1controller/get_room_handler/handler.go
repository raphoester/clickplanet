package get_room_handler

import (
	"context"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
)

type Hands interface {
	Held(ctx context.Context, holder charges.Holder) (charges.Held, error)
}

func New(hands Hands, rules func() charges.Rules) GetRoomHandler {
	return GetRoomHandler{hands: hands, rules: rules}
}

// GetRoomHandler answers, for each account asked, the kinds its hand has no room for under the rules in force: a
// stack at its size, a charge held where one is the most, and a bomb while bombs are off. The sizes never leave planet.
type GetRoomHandler struct {
	hands Hands
	rules func() charges.Rules
}

func (h GetRoomHandler) GetRoom(
	ctx context.Context, req *connect.Request[planetv1.GetRoomRequest],
) (*connect.Response[planetv1.GetRoomResponse], error) {
	rules := h.rules()

	var rooms []*planetv1.Room
	for _, account := range req.Msg.GetAccountIds() {
		holder, err := charges.ParseHolder(account)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		held, err := h.hands.Held(ctx, holder)
		if err != nil {
			return nil, err //nolint:wrapcheck // the wallet named it.
		}
		full := held.Full(rules)
		if len(full) == 0 {
			continue
		}
		room := &planetv1.Room{AccountId: account}
		for _, kind := range full {
			room.Full = append(room.Full, charges.EncodeKind(kind))
		}
		rooms = append(rooms, room)
	}

	return connect.NewResponse(&planetv1.GetRoomResponse{Rooms: rooms}), nil
}
