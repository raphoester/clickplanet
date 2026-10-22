package approve_connection_handler

import (
	"context"

	"connectrpc.com/connect"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/admincaller"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/describe_connection_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/connections/usecases/approve_connection_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, in approve_connection_usecase.In) (approve_connection_usecase.Out, error)
}

func New(operators admincaller.Operators, useCase UseCase) ApproveConnectionHandler {
	return ApproveConnectionHandler{operators: operators, useCase: useCase}
}

type ApproveConnectionHandler struct {
	operators admincaller.Operators
	useCase   UseCase
}

func (h ApproveConnectionHandler) ApproveConnection(
	ctx context.Context,
	req *connect.Request[authv1.ApproveConnectionRequest],
) (*connect.Response[authv1.ApproveConnectionResponse], error) {
	operator, err := admincaller.AccountOf(ctx, h.operators, req.Header())
	if err != nil {
		return nil, err //nolint:wrapcheck // admincaller answers the code.
	}

	out, err := h.useCase.Execute(ctx, approve_connection_usecase.In{Request: req.Msg.GetRequest(), Operator: operator})
	if err := describe_connection_handler.Refusal(err); err != nil {
		return nil, err //nolint:wrapcheck // Refusal answers the code.
	}

	res := connect.NewResponse(&authv1.ApproveConnectionResponse{RedirectUrl: out.RedirectURL})
	res.Header().Set("Cache-Control", "no-store")
	return res, nil
}
