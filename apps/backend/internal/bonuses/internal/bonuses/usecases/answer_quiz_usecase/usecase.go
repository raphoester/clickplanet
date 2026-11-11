package answer_quiz_usecase

import (
	"context"
	"errors"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	bonusesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/bonuses/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var ErrNoSuchQuiz = errors.New("no quiz to answer")

const grantTimeout = 5 * time.Second

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

func New(registry Registry, wallet Wallet, events Publisher, clock cptime.Clock) *UseCase {
	return &UseCase{registry: registry, wallet: wallet, events: events, clock: clock}
}

type UseCase struct {
	registry Registry
	wallet   Wallet
	events   Publisher
	clock    cptime.Clock
}

func (u *UseCase) Execute(ctx context.Context, in In) (Out, error) {
	answered, ok := u.registry.AnswerQuiz(in.Token, bonuses.EntrantOf(in.Caller), in.Choice)
	if !ok {
		return Out{}, ErrNoSuchQuiz
	}

	account := in.Caller.Account
	if !answered.Correct {
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

func (u *UseCase) publishAnswer(account bonuses.Account, correct bool) {
	if account == bonuses.NoAccount {
		return
	}
	u.events.Publish(&bonusesv1.QuizAnswered{
		AccountId: string(account), Correct: correct, AnsweredAt: timestamppb.New(u.clock.Now()),
	})
}
