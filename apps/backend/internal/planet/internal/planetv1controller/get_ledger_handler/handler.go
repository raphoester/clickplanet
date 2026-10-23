package get_ledger_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/usecases/excerpt_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/planetmessage"
)

type UseCase interface {
	Execute(ctx context.Context, in excerpt_usecase.In) (excerpt_usecase.Out, error)
}

func New(useCase UseCase) GetLedgerHandler {
	return GetLedgerHandler{useCase: useCase}
}

type GetLedgerHandler struct {
	useCase UseCase
}

func (h GetLedgerHandler) GetLedger(
	ctx context.Context,
	req *connect.Request[planetv1.GetLedgerRequest],
) (*connect.Response[planetv1.GetLedgerResponse], error) {
	var in excerpt_usecase.In
	if since := req.Msg.GetSince(); since != nil {
		in.Since = since.AsTime()
	}
	if until := req.Msg.GetUntil(); until != nil {
		in.Until = until.AsTime()
	}

	out, err := h.useCase.Execute(ctx, in)
	if errors.Is(err, excerpt_usecase.ErrInvalidWindow) {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err != nil {
		return nil, err
	}

	changes := make([]*planetv1.LedgerChange, len(out.Lines))
	for i, line := range out.Lines {
		changes[i] = &planetv1.LedgerChange{
			At:        timestamppb.New(line.At),
			Update:    planetmessage.TileUpdate(line.Update).GetTileUpdate(),
			AccountId: line.By.Account,
			Scope:     line.By.Scope,
		}
	}

	return connect.NewResponse(&planetv1.GetLedgerResponse{
		Since:   timestamppb.New(out.Since),
		Until:   timestamppb.New(out.Until),
		Changes: changes,
	}), nil
}
