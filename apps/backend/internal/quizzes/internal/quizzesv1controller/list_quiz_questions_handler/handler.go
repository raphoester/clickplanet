package list_quiz_questions_handler

import (
	"context"

	"connectrpc.com/connect"

	quizzesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/quizzes/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/quizzesv1controller/list_quiz_questions_handler/quiz_questions_query"
)

type Query interface {
	QuizQuestions(filter quiz_questions_query.Filter) *quizzesv1.ListQuizQuestionsResponse
}

func New(query Query) ListQuizQuestionsHandler {
	return ListQuizQuestionsHandler{query: query}
}

type ListQuizQuestionsHandler struct {
	query Query
}

func (h ListQuizQuestionsHandler) ListQuizQuestions(
	_ context.Context,
	req *connect.Request[quizzesv1.ListQuizQuestionsRequest],
) (*connect.Response[quizzesv1.ListQuizQuestionsResponse], error) {
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
