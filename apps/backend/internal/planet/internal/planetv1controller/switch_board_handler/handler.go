package switch_board_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/embedded_board"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/eras"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/eras/usecases/switch_board_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, command switch_board_usecase.Command) (switch_board_usecase.Out, error)
}

func New(useCase UseCase) SwitchBoardHandler {
	return SwitchBoardHandler{useCase: useCase}
}

type SwitchBoardHandler struct {
	useCase UseCase
}

// SwitchBoard runs on a context detached from the request: a caller that gives up never stops a switch halfway, and a
// call made again finds the map on its new board and is refused as a second call.
func (h SwitchBoardHandler) SwitchBoard(
	ctx context.Context, req *connect.Request[planetv1.SwitchBoardRequest],
) (*connect.Response[planetv1.SwitchBoardResponse], error) {
	out, err := h.useCase.Execute(context.WithoutCancel(ctx), switch_board_usecase.Command{
		Map:    clicks.MapID(req.Msg.GetMap()),
		Board:  clicks.BoardName(req.Msg.GetBoard()),
		DryRun: req.Msg.GetDryRun(),
	})
	switch {
	case err == nil:
		return connect.NewResponse(responseOf(out)), nil
	case errors.Is(err, switch_board_usecase.ErrNotTheWorld):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, eras.ErrNotCarried), errors.Is(err, embedded_board.ErrNoCarryover),
		errors.Is(err, clicks.ErrCarryover), errors.Is(err, switch_board_usecase.ErrArenaPlays),
		errors.Is(err, switch_board_usecase.ErrPlaysAlready):
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	default:
		return nil, err
	}
}

func responseOf(out switch_board_usecase.Out) *planetv1.SwitchBoardResponse {
	carried := out.Carried
	response := &planetv1.SwitchBoardResponse{
		From: string(carried.From), To: string(carried.To),
		Tiles: carried.Tiles, Carried: carried.Carried, Blank: carried.Blank,
		HeldBefore: carried.HeldBefore, HeldAfter: carried.HeldAfter,
		Dropped: carried.Dropped, DroppedHeld: carried.DroppedHeld,
		ShieldsBefore: carried.ShieldsBefore, ShieldsAfter: carried.ShieldsAfter,
		MinesMoved: carried.MinesMoved, MinesLost: carried.MinesLost,
		TrapsMoved: carried.TrapsMoved, TrapsLost: carried.TrapsLost,
		VirusesEnded: carried.VirusesEnded, LocksDropped: carried.LocksDropped,
	}
	if !out.SwitchedAt.IsZero() {
		response.SwitchedAt = timestamppb.New(out.SwitchedAt)
	}
	return response
}
