package answer_quiz_usecase

import (
	"context"
	"errors"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

var ErrNoSuchQuiz = errors.New("no quiz to answer")

type Registry interface {
	AnswerQuiz(token string, entrant bonuses.Entrant, choice int) (bonuses.Answered, bool)
	Publish(taken bonuses.Taken)
}

type Charger interface {
	Grant(ctx context.Context, holder bonuses.Holder, kind bonuses.Kind, amount int) error
	Held(ctx context.Context, holder bonuses.Holder) (bonuses.Held, error)
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

func New(registry Registry, charger Charger) *UseCase {
	return &UseCase{registry: registry, charger: charger}
}

type UseCase struct {
	registry Registry
	charger  Charger
}

func (u *UseCase) Execute(ctx context.Context, in In) (Out, error) {
	payer := clicks.PayerOf(ctx)

	answered, ok := u.registry.AnswerQuiz(in.Token, bonuses.EntrantOf(payer), in.Choice)
	if !ok {
		return Out{}, ErrNoSuchQuiz
	}

	holder := bonuses.HolderOf(payer)
	before, err := u.charger.Held(ctx, holder)
	if err != nil {
		return Out{}, err
	}
	out := Out{Correct: answered.Correct, CorrectChoice: answered.CorrectChoice, Held: before}

	if !answered.Correct {
		return out, nil
	}

	if err := u.charger.Grant(ctx, holder, answered.Reward.Kind, answered.Reward.Amount); err != nil {
		return Out{}, err
	}
	if out.Held, err = u.charger.Held(ctx, holder); err != nil {
		return Out{}, err
	}
	out.Kind = answered.Reward.Kind
	out.Amount = out.Held.Count(answered.Reward.Kind) - before.Count(answered.Reward.Kind)

	u.registry.Publish(bonuses.Taken{
		CountryID:   in.CountryID,
		Kind:        answered.Reward.Kind,
		QuizSubject: answered.Subject,
	})

	return out, nil
}
