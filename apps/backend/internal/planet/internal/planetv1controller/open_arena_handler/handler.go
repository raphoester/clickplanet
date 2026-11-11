package open_arena_handler

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/arenas"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/arenas/usecases/open_arena_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
)

type UseCase interface {
	Execute(ctx context.Context, command open_arena_usecase.Command) (bool, error)
}

var ErrNoEnd = errors.New("an arena opens for a battle that ends")

func New(useCase UseCase) OpenArenaHandler {
	return OpenArenaHandler{useCase: useCase}
}

type OpenArenaHandler struct {
	useCase UseCase
}

func (h OpenArenaHandler) OpenArena(
	ctx context.Context,
	req *connect.Request[planetv1.OpenArenaRequest],
) (*connect.Response[planetv1.OpenArenaResponse], error) {
	id, err := arenas.ParseID(req.Msg.GetArenaId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if req.Msg.GetEndsAt() == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("%w: arena %s has no end", ErrNoEnd, id))
	}
	// A caller that gives up never stops the swap of the map halfway.
	_, err = h.useCase.Execute(context.WithoutCancel(ctx), open_arena_usecase.Command{
		Arena: id, EndsAt: req.Msg.GetEndsAt().AsTime(), Set: cprules.SetName(req.Msg.GetRuleSet()),
	})
	switch {
	case errors.Is(err, arenas.ErrRefused):
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	case err != nil:
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}
	return connect.NewResponse(&planetv1.OpenArenaResponse{}), nil
}
