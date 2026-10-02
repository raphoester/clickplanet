// Package answer_quiz_usecase settles a quiz: whether the answer was right, and what it was worth.
package answer_quiz_usecase

import (
	"context"
	"errors"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

// ErrNoSuchQuiz is every way an answer can fail to land, told as one: unknown, already answered,
// never opened, and somebody else's.
//
// A **wrong** answer is not one of them. It succeeds, grants nothing, and says what the right one
// was — see the package's own note in the proto.
var ErrNoSuchQuiz = errors.New("no quiz to answer")

type Registry interface {
	AnswerQuiz(token string, scope string, choice int) (bonuses.Answered, bool)
	Publish(taken bonuses.Taken)
}

// Charger hands over the charge a right answer was worth, exactly as a caught box is handed over.
type Charger interface {
	Grant(holder bonuses.Holder, kind bonuses.Kind, amount int)
	Held(holder bonuses.Holder) bonuses.Held
}

type In struct {
	Token     string
	Choice    int
	CountryID string
}

type Out struct {
	Correct bool

	// Which option was right, whatever was pressed.
	CorrectChoice int

	// Empty unless Correct.
	Kind bonuses.Kind

	// What the charge added, which is less than the draw when a stack or a pool was near its size.
	// Zero unless Correct.
	Amount int

	// What the caller holds now, right or wrong: a wrong answer changes nothing, and saying so is
	// what lets the client settle on one reading of the inventory either way.
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

	answered, ok := u.registry.AnswerQuiz(in.Token, payer.Scope, in.Choice)
	if !ok {
		return Out{}, ErrNoSuchQuiz
	}

	holder := bonuses.HolderOf(payer)
	out := Out{Correct: answered.Correct, CorrectChoice: answered.CorrectChoice, Held: u.charger.Held(holder)}

	if !answered.Correct {
		return out, nil
	}

	before := out.Held
	u.charger.Grant(holder, answered.Reward.Kind, answered.Reward.Amount)
	out.Held = u.charger.Held(holder)
	out.Kind = answered.Reward.Kind
	out.Amount = out.Held.Count(answered.Reward.Kind) - before.Count(answered.Reward.Kind)

	// Only once the charge is held, for the reason a catch is announced only then: a win announced
	// to the planet that then failed to apply is the one lie this could tell.
	u.registry.Publish(bonuses.Taken{
		CountryID:   in.CountryID,
		Kind:        answered.Reward.Kind,
		QuizSubject: answered.Subject,
	})

	return out, nil
}
