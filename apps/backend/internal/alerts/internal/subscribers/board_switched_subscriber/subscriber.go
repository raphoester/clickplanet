package board_switched_subscriber

import (
	"context"
	"fmt"
	"math"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/pings"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/subscribers"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

type UseCase interface {
	Execute(ctx context.Context, start pings.Start) error
}

func New(useCase UseCase) Subscriber {
	return Subscriber{useCase: useCase}
}

type Subscriber struct {
	useCase UseCase
}

var _ cpredis.Handler[*planetv1.BoardSwitched] = Subscriber{}

// Handle tells everybody in the server once that the world grew, as a season's start is told: an arena's board is no
// switch of the world's.
func (s Subscriber) Handle(ctx context.Context, event *planetv1.BoardSwitched) error {
	if event.GetMap() != "world" || event.GetFromTiles() == 0 {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	at := event.GetSwitchedAt().AsTime()
	return s.useCase.Execute(ctx, pings.Start{ //nolint:wrapcheck // the use case named it.
		Key:  pings.StartKey(fmt.Sprintf("world/%s", event.GetTo())),
		Kind: pings.WorldGrew,
		//nolint:gosec // a ratio of two boards' tiles, a few times.
		Times:    uint32(math.Round(float64(event.GetToTiles()) / float64(event.GetFromTiles()))),
		StartsAt: at, EndsAt: at,
	})
}
