package record_answer_usecase

import (
	"context"
	"hash/fnv"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/ratings"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

// Store keeps the answer log and each question's running totals, in the use case's transaction.
type Store interface {
	SaveAnswer(ctx context.Context, answer ratings.Answer) error
	SaveRating(ctx context.Context, question ratings.QuestionID, rating ratings.Rating) error
}

// Ratings is each question's rating now: what its answers moved it to, else its prior. It is the bank's to answer,
// since the bank holds the priors.
type Ratings interface {
	Rating(question ratings.QuestionID) ratings.Rating
}

// Book is the copy in memory the draw reads.
type Book interface {
	Keep(question ratings.QuestionID, rating ratings.Rating)
}

func New(transactioner cpatomic.Transactioner, store Store, rated Ratings, book Book) *UseCase {
	return &UseCase{transactioner: transactioner, store: store, rated: rated, book: book}
}

// UseCase counts one answer: it logs it and moves its question's rating, both in one transaction, and the copy in
// memory only once that committed.
type UseCase struct {
	transactioner cpatomic.Transactioner
	store         Store
	rated         Ratings
	book          Book

	// A question's answers are counted one at a time, so that two at once never both move it from the same rating.
	// Striped, so a slow commit holds up only the questions that share its stripe.
	stripes [stripes]sync.Mutex
}

const stripes = 64

func (u *UseCase) Execute(ctx context.Context, answer ratings.Answer) error {
	stripe := &u.stripes[stripeOf(answer.Question)]
	stripe.Lock()
	defer stripe.Unlock()

	rating := u.rated.Rating(answer.Question).Answered(answer.Outcome)

	if err := u.transactioner.InTx(ctx, func(ctx context.Context) error {
		if err := u.store.SaveAnswer(ctx, answer); err != nil {
			return err //nolint:wrapcheck // the store named it.
		}
		return u.store.SaveRating(ctx, answer.Question, rating) //nolint:wrapcheck // the store named it.
	}); err != nil {
		return err
	}

	u.book.Keep(answer.Question, rating)
	return nil
}

func stripeOf(question ratings.QuestionID) int {
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(question))
	return int(hash.Sum32() % stripes)
}
