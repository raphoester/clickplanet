package list_quiz_questions_handler

import (
	"context"

	"connectrpc.com/connect"

	bonusesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/bonuses/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonusesv1controller/list_quiz_questions_handler/quiz_questions_query"
)

type Query interface {
	QuizQuestions(filter quiz_questions_query.Filter) *bonusesv1.ListQuizQuestionsResponse
}

func New(query Query) ListQuizQuestionsHandler {
	return ListQuizQuestionsHandler{query: query}
}

type ListQuizQuestionsHandler struct {
	query Query
}

func (h ListQuizQuestionsHandler) ListQuizQuestions(
	_ context.Context,
	req *connect.Request[bonusesv1.ListQuizQuestionsRequest],
) (*connect.Response[bonusesv1.ListQuizQuestionsResponse], error) {
	res := connect.NewResponse(h.query.QuizQuestions(quiz_questions_query.Filter{
		Order:        req.Msg.GetOrder(),
		EasiestFirst: req.Msg.GetEasiestFirst(),
		MinAnswers:   req.Msg.GetMinAnswers(),
		Limit:        req.Msg.GetLimit(),
		Offset:       req.Msg.GetOffset(),
	}))
	res.Header().Set("Cache-Control", "no-store")
	return res, nil
}
