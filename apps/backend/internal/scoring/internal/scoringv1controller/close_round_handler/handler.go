package close_round_handler

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books/usecases/close_round_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, command close_round_usecase.Command) (books.Ending, error)
}

func New(useCase UseCase) CloseRoundHandler {
	return CloseRoundHandler{useCase: useCase}
}

type CloseRoundHandler struct {
	useCase UseCase
}

func (h CloseRoundHandler) CloseRound(
	ctx context.Context,
	req *connect.Request[scoringv1.CloseRoundRequest],
) (*connect.Response[scoringv1.CloseRoundResponse], error) {
	round, err := books.ParseRoundID(req.Msg.GetRoundId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := req.Msg.GetEndsAt().CheckValid(); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("round %s has no end: %w", round, err))
	}
	ending, err := h.useCase.Execute(ctx, close_round_usecase.Command{Round: round, EndsAt: req.Msg.GetEndsAt().AsTime()})
	if err != nil {
		return nil, err //nolint:wrapcheck // a storage failure is the error net's to answer.
	}
	if ending.Missed {
		return connect.NewResponse(&scoringv1.CloseRoundResponse{
			Outcome: &scoringv1.CloseRoundResponse_MissedAt{MissedAt: timestamppb.New(ending.At)},
		}), nil
	}
	return connect.NewResponse(&scoringv1.CloseRoundResponse{
		Outcome: &scoringv1.CloseRoundResponse_ScoredAt{ScoredAt: timestamppb.New(ending.At)},
	}), nil
}
