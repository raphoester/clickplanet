package answer_quiz_usecase

import (
	"context"
	"errors"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var ErrNoSuchQuiz = errors.New("no quiz to answer")

const grantTimeout = 5 * time.Second

type Registry interface {
	AnswerQuiz(token string, entrant bonuses.Entrant, choice int) (bonuses.Answered, bool)
	Publish(taken bonuses.Taken)
}

type Charger interface {
	Grant(ctx context.Context, holder bonuses.Holder, kind bonuses.Kind, amount int) (bonuses.Gain, error)
	Held(ctx context.Context, holder bonuses.Holder) (bonuses.Held, error)
}

type Publisher interface {
	Publish(events ...proto.Message)
}

type In struct {
	Token     string
	Choice    int
	CountryID string
}

type Out struct {
	Correct bool

	CorrectChoice int

	Kind bonuses.Kind

	Amount int

	Held bonuses.Held
}

func New(registry Registry, charger Charger, events Publisher, clock cptime.Clock) *UseCase {
	return &UseCase{registry: registry, charger: charger, events: events, clock: clock}
}

type UseCase struct {
	registry Registry
	charger  Charger
	events   Publisher
	clock    cptime.Clock
}

func (u *UseCase) Execute(ctx context.Context, in In) (Out, error) {
	payer := clicks.PayerOf(ctx)

	answered, ok := u.registry.AnswerQuiz(in.Token, bonuses.EntrantOf(payer), in.Choice)
	if !ok {
		return Out{}, ErrNoSuchQuiz
	}

	holder := bonuses.HolderOf(payer)
	if !answered.Correct {
		held, err := u.charger.Held(ctx, holder)
		if err != nil {
			return Out{}, err
		}
		u.publishAnswer(holder, false)
		return Out{CorrectChoice: answered.CorrectChoice, Held: held}, nil
	}

	// The quiz is spent: a caller who leaves now must still get the charge.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), grantTimeout)
	defer cancel()

	gain, err := u.charger.Grant(ctx, holder, answered.Reward.Kind, answered.Reward.Amount)
	if err != nil {
		return Out{}, err
	}

	u.registry.Publish(bonuses.Taken{
		CountryID:   in.CountryID,
		Kind:        answered.Reward.Kind,
		QuizSubject: answered.Subject,
	})
	u.publishAnswer(holder, true)

	return Out{
		Correct:       true,
		CorrectChoice: answered.CorrectChoice,
		Kind:          answered.Reward.Kind,
		Amount:        gain.Count(answered.Reward.Kind),
		Held:          gain.After,
	}, nil
}

func (u *UseCase) publishAnswer(holder bonuses.Holder, correct bool) {
	if holder == bonuses.NoHolder {
		return
	}
	u.events.Publish(&planetv1.QuizAnswered{
		AccountId: string(holder), Correct: correct, AnsweredAt: timestamppb.New(u.clock.Now()),
	})
}
