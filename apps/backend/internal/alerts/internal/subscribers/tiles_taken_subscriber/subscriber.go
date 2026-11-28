package tiles_taken_subscriber

import (
	"context"
	"fmt"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/nations"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/pings"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/subscribers"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpboards"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

type UseCase interface {
	Execute(ctx context.Context, takes ...pings.Take) error
}

// nations.Catalog.
type Flags interface {
	FlagOf(code string) (nations.Flag, bool)
}

func New(flags Flags, useCase UseCase) Subscriber {
	return Subscriber{flags: flags, useCase: useCase}
}

type Subscriber struct {
	flags   Flags
	useCase UseCase
}

var _ cpredis.BatchHandler[*planetv1.TilesTaken] = Subscriber{}

// The batch's takes are heard together, in order. A blank tile or a flag's own is nobody's loss.
func (s Subscriber) HandleBatch(ctx context.Context, events []*planetv1.TilesTaken) error {
	var takes []pings.Take
	for _, event := range events {
		heard, err := s.takesOf(event)
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

func (s Subscriber) takesOf(event *planetv1.TilesTaken) ([]pings.Take, error) {
	// The tiles' points are their board's, and so is how close a ping's view comes.
	scale, err := cpboards.ScaleOf(event.GetBoard())
	if err != nil {
		return nil, fmt.Errorf("failed to scale the takes to their board: %w", err)
	}
	var takes []pings.Take
	for _, tile := range event.GetTiles() {
		if tile.GetPrevious() == "" || tile.GetPrevious() == event.GetCountry() {
			continue
		}
		// A sold flag whose name the alerts never heard has no role to ping, nor a name to tell.
		victim, ok := s.flags.FlagOf(tile.GetPrevious())
		if !ok {
			continue
		}
		attacker, ok := s.flags.FlagOf(event.GetCountry())
		if !ok {
			continue
		}
		ground, ok := s.flags.FlagOf(tile.GetGround())
		if !ok {
			ground = victim
		}

		point := tile.GetPoint()
		takes = append(takes, pings.Take{
			Victim: victim, Attacker: attacker, Ground: ground, At: event.GetTakenAt().AsTime(),
			Point: pings.Point{X: point.GetX(), Y: point.GetY(), Z: point.GetZ()}, Scale: scale,
		})
	}
	return takes, nil
}
