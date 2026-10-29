package click_tried_subscriber

import (
	"context"
	"errors"
	"fmt"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/flagmissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/subscribers"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

type UseCase interface {
	Execute(ctx context.Context, strike flagmissions.Strike) error
}

func New(useCase UseCase) Subscriber {
	return Subscriber{useCase: useCase}
}

type Subscriber struct {
	useCase UseCase
}

var _ cpredis.Handler[*planetv1.ClickTried] = Subscriber{}

var errNoTime = errors.New("the try has no time")

// Only a click a shield took is a shield broken.
func (s Subscriber) Handle(ctx context.Context, event *planetv1.ClickTried) error {
	if event.GetOutcome() != planetv1.ClickOutcome_CLICK_OUTCOME_STRUCK {
		return nil
	}
	if err := event.GetTriedAt().CheckValid(); err != nil {
		return fmt.Errorf("tile %d: %w: %w", event.GetTileId(), errNoTime, err)
	}

	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	return s.useCase.Execute(ctx, flagmissions.Strike{ //nolint:wrapcheck // the use case named it.
		Flag: event.GetCountry(), Owner: event.GetOwner(), At: event.GetTriedAt().AsTime(),
	})
}
