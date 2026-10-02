// Package answer_quiz_handler serves planet.v1.ClickService/AnswerQuiz.
package answer_quiz_handler

import (
	"context"

	"connectrpc.com/connect"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/answer_quiz_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/chargesheld"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/claim_bonus_handler"
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
	req *connect.Request[planetv1.AnswerQuizRequest],
) (*connect.Response[planetv1.AnswerQuizResponse], error) {
	out, err := h.useCase.Execute(ctx, answer_quiz_usecase.In{
		Token: req.Msg.GetToken(),
		// A choice past the end of three is a guess, not an argument fault: the registry reads it
		// as wrong, which is what it is.
		Choice:    int(req.Msg.GetChoice()),
		CountryID: req.Msg.GetCountryId(),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, answer_quiz_usecase.ErrNoSuchQuiz)
	}

	return connect.NewResponse(&planetv1.AnswerQuizResponse{
		Correct: out.Correct,
		//nolint:gosec // an index into three choices
		CorrectChoice: uint32(out.CorrectChoice),
		// The same encoding a caught box's kind goes out under; unspecified when the answer was
		// wrong, which is the kind a client draws nothing for.
		Kind: claim_bonus_handler.EncodeKind(out.Kind),
		//nolint:gosec // a charge count, never negative
		Amount:  uint32(out.Amount),
		Charges: chargesheld.Encode(out.Held),
	}), nil
}
