// Package open_quiz_handler serves planet.v1.ClickService/OpenQuiz.
package open_quiz_handler

import (
	"context"

	"connectrpc.com/connect"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/open_quiz_usecase"
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
	req *connect.Request[planetv1.OpenQuizRequest],
) (*connect.Response[planetv1.OpenQuizResponse], error) {
	out, err := h.useCase.Execute(ctx, open_quiz_usecase.In{Token: req.Msg.GetToken()})
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, open_quiz_usecase.ErrNoSuchQuiz)
	}

	return connect.NewResponse(&planetv1.OpenQuizResponse{
		Question:       out.Question,
		Choices:        out.Options,
		DeadlineUnixMs: out.Deadline.UnixMilli(),
		AnswerSeconds:  out.Window.Seconds(),
	}), nil
}
