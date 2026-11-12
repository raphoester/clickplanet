package answer_quiz_usecase

import (
	"context"
	"errors"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	bonusesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/bonuses/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/ratings"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var ErrNoSuchQuiz = errors.New("no quiz to answer")

const (
	grantTimeout = 5 * time.Second
	countTimeout = 5 * time.Second
)

type Registry interface {
	AnswerQuiz(token string, entrant bonuses.Entrant, choice int) (bonuses.Answered, bool)
	Publish(taken bonuses.Taken)
}

// Wallet pays what a right answer grants, and answers the hand a wrong one leaves: it is planet's.
type Wallet interface {
	Pay(ctx context.Context, payment bonuses.Payment) (bonuses.Paid, error)
	Hand(ctx context.Context, account bonuses.Account) (bonuses.Hand, error)
}

type Publisher interface {
	Publish(events ...proto.Message)
}

// Answers counts each answer for its question's rating (ratings' record_answer_usecase).
type Answers interface {
	Execute(ctx context.Context, answer ratings.Answer) error
}

type In struct {
	Caller    bonuses.Caller
	Token     string
	Choice    int
	CountryID string
}

type Out struct {
	Correct bool

	CorrectChoice int

	Kind bonuses.Kind

	// Amount is what the wallet kept.
	Amount int

	Hand bonuses.Hand
}

func New(registry Registry, wallet Wallet, answers Answers, events Publisher, clock cptime.Clock) *UseCase {
	return &UseCase{registry: registry, wallet: wallet, answers: answers, events: events, clock: clock}
}

type UseCase struct {
	registry Registry
	wallet   Wallet
	answers  Answers
	events   Publisher
	clock    cptime.Clock
}

func (u *UseCase) Execute(ctx context.Context, in In) (Out, error) {
	answered, ok := u.registry.AnswerQuiz(in.Token, bonuses.EntrantOf(in.Caller), in.Choice)
	if !ok {
		return Out{}, ErrNoSuchQuiz
	}

	u.count(ctx, in.Caller.Account, answered)
	if answered.Late {
		return Out{}, ErrNoSuchQuiz
	}

	account := in.Caller.Account
	if answered.Outcome != ratings.Right {
		hand, err := u.wallet.Hand(ctx, account)
		if err != nil {
			return Out{}, err //nolint:wrapcheck // the wallet named it.
		}
		u.publishAnswer(account, false)
		return Out{CorrectChoice: answered.CorrectChoice, Hand: hand}, nil
	}

	// The quiz is spent: a caller who leaves now must still get the charge.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), grantTimeout)
	defer cancel()

	paid, err := u.wallet.Pay(ctx, bonuses.Payment{
		Grant:   bonuses.OfferGrantOf(in.Token),
		Account: account,
		Kind:    answered.Reward.Kind,
		Amount:  answered.Reward.Amount,
	})
	if err != nil {
		return Out{}, err //nolint:wrapcheck // the wallet named it.
	}

	u.registry.Publish(bonuses.Taken{
		CountryID:   in.CountryID,
		Kind:        answered.Reward.Kind,
		QuizSubject: answered.Subject,
	})
	u.publishAnswer(account, true)

	return Out{
		Correct:       true,
		CorrectChoice: answered.CorrectChoice,
		Kind:          answered.Reward.Kind,
		Amount:        paid.Kept,
		Hand:          paid.Hand,
	}, nil
}

// count logs the answer and moves its question's rating, for a caller with no account too. The answer is spent, so a
// caller who leaves now is still counted. A count that fails is logged and counted by its decorators, and the player's
// answer stands without it.
func (u *UseCase) count(ctx context.Context, account bonuses.Account, answered bonuses.Answered) {
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

func (u *UseCase) publishAnswer(account bonuses.Account, correct bool) {
	if account == bonuses.NoAccount {
		return
	}
	u.events.Publish(&bonusesv1.QuizAnswered{
		AccountId: string(account), Correct: correct, AnsweredAt: timestamppb.New(u.clock.Now()),
	})
}
