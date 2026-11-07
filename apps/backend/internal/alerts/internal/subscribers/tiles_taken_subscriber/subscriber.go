package tiles_taken_subscriber

import (
	"context"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/nations"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/pings"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/subscribers"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

type UseCase interface {
	Execute(ctx context.Context, takes ...pings.Take) error
}

func New(useCase UseCase) Subscriber {
	return Subscriber{useCase: useCase}
}

type Subscriber struct {
	useCase UseCase
}

var _ cpredis.BatchHandler[*planetv1.TilesTaken] = Subscriber{}

// The batch's takes are heard together, in order. A blank tile or a flag's own is nobody's loss.
func (s Subscriber) HandleBatch(ctx context.Context, events []*planetv1.TilesTaken) error {
	var takes []pings.Take
	for _, event := range events {
		heard, err := takesOf(event)
		if err != nil {
			return err
		}
		takes = append(takes, heard...)
	}
	if len(takes) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	return s.useCase.Execute(ctx, takes...) //nolint:wrapcheck // the use case named it.
}

func takesOf(event *planetv1.TilesTaken) ([]pings.Take, error) {
	var takes []pings.Take
	for _, tile := range event.GetTiles() {
		if tile.GetPrevious() == "" || tile.GetPrevious() == event.GetCountry() {
			continue
		}
		victim, err := nations.FlagOf(tile.GetPrevious())
		if err != nil {
			return nil, err //nolint:wrapcheck // the flag named it.
		}
		attacker, err := nations.FlagOf(event.GetCountry())
		if err != nil {
			return nil, err //nolint:wrapcheck // the flag named it.
		}
		ground, err := nations.FlagOf(tile.GetGround())
		if err != nil {
			ground = victim
		}

		point := tile.GetPoint()
		takes = append(takes, pings.Take{
			Victim: victim, Attacker: attacker, Ground: ground, At: event.GetTakenAt().AsTime(),
			Point: pings.Point{X: point.GetX(), Y: point.GetY(), Z: point.GetZ()},
		})
	}
	return takes, nil
}
