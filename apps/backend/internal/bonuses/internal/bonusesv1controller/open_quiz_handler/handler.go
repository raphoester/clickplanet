package open_quiz_handler

import (
	"context"

	"connectrpc.com/connect"
	bonusesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/bonuses/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses/usecases/open_quiz_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonusesv1controller/caller"
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
	req *connect.Request[bonusesv1.OpenQuizRequest],
) (*connect.Response[bonusesv1.OpenQuizResponse], error) {
	out, err := h.useCase.Execute(ctx, open_quiz_usecase.In{Caller: caller.Of(ctx), Token: req.Msg.GetToken()})
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, open_quiz_usecase.ErrNoSuchQuiz)
	}

	return connect.NewResponse(&bonusesv1.OpenQuizResponse{
		Question:       out.Question,
		Choices:        out.Options,
		DeadlineUnixMs: out.Deadline.UnixMilli(),
		AnswerSeconds:  out.Window.Seconds(),
	}), nil
}
