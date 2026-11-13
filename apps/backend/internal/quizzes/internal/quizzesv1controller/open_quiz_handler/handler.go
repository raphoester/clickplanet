package open_quiz_handler

import (
	"context"

	"connectrpc.com/connect"
	quizzesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/quizzes/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/quizzes/usecases/open_quiz_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/quizzesv1controller/caller"
)

type UseCase interface {
	Execute(ctx context.Context, in open_quiz_usecase.In) (open_quiz_usecase.Out, error)
}

func New(useCase UseCase) OpenQuizHandler {
	return OpenQuizHandler{useCase: useCase}
}

type OpenQuizHandler struct {
	useCase UseCase
}

func (h OpenQuizHandler) OpenQuiz(
	ctx context.Context,
	req *connect.Request[quizzesv1.OpenQuizRequest],
) (*connect.Response[quizzesv1.OpenQuizResponse], error) {
	out, err := h.useCase.Execute(ctx, open_quiz_usecase.In{Caller: caller.Of(ctx), Token: req.Msg.GetToken()})
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, open_quiz_usecase.ErrNoSuchQuiz)
	}

	return connect.NewResponse(&quizzesv1.OpenQuizResponse{
		Question:       out.Question,
		Choices:        out.Options,
		DeadlineUnixMs: out.Deadline.UnixMilli(),
		AnswerSeconds:  out.Window.Seconds(),
	}), nil
}
