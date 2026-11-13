package record_answer_usecase_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/ratings"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/ratings/usecases/record_answer_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

var at = time.Date(2026, 11, 11, 12, 0, 0, 0, time.UTC)

// store keeps what a transaction wrote; a failed one keeps nothing, as postgres would.
type store struct {
	mu      sync.Mutex
	answers []ratings.Answer
	ratings map[ratings.QuestionID]ratings.Rating
	fail    error
}

func (s *store) SaveAnswer(_ context.Context, answer ratings.Answer) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.fail != nil {
		return s.fail
	}
	s.answers = append(s.answers, answer)
	return nil
}

func (s *store) SaveRating(_ context.Context, question ratings.QuestionID, rating ratings.Rating) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.ratings == nil {
		s.ratings = make(map[ratings.QuestionID]ratings.Rating)
	}
	s.ratings[question] = rating
	return nil
}

// bank answers a question's rating as the bank does: the book's, else its prior.
type bank struct {
	book  *ratings.Book
	prior float64
}

func (b bank) Rating(question ratings.QuestionID) ratings.Rating {
	if rating, ok := b.book.Rated(question); ok {
		return rating
	}
	return ratings.PriorOf(b.prior)
}

func newUseCase(kept *store) (*record_answer_usecase.UseCase, *ratings.Book) {
	book := ratings.NewBook(nil)
	return record_answer_usecase.New(cpatomic.FakeTransactioner{}, kept, bank{book: book, prior: 0.6}, book), book
}

func TestAnAnswerIsLoggedAndMovesItsQuestionFromItsPrior(t *testing.T) {
	kept := &store{}
	useCase, book := newUseCase(kept)
	answer := ratings.Answer{Question: "capital:ee", Outcome: ratings.Right, After: 2 * time.Second, At: at}

	require.NoError(t, useCase.Execute(t.Context(), answer))

	assert.Equal(t, []ratings.Answer{answer}, kept.answers)
	want := ratings.PriorOf(0.6).Answered(ratings.Right)
	assert.Equal(t, want, kept.ratings["capital:ee"])
	rated, ok := book.Rated("capital:ee")
	require.True(t, ok)
	assert.Equal(t, want, rated, "the draw reads what committed")
}

func TestTheNextAnswerMovesTheRatingTheLastOneLeft(t *testing.T) {
	kept := &store{}
	useCase, book := newUseCase(kept)

	require.NoError(t, useCase.Execute(t.Context(), ratings.Answer{Question: "capital:ee", Outcome: ratings.Right, At: at}))
	require.NoError(t, useCase.Execute(t.Context(), ratings.Answer{Question: "capital:ee", Outcome: ratings.Wrong, At: at}))

	rated, _ := book.Rated("capital:ee")
	assert.Equal(t, ratings.PriorOf(0.6).Answered(ratings.Right).Answered(ratings.Wrong), rated)
	assert.Equal(t, 2, rated.Answers())
	assert.Equal(t, 1, rated.Right())
}

func TestAnAnswerThatDoesNotCommitMovesNothingInMemory(t *testing.T) {
	down := errors.New("postgres is down")
	useCase, book := newUseCase(&store{fail: down})

	err := useCase.Execute(t.Context(), ratings.Answer{Question: "capital:ee", Outcome: ratings.Right, At: at})

	require.ErrorIs(t, err, down)
	_, ok := book.Rated("capital:ee")
	assert.False(t, ok, "the copy in memory moves only after the commit")
}

func TestAnswersAtOnceAreEachCounted(t *testing.T) {
	kept := &store{}
	useCase, book := newUseCase(kept)

	var wait sync.WaitGroup
	for i := range 50 {
		wait.Go(func() {
			question := ratings.QuestionID("capital:ee")
			if i%2 == 1 {
				question = "capital:lv"
			}
			assert.NoError(t, useCase.Execute(t.Context(), ratings.Answer{Question: question, Outcome: []ratings.Outcome{ratings.Right, ratings.Wrong, ratings.TimedOut}[i%3], At: at}))
		})
	}
	wait.Wait()

	estonia, _ := book.Rated("capital:ee")
	latvia, _ := book.Rated("capital:lv")
	assert.Equal(t, 25, estonia.Answers(), "no answer moved a question from a rating another had already moved")
	assert.Equal(t, 25, latvia.Answers())
	assert.Equal(t, estonia, kept.ratings["capital:ee"])
	assert.Equal(t, latvia, kept.ratings["capital:lv"])
}
