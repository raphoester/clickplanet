package answer_quiz_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	quizzesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/quizzes/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/quizzes/usecases/answer_quiz_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/quizzesv1controller/caller"
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
	req *connect.Request[quizzesv1.AnswerQuizRequest],
) (*connect.Response[quizzesv1.AnswerQuizResponse], error) {
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

	return connect.NewResponse(&quizzesv1.AnswerQuizResponse{
		Correct: out.Correct,
		//nolint:gosec // an index into three choices
		CorrectChoice: uint32(out.CorrectChoice),
	}), nil
}
