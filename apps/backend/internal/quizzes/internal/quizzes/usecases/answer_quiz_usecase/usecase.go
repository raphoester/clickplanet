package answer_quiz_usecase

import (
	"context"
	"errors"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	quizzesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/quizzes/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/quizzes"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/ratings"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var ErrNoSuchQuiz = errors.New("no quiz to answer")

const (
	tellTimeout  = 5 * time.Second
	countTimeout = 5 * time.Second
)

type Registry interface {
	AnswerQuiz(token string, entrant quizzes.Entrant, choice int) (quizzes.Answered, bool)
	Publish(won quizzes.Won)
}

// Answers counts each answer for its question's rating (ratings' record_answer_usecase).
type Answers interface {
	Execute(ctx context.Context, answer ratings.Answer) error
}

type In struct {
	Caller    quizzes.Caller
	Token     string
	Choice    int
	CountryID string
}

type Out struct {
	Correct bool

	CorrectChoice int
}

func New(
	registry Registry, answers Answers, transactioner cpatomic.Transactioner, outbox cpatomic.Outbox, clock cptime.Clock,
) *UseCase {
	return &UseCase{registry: registry, answers: answers, transactioner: transactioner, outbox: outbox, clock: clock}
}

type UseCase struct {
	registry      Registry
	answers       Answers
	transactioner cpatomic.Transactioner
	outbox        cpatomic.Outbox
	clock         cptime.Clock
}

func (u *UseCase) Execute(ctx context.Context, in In) (Out, error) {
	answered, ok := u.registry.AnswerQuiz(in.Token, quizzes.EntrantOf(in.Caller), in.Choice)
	if !ok {
		return Out{}, ErrNoSuchQuiz
	}

	u.count(ctx, in.Caller.Account, answered)
	if answered.Late {
		return Out{}, ErrNoSuchQuiz
	}

	correct := answered.Outcome == ratings.Right
	if err := u.tell(ctx, in, answered); err != nil {
		return Out{}, err
	}
	if correct {
		u.registry.Publish(quizzes.Won{CountryID: in.CountryID, Subject: answered.Subject})
	}

	return Out{Correct: correct, CorrectChoice: answered.CorrectChoice}, nil
}

// count logs the answer and moves its question's rating, for a caller with no account too. The answer is spent, so a
// caller who leaves now is still counted. A count that fails is logged and counted by its decorators, and the player's
// answer stands without it.
func (u *UseCase) count(ctx context.Context, account quizzes.Account, answered quizzes.Answered) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), countTimeout)
	defer cancel()

	answerer, err := ratings.AccountIDOf(string(account))
	if err != nil {
		answerer = ratings.NoAccount
	}

	_ = u.answers.Execute(ctx, ratings.Answer{
		Question: answered.Question,
		Account:  answerer,
		Outcome:  answered.Outcome,
		After:    answered.After,
		At:       u.clock.Now(),
	})
}

// tell keeps QuizAnswered in the outbox, so a right answer reaches whoever pays it though Redis be away a moment. The
// quiz is spent: a caller who leaves now is still told.
func (u *UseCase) tell(ctx context.Context, in In, answered quizzes.Answered) error {
	if in.Caller.Account == quizzes.NoAccount {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), tellTimeout)
	defer cancel()

	return u.transactioner.InTx(ctx, func(ctx context.Context) error { //nolint:wrapcheck // the outbox names the event.
		return u.outbox.Record(ctx, &quizzesv1.QuizAnswered{
			AccountId:  string(in.Caller.Account),
			Correct:    answered.Outcome == ratings.Right,
			AnsweredAt: timestamppb.New(u.clock.Now()),
			Token:      in.Token,
			Welcome:    answered.Welcome,
		})
	})
}
