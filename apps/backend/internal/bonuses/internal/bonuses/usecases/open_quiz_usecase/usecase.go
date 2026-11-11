package open_quiz_usecase

import (
	"context"
	"errors"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses"
)

var ErrNoSuchQuiz = errors.New("no quiz to open")

type Registry interface {
	OpenQuiz(token string, entrant bonuses.Entrant) (bonuses.Asked, bool)
}

type In struct {
	Caller bonuses.Caller
	Token  string
}

type Out struct {
	Question string

	Options []string

	Deadline time.Time

	Window time.Duration
}

func New(registry Registry) *UseCase {
	return &UseCase{registry: registry}
}

type UseCase struct {
	registry Registry
}

func (u *UseCase) Execute(_ context.Context, in In) (Out, error) {
	asked, ok := u.registry.OpenQuiz(in.Token, bonuses.EntrantOf(in.Caller))
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
