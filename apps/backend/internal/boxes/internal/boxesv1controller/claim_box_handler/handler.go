package claim_box_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	boxesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/boxes/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/boxes/internal/boxes/usecases/claim_box_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/boxes/internal/boxesv1controller/caller"
)

type UseCase interface {
	Execute(ctx context.Context, in claim_box_usecase.In) error
}

func New(useCase UseCase) ClaimBoxHandler {
	return ClaimBoxHandler{useCase: useCase}
}

type ClaimBoxHandler struct {
	useCase UseCase
}

// ClaimBox answers nothing of what the box holds: the bonuses module draws it, and planet's stream tells the grant.
func (h ClaimBoxHandler) ClaimBox(
	ctx context.Context,
	req *connect.Request[boxesv1.ClaimBoxRequest],
) (*connect.Response[boxesv1.ClaimBoxResponse], error) {
	err := h.useCase.Execute(ctx, claim_box_usecase.In{Caller: caller.Of(ctx), Token: req.Msg.GetToken()})
	if errors.Is(err, claim_box_usecase.ErrNoSuchBox) {
		return nil, connect.NewError(connect.CodeNotFound, claim_box_usecase.ErrNoSuchBox)
	}
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&boxesv1.ClaimBoxResponse{}), nil
}
