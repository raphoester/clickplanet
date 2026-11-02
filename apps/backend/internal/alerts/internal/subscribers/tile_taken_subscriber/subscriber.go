package tile_taken_subscriber

import (
	"context"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/nations"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/pings"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/subscribers"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

type UseCase interface {
	Execute(ctx context.Context, take pings.Take) error
}

func New(useCase UseCase) Subscriber {
	return Subscriber{useCase: useCase}
}

type Subscriber struct {
	useCase UseCase
}

var _ cpredis.Handler[*planetv1.TileTaken] = Subscriber{}

// A blank tile or a flag's own is nobody's loss.
func (s Subscriber) Handle(ctx context.Context, event *planetv1.TileTaken) error {
	if event.GetPrevious() == "" || event.GetPrevious() == event.GetCountry() {
		return nil
	}
	victim, err := nations.FlagOf(event.GetPrevious())
	if err != nil {
		return err //nolint:wrapcheck // the flag named it.
	}
	attacker, err := nations.FlagOf(event.GetCountry())
	if err != nil {
		return err //nolint:wrapcheck // the flag named it.
	}
	ground, err := nations.FlagOf(event.GetGround())
	if err != nil {
		ground = victim
	}

	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	return s.useCase.Execute(ctx, pings.Take{ //nolint:wrapcheck // the use case named it.
		Victim: victim, Attacker: attacker, Ground: ground, At: event.GetTakenAt().AsTime(),
	})
}
