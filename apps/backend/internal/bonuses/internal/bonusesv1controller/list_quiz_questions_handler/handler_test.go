package list_quiz_questions_handler_test

import (
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	bonusesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/bonuses/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonusesv1controller/list_quiz_questions_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonusesv1controller/list_quiz_questions_handler/quiz_questions_query"
)

type query struct {
	asked quiz_questions_query.Filter
}

func (q *query) QuizQuestions(filter quiz_questions_query.Filter) *bonusesv1.ListQuizQuestionsResponse {
	q.asked = filter
	return &bonusesv1.ListQuizQuestionsResponse{Total: 7}
}

func TestTheRequestIsTheFilter(t *testing.T) {
	read := &query{}

	res, err := list_quiz_questions_handler.New(read).ListQuizQuestions(t.Context(),
		connect.NewRequest(&bonusesv1.ListQuizQuestionsRequest{
			Order: bonusesv1.QuizQuestionOrder_QUIZ_QUESTION_ORDER_MEASURED, EasiestFirst: true, MinAnswers: 15,
			Limit: 20, Offset: 40,
		}))
	require.NoError(t, err)

	assert.Equal(t, quiz_questions_query.Filter{
		Order: bonusesv1.QuizQuestionOrder_QUIZ_QUESTION_ORDER_MEASURED, EasiestFirst: true, MinAnswers: 15,
		Limit: 20, Offset: 40,
	}, read.asked)
	assert.Equal(t, uint32(7), res.Msg.GetTotal())
	assert.Equal(t, "no-store", res.Header().Get("Cache-Control"), "the ratings move with every answer")
}
