package inmemory_quiz_questions

import (
	quizzesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/quizzes/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/bank"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/quizzesv1controller/list_quiz_questions_handler/quiz_questions_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/ratings"
)

// Bank is the questions and each one's rating now: what its answers moved it to, else its prior.
type Bank interface {
	Questions() []bank.Question
	Rating(question ratings.QuestionID) ratings.Rating
}

func New(bank Bank) *Questions {
	return &Questions{bank: bank}
}

// Questions reads the bank for the operator's list, the answers in it: it is served on the admin listener alone.
type Questions struct {
	bank Bank
}

var _ quiz_questions_query.Questions = (*Questions)(nil)

func (q *Questions) Questions() []*quizzesv1.QuizQuestion {
	questions := q.bank.Questions()

	listed := make([]*quizzesv1.QuizQuestion, 0, len(questions))
	for _, question := range questions {
		rating := q.bank.Rating(question.ID)
		listed = append(listed, &quizzesv1.QuizQuestion{
			Id:             string(question.ID),
			Text:           question.Text,
			Answer:         question.Answer,
			Subject:        question.Subject,
			Prior:          question.Prior,
			Rating:         rating.Level(),
			PredictedShare: rating.Chance(),
			Answers:        uint32(rating.Answers()), //nolint:gosec // a count of answers, never negative.
			Right:          uint32(rating.Right()),   //nolint:gosec // a count of answers, never negative.
			MeasuredShare:  measured(rating),
		})
	}

	return listed
}

func measured(rating ratings.Rating) *float64 {
	if rating.Answers() == 0 {
		return nil
	}
	share := float64(rating.Right()) / float64(rating.Answers())
	return &share
}
