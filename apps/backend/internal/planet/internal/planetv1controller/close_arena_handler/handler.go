package close_arena_handler

import (
	"context"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/arenas"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/arenas/usecases/close_arena_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, arena arenas.ID) (close_arena_usecase.Closed, error)
}

func New(useCase UseCase) CloseArenaHandler {
	return CloseArenaHandler{useCase: useCase}
}

type CloseArenaHandler struct {
	useCase UseCase
}

func (h CloseArenaHandler) CloseArena(
	ctx context.Context,
	req *connect.Request[planetv1.CloseArenaRequest],
) (*connect.Response[planetv1.CloseArenaResponse], error) {
	id, err := arenas.ParseID(req.Msg.GetArenaId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	// A caller that gives up never stops the swap of the map halfway.
	if _, err := h.useCase.Execute(context.WithoutCancel(ctx), id); err != nil {
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}
	return connect.NewResponse(&planetv1.CloseArenaResponse{}), nil
}
