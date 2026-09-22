// Package open_quiz_usecase reads the question a banner named, and starts its clock.
package open_quiz_usecase

import (
	"context"
	"errors"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

// ErrNoSuchQuiz covers every way an open can fail and tells nobody which: unknown, lapsed, answered
// and somebody else's are one answer, because the difference is what a script guessing tokens would
// measure. The same rule ClaimBonus follows.
var ErrNoSuchQuiz = errors.New("no quiz to open")

type Registry interface {
	OpenQuiz(token string, scope string) (bonuses.Asked, bool)
}

type In struct {
	Token string
}

type Out struct {
	Question string

	// Three of them, in the order they go on the wire. Which is right is not here and never leaves
	// the registry.
	Options []string

	Deadline time.Time

	// The whole window the player was given, so the countdown is drawn against that rather than
	// against what a slow round trip left of it.
	Window time.Duration
}

func New(registry Registry) *UseCase {
	return &UseCase{registry: registry}
}

type UseCase struct {
	registry Registry
}

// Execute derives the scope the way the throttle and the claim do, so the banner, the question and
// the answer are tied to one caller by construction rather than by agreement.
func (u *UseCase) Execute(ctx context.Context, in In) (Out, error) {
	asked, ok := u.registry.OpenQuiz(in.Token, clicks.PayerOf(ctx).Scope)
	if !ok {
		return Out{}, ErrNoSuchQuiz
	}

	return Out{
		Question: asked.Question,
		Options:  asked.Options,
		Deadline: asked.Deadline,
		Window:   asked.Window,
	}, nil
}
