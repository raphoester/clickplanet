package describe_connection_handler

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/connections"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/connections/usecases/describe_connection_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, sealed string) (describe_connection_usecase.Out, error)
}

func New(useCase UseCase) DescribeConnectionHandler {
	return DescribeConnectionHandler{useCase: useCase}
}

type DescribeConnectionHandler struct {
	useCase UseCase
}

func (h DescribeConnectionHandler) DescribeConnection(
	ctx context.Context,
	req *connect.Request[authv1.DescribeConnectionRequest],
) (*connect.Response[authv1.DescribeConnectionResponse], error) {
	out, err := h.useCase.Execute(ctx, req.Msg.GetRequest())
	if err := Refusal(err); err != nil {
		return nil, err
	}

	res := connect.NewResponse(&authv1.DescribeConnectionResponse{
		ClientName: out.Client, RedirectHost: out.RedirectHost, DenyUrl: out.DeniedURL,
	})
	res.Header().Set("Cache-Control", "no-store")
	return res, nil
}

// For both halves of the consent page: a request past its time is to start again from the app, and one this server
// did not seal is nothing to start from.
func Refusal(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, connections.ErrExpired):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.Is(err, connections.ErrInvalidRequest):
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
	return fmt.Errorf("failed to read the connection: %w", err)
}
