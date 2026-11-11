package answer_quiz_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	bonusesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/bonuses/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses/usecases/answer_quiz_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonusesv1controller/caller"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/planetwords"
)

type UseCase interface {
	Execute(ctx context.Context, in answer_quiz_usecase.In) (answer_quiz_usecase.Out, error)
}

func New(useCase UseCase) AnswerQuizHandler {
	return AnswerQuizHandler{useCase: useCase}
}

type AnswerQuizHandler struct {
	useCase UseCase
}

func (h AnswerQuizHandler) AnswerQuiz(
	ctx context.Context,
	req *connect.Request[bonusesv1.AnswerQuizRequest],
) (*connect.Response[bonusesv1.AnswerQuizResponse], error) {
	out, err := h.useCase.Execute(ctx, answer_quiz_usecase.In{
		Caller:    caller.Of(ctx),
		Token:     req.Msg.GetToken(),
		Choice:    int(req.Msg.GetChoice()),
		CountryID: req.Msg.GetCountryId(),
	})
	if errors.Is(err, answer_quiz_usecase.ErrNoSuchQuiz) {
		return nil, connect.NewError(connect.CodeNotFound, answer_quiz_usecase.ErrNoSuchQuiz)
	}
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&bonusesv1.AnswerQuizResponse{
		Correct: out.Correct,
		//nolint:gosec // an index into three choices
		CorrectChoice: uint32(out.CorrectChoice),
		Kind:          planetwords.Kind(out.Kind),
		//nolint:gosec // a charge count, never negative
		Amount:  uint32(out.Amount),
		Charges: planetwords.Hand(out.Hand),
	}), nil
}
