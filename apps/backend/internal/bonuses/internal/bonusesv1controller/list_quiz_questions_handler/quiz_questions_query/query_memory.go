package quiz_questions_query

import (
	"cmp"
	"slices"

	bonusesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/bonuses/v1"
)

const (
	DefaultLimit = 50
	MaxLimit     = 500
)

// Questions is every question of the bank with how hard it is, as the wire writes it.
type Questions interface {
	Questions() []*bonusesv1.QuizQuestion
}

type Filter struct {
	Order        bonusesv1.QuizQuestionOrder
	EasiestFirst bool
	MinAnswers   uint32
	Limit        uint32
	Offset       uint32
}

func NewMemoryQuery(questions Questions) *MemoryQuery {
	return &MemoryQuery{questions: questions}
}

// MemoryQuery reads the bank and the ratings in memory: the draw's own copy, as postgres committed it.
type MemoryQuery struct {
	questions Questions
}

func (q *MemoryQuery) QuizQuestions(filter Filter) *bonusesv1.ListQuizQuestionsResponse {
	kept := slices.DeleteFunc(q.questions.Questions(), func(question *bonusesv1.QuizQuestion) bool {
		return question.GetAnswers() < filter.MinAnswers
	})

	slices.SortStableFunc(kept, func(a, b *bonusesv1.QuizQuestion) int {
		return cmp.Or(compare(a, b, filter), cmp.Compare(a.GetId(), b.GetId()))
	})

	//nolint:gosec // the bank holds a few thousand questions.
	total := uint32(len(kept))
	from := min(int(filter.Offset), len(kept))
	to := min(from+int(limitOf(filter.Limit)), len(kept))

	return &bonusesv1.ListQuizQuestionsResponse{Questions: kept[from:to], Total: total}
}

// compare puts the hardest first, the lowest share, unless the easiest are asked first; a question never answered
// comes last by its measured share either way.
func compare(a, b *bonusesv1.QuizQuestion, filter Filter) int {
	direction := 1
	if filter.EasiestFirst {
		direction = -1
	}

	if filter.Order != bonusesv1.QuizQuestionOrder_QUIZ_QUESTION_ORDER_MEASURED {
		return direction * cmp.Compare(a.GetPredictedShare(), b.GetPredictedShare())
	}

	switch {
	case a.MeasuredShare == nil && b.MeasuredShare == nil:
		return 0
	case a.MeasuredShare == nil:
		return 1
	case b.MeasuredShare == nil:
		return -1
	}
	return direction * cmp.Compare(a.GetMeasuredShare(), b.GetMeasuredShare())
}

func limitOf(asked uint32) uint32 {
	switch {
	case asked == 0:
		return DefaultLimit
	case asked > MaxLimit:
		return MaxLimit
	default:
		return asked
	}
}
