package close_book_handler

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books/usecases/close_book_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, command close_book_usecase.Command) (books.Ending, error)
}

func New(useCase UseCase) CloseBookHandler {
	return CloseBookHandler{useCase: useCase}
}

type CloseBookHandler struct {
	useCase UseCase
}

func (h CloseBookHandler) CloseBook(
	ctx context.Context,
	req *connect.Request[scoringv1.CloseBookRequest],
) (*connect.Response[scoringv1.CloseBookResponse], error) {
	command, err := commandOf(req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	ending, err := h.useCase.Execute(ctx, command)
	if err != nil {
		return nil, err //nolint:wrapcheck // a storage failure is the error net's to answer.
	}
	if ending.Missed {
		return connect.NewResponse(&scoringv1.CloseBookResponse{
			Outcome: &scoringv1.CloseBookResponse_MissedAt{MissedAt: timestamppb.New(ending.At)},
		}), nil
	}
	return connect.NewResponse(&scoringv1.CloseBookResponse{
		Outcome: &scoringv1.CloseBookResponse_ClosedAt{ClosedAt: timestamppb.New(ending.At)},
	}), nil
}

func commandOf(req *scoringv1.CloseBookRequest) (close_book_usecase.Command, error) {
	book, err := books.ParseBookID(req.GetBookId())
	if err != nil {
		return close_book_usecase.Command{}, err //nolint:wrapcheck // the id says why.
	}
	if err := req.GetEndsAt().CheckValid(); err != nil {
		return close_book_usecase.Command{}, fmt.Errorf("book %s has no end: %w", book, err)
	}
	var tiebreak books.RoundID
	if req.GetTiebreakRoundId() != "" {
		if tiebreak, err = books.ParseRoundID(req.GetTiebreakRoundId()); err != nil {
			return close_book_usecase.Command{}, err //nolint:wrapcheck // the id says why.
		}
	}
	return close_book_usecase.Command{Book: book, EndsAt: req.GetEndsAt().AsTime(), Tiebreak: tiebreak}, nil
}
