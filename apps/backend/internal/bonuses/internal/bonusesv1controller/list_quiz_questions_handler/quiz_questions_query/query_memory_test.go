package quiz_questions_query_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	bonusesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/bonuses/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonusesv1controller/list_quiz_questions_handler/quiz_questions_query"
)

type bank []*bonusesv1.QuizQuestion

// Questions answers a fresh copy each time, as the bank does.
func (b bank) Questions() []*bonusesv1.QuizQuestion {
	listed := make([]*bonusesv1.QuizQuestion, 0, len(b))
	for _, question := range b {
		listed = append(listed, proto.CloneOf(question))
	}
	return listed
}

func question(id string, predicted float64, answers, right uint32) *bonusesv1.QuizQuestion {
	listed := &bonusesv1.QuizQuestion{Id: id, PredictedShare: predicted, Answers: answers, Right: right}
	if answers > 0 {
		listed.MeasuredShare = proto.Float64(float64(right) / float64(answers))
	}
	return listed
}

// "trap" looks easy and is measured hard; "unasked" was never answered.
var sample = bank{
	question("fair", 0.70, 20, 14),
	question("trap", 0.62, 30, 9),
	question("unasked", 0.60, 0, 0),
	question("obvious", 0.95, 16, 16),
	question("niche", 0.40, 3, 1),
}

func ids(res *bonusesv1.ListQuizQuestionsResponse) []string {
	listed := make([]string, 0, len(res.GetQuestions()))
	for _, each := range res.GetQuestions() {
		listed = append(listed, each.GetId())
	}
	return listed
}

func TestTheHardestPredictedComeFirstUnlessAsked(t *testing.T) {
	query := quiz_questions_query.NewMemoryQuery(sample)

	assert.Equal(t, []string{"niche", "unasked", "trap", "fair", "obvious"}, ids(query.QuizQuestions(quiz_questions_query.Filter{})))
	assert.Equal(t, []string{"obvious", "fair", "trap", "unasked", "niche"},
		ids(query.QuizQuestions(quiz_questions_query.Filter{EasiestFirst: true})))
}

func TestTheMeasuredShareSortsWithTheNeverAnsweredLast(t *testing.T) {
	query := quiz_questions_query.NewMemoryQuery(sample)
	measured := quiz_questions_query.Filter{Order: bonusesv1.QuizQuestionOrder_QUIZ_QUESTION_ORDER_MEASURED}

	assert.Equal(t, []string{"trap", "niche", "fair", "obvious", "unasked"}, ids(query.QuizQuestions(measured)))

	measured.EasiestFirst = true
	assert.Equal(t, []string{"obvious", "fair", "niche", "trap", "unasked"}, ids(query.QuizQuestions(measured)))
}

func TestAMinimumOfAnswersKeepsTheMeasuredAlone(t *testing.T) {
	res := quiz_questions_query.NewMemoryQuery(sample).QuizQuestions(quiz_questions_query.Filter{
		Order: bonusesv1.QuizQuestionOrder_QUIZ_QUESTION_ORDER_MEASURED, MinAnswers: 15,
	})

	assert.Equal(t, []string{"trap", "fair", "obvious"}, ids(res))
	assert.Equal(t, uint32(3), res.GetTotal())
}

func TestTiesComeByID(t *testing.T) {
	res := quiz_questions_query.NewMemoryQuery(bank{
		question("b", 0.6, 0, 0), question("c", 0.6, 0, 0), question("a", 0.6, 0, 0),
	}).QuizQuestions(quiz_questions_query.Filter{})

	assert.Equal(t, []string{"a", "b", "c"}, ids(res))
}

func TestAPageIsTheSortedQuestionsFromItsOffset(t *testing.T) {
	many := make(bank, 0, 620)
	for i := range 620 {
		many = append(many, question(fmt.Sprintf("q%03d", i), float64(i)/1000, 0, 0))
	}
	query := quiz_questions_query.NewMemoryQuery(many)

	first := query.QuizQuestions(quiz_questions_query.Filter{})
	require.Len(t, first.GetQuestions(), quiz_questions_query.DefaultLimit)
	assert.Equal(t, uint32(620), first.GetTotal())
	assert.Equal(t, "q000", first.GetQuestions()[0].GetId())

	second := query.QuizQuestions(quiz_questions_query.Filter{Limit: 10, Offset: 50})
	assert.Equal(t, "q050", second.GetQuestions()[0].GetId())
	assert.Len(t, second.GetQuestions(), 10)

	assert.Len(t, query.QuizQuestions(quiz_questions_query.Filter{Limit: 10000}).GetQuestions(),
		quiz_questions_query.MaxLimit)
	assert.Empty(t, query.QuizQuestions(quiz_questions_query.Filter{Offset: 700}).GetQuestions())
	assert.Len(t, query.QuizQuestions(quiz_questions_query.Filter{Offset: 600}).GetQuestions(), 20)
}
