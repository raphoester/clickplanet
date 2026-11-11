package open_round_handler

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books/usecases/open_round_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, command open_round_usecase.Command) (bool, error)
}

func New(useCase UseCase) OpenRoundHandler {
	return OpenRoundHandler{useCase: useCase}
}

type OpenRoundHandler struct {
	useCase UseCase
}

func (h OpenRoundHandler) OpenRound(
	ctx context.Context,
	req *connect.Request[scoringv1.OpenRoundRequest],
) (*connect.Response[scoringv1.OpenRoundResponse], error) {
	command, err := commandOf(req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	_, err = h.useCase.Execute(ctx, command)
	if errors.Is(err, books.ErrInvalidTimes) {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if errors.Is(err, books.ErrRefused) {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	if err != nil {
		return nil, err //nolint:wrapcheck // a storage failure is the error net's to answer.
	}
	return connect.NewResponse(&scoringv1.OpenRoundResponse{}), nil
}

func commandOf(req *scoringv1.OpenRoundRequest) (open_round_usecase.Command, error) {
	round, err := books.ParseRoundID(req.GetRoundId())
	if err != nil {
		return open_round_usecase.Command{}, err //nolint:wrapcheck // the id says why.
	}
	book, err := books.ParseBookID(req.GetBookId())
	if err != nil {
		return open_round_usecase.Command{}, err //nolint:wrapcheck // the id says why.
	}
	if err := req.GetStartsAt().CheckValid(); err != nil {
		return open_round_usecase.Command{}, fmt.Errorf("round %s has no start: %w", round, err)
	}
	ranking, err := rankingOf(req.GetRanking())
	if err != nil {
		return open_round_usecase.Command{}, err
	}
	return open_round_usecase.Command{
		Round: round, Book: book, Times: req.GetTimes(), Ranking: ranking, StartsAt: req.GetStartsAt().AsTime(),
		FromNothing: req.GetFromNothing(),
	}, nil
}

func rankingOf(ranking scoringv1.Ranking) (books.Ranking, error) {
	switch ranking {
	case scoringv1.Ranking_RANKING_GAIN:
		return books.GainRanking{}, nil
	case scoringv1.Ranking_RANKING_HELD_AND_GAIN:
		return books.HeldAndGainRanking{}, nil
	case scoringv1.Ranking_RANKING_UNSPECIFIED:
	}
	return nil, fmt.Errorf("%w: %s", books.ErrUnknownRanking, ranking)
}
